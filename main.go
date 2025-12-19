package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"whatsabladerunner/pkg/agent"
	"whatsabladerunner/pkg/history"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/workflows"
)

var (
	ollamaClient   *ollama.Client
	convManager    *agent.ConversationManager
	whatsAppClient *whatsmeow.Client
	historyStore   *history.HistoryStore
)

const BotPrefix = "[Blady] : "

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("------------------------------------------------")
		fmt.Printf("Received a message!\n")
		//fmt.Printf("ID: %s\n", v.Info.ID)
		fmt.Printf("Time: %s\n", v.Info.Timestamp)
		fmt.Printf("Sender: %s (PushName: %s)\n", v.Info.Sender, v.Info.PushName)
		fmt.Printf("Chat: %s\n", v.Info.Chat)

		// Save message to history
		// We want to save ALL text messages to history to have a full log.
		// Extract text again or reuse if possible.
		// Refactoring slightly to extract text earlier for both saving and workflow.

		msgText := ""
		if v.Message != nil {
			if v.Message.ExtendedTextMessage != nil {
				msgText = *v.Message.ExtendedTextMessage.Text
			} else if v.Message.Conversation != nil {
				msgText = *v.Message.Conversation
			}
		}

		if msgText != "" {
			err := historyStore.SaveMessage(v.Info.ID, v.Info.Chat.String(), v.Info.Sender.String(), msgText, v.Info.Timestamp, v.Info.IsFromMe)
			if err != nil {
				fmt.Printf("Failed to save message to history: %v\n", err)
			}
		}

		// Check if the message is sent to self (Note to Self)
		if v.Info.IsFromMe && v.Info.Chat.User == v.Info.Sender.User {
			fmt.Println("it's you - triggering workflow")

			fmt.Printf("DEBUG: Message Struct: %+v\n", v.Message)
			if msgText != "" {
				// Ignore messages starting with BotPrefix
				if len(msgText) >= len(BotPrefix) && msgText[:len(BotPrefix)] == BotPrefix {
					fmt.Println("Ignoring bot message")
					return
				}

				fmt.Printf("DEBUG: Extracted text: %s\n", msgText)
				// Start workflow in background, managed by ConversationManager
				chatID := v.Info.Chat.String()
				convManager.StartWorkflow(chatID, func(ctx context.Context) {
					sendFunc := func(msg string) {
						if whatsAppClient != nil {
							resp, err := whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
								Conversation: proto.String(msg),
							})
							if err != nil {
								fmt.Printf("Failed to send message: %v\n", err)
							} else {
								// START FIX: Save bot response to history
								// We need to save the bot's response as well so it appears in future context.
								// It is "FromMe" in the sense that the bot sends it on my behalf?
								// Technically yes, we send it via 'SendMessage'.
								// So 'isFromMe' = true.
								// We should probably construct it safely.
								// Note: The timestamp will be 'now'.
								// Use resp.ID for uniqueness.
								err := historyStore.SaveMessage(resp.ID, v.Info.Chat.String(), v.Info.Sender.String(), msg, time.Now(), true)
								if err != nil {
									fmt.Printf("Failed to save bot response to history: %v\n", err)
								}
								// END FIX
							}
						}
					}

					// Fetch context (last 9 messages)
					contextMsgs, err := historyStore.GetRecentMessages(chatID, 9)
					if err != nil {
						fmt.Printf("Failed to get recent messages: %v\n", err)
						contextMsgs = []string{} // Fallback
					}

					fmt.Printf("DEBUG: Context Messages: %v\n", contextMsgs)

					sendMasterFunc := func(msg string) {
						// For "Note to Self" (User == Sender.User), message_master is just a response back to the chat.
						// But specifically, we can ensure it goes to the user.
						// In this context, v.Info.Chat is the user's chat.
						if whatsAppClient != nil {
							// We can prefix it to distinguish slightly if we want, or rely on normal sending.
							// The SendFunc above prefixes with '[Blady] : '.
							// If 'message_master' is meant to be private, it's already private in "Note to Self".
							// If we are in a group, this should go to a private chat with the user.
							// But v.Info.Chat might be a group.
							// So we should construct a JID for the user.

							// v.Info.Sender is the JID of the sender.
							// If v.Info.IsFromMe, Sender is us. Wait.
							// If I (me) send a note to self, v.Info.Sender is My JID. v.Info.Chat is My JID (as User).
							// So targeting v.Info.Sender is correct for replying to the "Master".

							targetJID := v.Info.Sender

							_, err := whatsAppClient.SendMessage(context.Background(), targetJID, &waProto.Message{
								Conversation: proto.String(msg),
							})
							if err != nil {
								fmt.Printf("Failed to send master message: %v\n", err)
							}
						}
					}

					wf := workflows.NewCommandWorkflow(ollamaClient, sendFunc, sendMasterFunc)
					wf.Run(ctx, msgText, contextMsgs)
				})
			} else {
				fmt.Println("DEBUG: No text found in message")
			}
		} else {
			if v.Message != nil && v.Message.ExtendedTextMessage != nil {
				fmt.Printf("Content: %+v\n", v.Message.ExtendedTextMessage.Text)
			}
		}
		fmt.Println("------------------------------------------------")
	case *events.HistorySync:
		// Handle history sync to backfill messages
		id := v.Data.GetSyncType()
		if id == waHistorySync.HistorySync_FULL || id == waHistorySync.HistorySync_RECENT {
			fmt.Printf("Received History Sync (Type: %s)\n", id)

			for _, conv := range v.Data.GetConversations() {
				chatJID := conv.GetID()
				for _, msg := range conv.GetMessages() {
					// history sync messages are wrapped in WebMessageInfo, we need to extract the actual message content
					// The structure is slightly different or dependent on how whatsmeow exposes it.
					// Actually conv.GetMessages() returns []*waHistorySync.HistorySyncMsg
					// which has a Message field of type *waE2E.Message

					// We need to determine sender. In history sync, it might NOT be explicitly provided in the same way
					// as live events if it's a 1:1 chat (it's the other person) or group.
					// But usually, we can infer.

					// Extract timestamp
					// `msg` is HistorySyncMsg. It wraps WebMessageInfo.
					// WebMessageInfo has MessageTimestamp.
					ts := time.Unix(int64(msg.GetMessage().GetMessageTimestamp()), 0)

					// Extract Sender
					// Is it FromMe?
					isFromMe := msg.GetMessage().GetKey().GetFromMe()

					// Sender JID
					// If it's a group, we need the participant.
					// If it's 1:1, it's either us or them.
					// For simple "Note to Self", chatJID == senderJID (if from them) or chatJID == MyJID (if from me).
					// Let's rely on Key.Participant if available, or ChatJID/Key.RemoteJid
					var senderJID string
					if isFromMe {
						// It's me. We can't easily get my own JID here without the client store,
						// but we can mark it as 'Me' in the store and use a placeholder or
						// try to parse the Key if needed.
						// For now, let's just use "Me" or empty for sender_jid if FromMe is true.
						senderJID = "Me"
					} else {
						// It's them.
						// If Key.Participant is present (groups), use it.
						// Else use ChatJID.
						if msg.GetMessage().GetKey().GetParticipant() != "" {
							senderJID = msg.GetMessage().GetKey().GetParticipant()
						} else {
							senderJID = chatJID
						}
					}

					// Extract Text
					waMsg := msg.GetMessage().GetMessage() // This is *waE2E.Message
					text := ""
					if waMsg != nil {
						if waMsg.ExtendedTextMessage != nil {
							text = *waMsg.ExtendedTextMessage.Text
						} else if waMsg.Conversation != nil {
							text = *waMsg.Conversation
						}
					}

					if text != "" {
						msgID := msg.GetMessage().GetKey().GetID()
						err := historyStore.SaveMessage(msgID, chatJID, senderJID, text, ts, isFromMe)
						if err != nil {
							fmt.Printf("Failed to save history sync message: %v\n", err)
						}
					}
				}
			}
			fmt.Println("History Sync processing complete.")
		}
	}
}

func main() {
	// |------------------------------------------------------------------------------------------------------|
	// | NOTE: You must also import the appropriate DB connector, e.g. github.com/mattn/go-sqlite3 for SQLite |
	// |------------------------------------------------------------------------------------------------------|

	dbLog := waLog.Stdout("Database", "INFO", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:blady_whatsmeow.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	whatsAppClient = whatsmeow.NewClient(deviceStore, clientLog)
	client := whatsAppClient

	// Initialize custom services
	ollamaClient = ollama.NewClient("http://localhost:11434", "qwen3:8b")
	convManager = agent.NewConversationManager()

	historyStore, err = history.New("history.db")
	if err != nil {
		panic(err)
	}

	client.AddEventHandler(eventHandler)

	// Configure Identification
	// deviceStore.Platform = "chrome" // Deprecated or handled by payload? Best to set it for completeness if impactful.
	// Actually, the Platform field in deviceStore might be "stored" but the payload matters more for the session.
	// Let's rely on the payload override.

	// Configure Identification
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	store.DeviceProps.Os = proto.String("Windows")
	// SetOSInfo updates BaseClientPayload.UserAgent.OsVersion and OsBuildNumber
	store.SetOSInfo("Windows", [3]uint32{10, 0, 19045})

	// Customize other payload fields
	store.BaseClientPayload.UserAgent.Manufacturer = proto.String("Microsoft")
	store.BaseClientPayload.UserAgent.Device = proto.String("Windows")

	if client.Store.ID == nil {
		// No ID stored, new login
		fmt.Println("No ID stored, new login")
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// List stored conversations after a short delay to allow connection/store readiness
	go func() {
		time.Sleep(3 * time.Second)
		listStoredConversations(client)
	}()

	<-c

	client.Disconnect()
}

func listStoredConversations(client *whatsmeow.Client) {
	fmt.Println("Fetching stored conversations...")

	// List Groups
	groups, err := client.GetJoinedGroups(context.Background())
	if err != nil {
		fmt.Println("Failed to get groups:", err)
	} else {
		fmt.Println("Groups:")
		for _, g := range groups {
			fmt.Printf("- %s (%s)\n", g.Name, g.JID)
			fmt.Println("  Participants:")
			for _, p := range g.Participants {
				fmt.Printf("    - %s\n", p.JID)
			}

			// Check chat settings
			if client.Store.ChatSettings != nil {
				settings, err := client.Store.ChatSettings.GetChatSettings(context.Background(), g.JID)
				if err == nil {
					if settings.MutedUntil.After(time.Now()) {
						fmt.Println("  [MUTED]")
					}
					if settings.Pinned {
						fmt.Println("  [PINNED]")
					}
					if settings.Archived {
						fmt.Println("  [ARCHIVED]")
					}
				}
			}
		}
	}

	// List Contacts
	if client.Store.Contacts != nil {
		contacts, err := client.Store.Contacts.GetAllContacts(context.Background())
		if err != nil {
			fmt.Println("Failed to get contacts:", err)
		} else {
			fmt.Println("Contacts:")
			for jid, info := range contacts {
				name := info.PushName
				if name == "" {
					name = info.FullName
				}
				if name == "" {
					name = info.BusinessName
				}
				if name == "" {
					name = "Unknown"
				}
				fmt.Printf("- %s (%s)\n", name, jid)

				// Check chat settings
				if client.Store.ChatSettings != nil {
					settings, err := client.Store.ChatSettings.GetChatSettings(context.Background(), jid)
					if err == nil {
						if settings.MutedUntil.After(time.Now()) {
							fmt.Println("  [MUTED]")
						}
						if settings.Pinned {
							fmt.Println("  [PINNED]")
						}
						if settings.Archived {
							fmt.Println("  [ARCHIVED]")
						}
					}
				}
			}
		}
	} else {
		fmt.Println("Contact store not available")
	}
}
