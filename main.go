package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"whatsabladerunner/pkg/agent"
	"whatsabladerunner/pkg/batata"
	"whatsabladerunner/pkg/bot"
	"whatsabladerunner/pkg/buttons"
	"whatsabladerunner/pkg/cerebras"
	"whatsabladerunner/pkg/history"
	"whatsabladerunner/pkg/llm"
	"whatsabladerunner/pkg/locks"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/pkg/tasks"
	"whatsabladerunner/workflows"
)

var (
	batataKernel   *batata.Kernel
	llmClient      llm.Client
	convManager    *agent.ConversationManager
	whatsAppClient *whatsmeow.Client
	historyStore   *history.HistoryStore
	taskBot        *bot.Bot // Global bot instance for task handling
	taskLocks      *locks.KeyedMutex
)

// buttonManager handles interactive message context and responses
var buttonManager *buttons.Manager

// WithheldMessage stores a blocked message for potential "LET IT BE" override
type WithheldMessage struct {
	Message       string       // The blocked message text
	TargetChatJID types.JID    // Where to send if unblocked
	SendFunc      func(string) // Function to use for sending
}

// lastWithheldMessage stores the most recent watcher-blocked message
var lastWithheldMessage *WithheldMessage

const BotPrefix = "[Blady] : "

func downloadMedia(msg *events.Message) ([]byte, string, string, error) {
	var (
		data  []byte
		err   error
		ext   string
		mtype string
	)

	if msg.Message.ImageMessage != nil {
		data, err = whatsAppClient.Download(context.Background(), msg.Message.ImageMessage)
		ext = "jpg"
		mtype = "image"
	} else if msg.Message.VideoMessage != nil {
		data, err = whatsAppClient.Download(context.Background(), msg.Message.VideoMessage)
		ext = "mp4"
		mtype = "video"
	} else if msg.Message.AudioMessage != nil {
		data, err = whatsAppClient.Download(context.Background(), msg.Message.AudioMessage)
		ext = "ogg"
		if msg.Message.AudioMessage.GetPTT() {
			ext = "ogg"
		}
		mtype = "audio"
	} else if msg.Message.DocumentMessage != nil {
		data, err = whatsAppClient.Download(context.Background(), msg.Message.DocumentMessage)
		mtype = "docs"
		// Try to get extension from mimetype or filename
		ext = "bin"
		if msg.Message.DocumentMessage.Mimetype != nil {
			parts := strings.Split(*msg.Message.DocumentMessage.Mimetype, "/")
			if len(parts) > 1 {
				ext = parts[1]
			}
		}
	} else {
		return nil, "", "", fmt.Errorf("unsupported media type")
	}

	if err != nil {
		return nil, "", "", err
	}
	return data, mtype, ext, nil
}

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("------------------------------------------------")
		fmt.Printf("Received a message!\n")
		fmt.Printf("  ID: %s\n", v.Info.ID)
		fmt.Printf("  Time: %s\n", v.Info.Timestamp)
		fmt.Printf("  Sender: %s\n", v.Info.Sender)
		fmt.Printf("  PushName: %s\n", v.Info.PushName)
		fmt.Printf("  Chat: %s\n", v.Info.Chat)
		fmt.Printf("  IsFromMe: %v\n", v.Info.IsFromMe)
		fmt.Printf("  IsGroup: %v\n", v.Info.IsGroup)
		fmt.Printf("  MessageSource: %+v\n", v.Info.MessageSource)
		fmt.Printf("DEBUG: Message Struct: %+v\n", v.Message)

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
			} else if v.Message.ButtonsMessage != nil {
				// Handle buttons message - extract content text and button options with JSON IDs
				bm := v.Message.ButtonsMessage
				if bm.ContentText != nil {
					msgText = *bm.ContentText
				}
				// Append button options with both display text and button ID JSON
				if len(bm.Buttons) > 0 {
					msgText += "\n\n[Opciones de respuesta - responder con el buttonID JSON]:"
					for _, btn := range bm.Buttons {
						displayText := ""
						buttonID := ""
						if btn.ButtonText != nil && btn.ButtonText.DisplayText != nil {
							displayText = *btn.ButtonText.DisplayText
						}
						if btn.ButtonID != nil {
							buttonID = *btn.ButtonID
						}
						msgText += fmt.Sprintf("\n- \"%s\" -> buttonID: %s", displayText, buttonID)
					}
				}
				// Store buttons context for later response
				buttonManager.Store(v.Info.Chat.String(), &buttons.ButtonsContext{
					MessageID: v.Info.ID,
					ChatJID:   v.Info.Chat,
					SenderJID: v.Info.Sender,
					SenderAlt: v.Info.MessageSource.SenderAlt,
					Message:   v.Message,
				})
				fmt.Printf("[ButtonsContext] Stored buttons message ID=%s from chat=%s sender=%s senderAlt=%s\n",
					v.Info.ID, v.Info.Chat, v.Info.Sender, v.Info.MessageSource.SenderAlt)
			} else if v.Message.ListMessage != nil {
				// Handle list message (dropdown options)
				lm := v.Message.ListMessage
				if lm.Description != nil {
					msgText = *lm.Description
				}
				// Extract list sections and rows
				if len(lm.Sections) > 0 {
					msgText += "\n\n[Opciones de lista - responder con el rowID JSON]:"
					for _, section := range lm.Sections {
						for _, row := range section.Rows {
							title := ""
							rowID := ""
							if row.Title != nil {
								title = *row.Title
							}
							if row.RowID != nil {
								rowID = *row.RowID
							}
							msgText += fmt.Sprintf("\n- \"%s\" -> rowID: %s", title, rowID)
						}
					}
				}
				// Store list context for later response
				buttonManager.Store(v.Info.Chat.String(), &buttons.ButtonsContext{
					MessageID: v.Info.ID,
					ChatJID:   v.Info.Chat,
					SenderJID: v.Info.Sender,
					SenderAlt: v.Info.MessageSource.SenderAlt,
					Message:   v.Message,
				})
				fmt.Printf("[ButtonsContext] Stored list message ID=%s from chat=%s sender=%s senderAlt=%s\n",
					v.Info.ID, v.Info.Chat, v.Info.Sender, v.Info.MessageSource.SenderAlt)
			}
		}

		// Media Handling
		isSelfChat := v.Info.IsFromMe && v.Info.Chat.User == v.Info.Sender.User
		var activeTask *tasks.Task
		if !v.Info.IsFromMe {
			activeTask, _ = taskBot.TaskManager.GetTaskByContact(v.Info.Chat.String())
		}

		if (isSelfChat || activeTask != nil) && v.Message != nil {
			// Check if it's a media message
			isMedia := v.Message.ImageMessage != nil || v.Message.VideoMessage != nil || v.Message.AudioMessage != nil || v.Message.DocumentMessage != nil
			if isMedia {
				fmt.Printf("Processing media message (Self: %v, Task: %v)\n", isSelfChat, activeTask != nil)
				data, mtype, ext, err := downloadMedia(v)
				if err != nil {
					fmt.Printf("Failed to download media: %v\n", err)
				} else {
					// Save metadata to DB
					var mimetype string
					if v.Message.ImageMessage != nil {
						mimetype = v.Message.ImageMessage.GetMimetype()
					} else if v.Message.VideoMessage != nil {
						mimetype = v.Message.VideoMessage.GetMimetype()
					} else if v.Message.AudioMessage != nil {
						mimetype = v.Message.AudioMessage.GetMimetype()
					} else if v.Message.DocumentMessage != nil {
						mimetype = v.Message.DocumentMessage.GetMimetype()
					}

					mediaID, err := historyStore.SaveMedia(history.MediaInfo{
						MessageID: v.Info.ID,
						ChatJID:   v.Info.Chat.String(),
						SenderJID: v.Info.Sender.String(),
						MediaType: mtype,
						MimeType:  mimetype,
						Timestamp: v.Info.Timestamp,
					})

					if err != nil {
						fmt.Printf("Failed to save media metadata: %v\n", err)
					} else {
						// Store file
						dir := filepath.Join("plain_media", mtype)
						os.MkdirAll(dir, 0755)
						filePath := filepath.Join(dir, fmt.Sprintf("%d.%s", mediaID, ext))
						err = os.WriteFile(filePath, data, 0644)
						if err != nil {
							fmt.Printf("Failed to save media file: %v\n", err)
						} else {
							fmt.Printf("Media stored at %s\n", filePath)

							// Update msgText so it's saved in history for LLM
							mediaRef := fmt.Sprintf("[Media: %s ID: %d]", mtype, mediaID)
							if msgText != "" {
								msgText += "\n" + mediaRef
							} else {
								msgText = mediaRef
							}

							// Report to user in self-chat
							if whatsAppClient != nil && whatsAppClient.Store.ID != nil {
								selfJID := whatsAppClient.Store.ID.ToNonAD()
								replyFunc := func(msg string) {
									whatsAppClient.SendMessage(context.Background(), selfJID, &waProto.Message{
										Conversation: proto.String(msg),
									})
								}
								batataKernel.ReportMediaStored(mtype, mediaID, replyFunc)
							}
						}
					}
				}
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

				// Handle "LET IT BE" override for watcher-blocked messages
				if msgText == "LET IT BE" {
					if lastWithheldMessage != nil {
						fmt.Println("[LET IT BE] Overriding watcher block - sending withheld message")
						lastWithheldMessage.SendFunc(lastWithheldMessage.Message)
						lastWithheldMessage = nil
						// Send confirmation
						if whatsAppClient != nil {
							whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
								Conversation: proto.String("[Blady][Watcher] : Message sent."),
							})
						}
					} else {
						fmt.Println("[LET IT BE] No withheld message to send")
						if whatsAppClient != nil {
							whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
								Conversation: proto.String("[Blady][Watcher] : No blocked message to release."),
							})
						}
					}
					return
				}

				fmt.Printf("DEBUG: Extracted text: %s\n", msgText)
				// Start workflow in background, managed by ConversationManager
				chatID := v.Info.Chat.String()

				// BATATA INTERCEPTION
				// Self-chat messages are passed to Batata first.
				// We need a reply function.
				replyFunc := func(msg string) {
					if whatsAppClient != nil {
						_, err := whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
							Conversation: proto.String(msg),
						})
						if err != nil {
							fmt.Printf("Failed to send Batata message: %v\n", err)
						}
					}
				}
				killFunc := func() {
					fmt.Println("Batata requested kill. Exiting...")
					os.Exit(0)
				}

				if batataKernel.HandleMessage(msgText, v.Info.Sender, v.Info.Chat, replyFunc, killFunc) {
					// Message handled by Batata, refresh LLM if config changed or just return
					// Ideally we check if config changed, but for now we can just lazily reload or rely on explicit actions.
					// If the user changed the brain, we should update the llmClient and taskBot.
					// We can do this by checking if the state returned to Idle from a change state, or just re-init always on "Back to Blady".
					// But HandleMessage returns true even for intermediate steps.
					// Let's just return for now.
					// If the user *just* finished config (HandleMessage returned true, but State went to Idle),
					// we might want to ensure 'llmClient' is up to date for the next message.
					// We can re-init LLM client here if needed?
					// It's safer to re-init on demand or check a "Dirty" flag.
					// For simplicity, we'll re-init LLM client if Batata is IDLE after handling (meaning it exited a menu).
					if batataKernel.State == batata.StateIdle {
						reinitLLM(v.Info.Chat)
					}
					return
				}

				// Normal Note-to-Self Workflow
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

					wf := workflows.NewCommandWorkflow(llmClient, sendFunc, sendMasterFunc, getAllContactsJSON(whatsAppClient), taskBot.StartTaskCallback)
					wf.Run(ctx, msgText, contextMsgs)
				})
			} else {
				fmt.Println("DEBUG: No text found in message")
			}
		} else {
			// Not a self-message - always ignore messages from me in other chats to avoid echo
			if v.Info.IsFromMe {
				fmt.Println("Ignoring my own message in non-self conversation")
				return
			}

			// Check if there's an active task for this chat
			chatJID := v.Info.Chat.String()

			if msgText != "" && taskBot != nil {
				// Check by both chat ID and contact for task matching
				fmt.Printf("DEBUG: Checking for task with chat ID: %s\n", chatJID)
				task, err := taskBot.TaskManager.GetTaskByContact(chatJID)
				if err != nil {
					fmt.Printf("Error checking for task: %v\n", err)
				} else if task != nil {
					// Found active task - route incoming message from contact to task mode
					fmt.Printf("Active task %d found for chat %s - routing to task mode\n", task.ID, chatJID)

					// Update ChatID if it changed (e.g., bot responded from different JID)
					if task.ChatID != chatJID {
						if err := taskBot.TaskManager.SetTaskChatID(task.ID, chatJID); err != nil {
							fmt.Printf("Failed to update task chat ID: %v\n", err)
						}
					}

					// Route to task mode with DEBOUNCE and LOCKING
					// 1. Wait 5s
					// 2. Lock task
					// 3. Fetch new messages
					// 4. Process
					go func(tID int, cJID string) {
						// 1. Delay
						// fmt.Printf("Task %d: Waiting 5s before processing...\n", tID)
						time.Sleep(5 * time.Second)

						// 2. Lock
						lockKey := fmt.Sprintf("task:%d", tID)
						taskLocks.Lock(lockKey)
						defer taskLocks.Unlock(lockKey)

						// fmt.Printf("Task %d: Acquired lock, processing...\n", tID)

						// 3. Reload Task to get latest timestamp safely
						// We need to reload because another goroutine might have updated it
						currentTask, err := taskBot.TaskManager.LoadTask(tID)
						if err != nil {
							fmt.Printf("Failed to reload task %d: %v\n", tID, err)
							return
						}

						// 4. Fetch new messages since last processed
						// If timestamp is 0, it gets recent context. If > 0, it gets strictly new ones.
						newMsgs, maxUnix, err := historyStore.GetMessagesSince(cJID, currentTask.LastProcessedTimestamp)
						if err != nil {
							fmt.Printf("Failed to get new messages for task %d: %v\n", tID, err)
							return
						}

						if len(newMsgs) == 0 {
							// No new messages found?
							// This can happen if multiple messages came in within 5s.
							// The first one wakes up after 5s, processes ALL of them (updating timestamp).
							// The second one wakes up, sees timestamp is now moved forward, finds 0 new messages.
							// So we just quit.
							// fmt.Printf("Task %d: No new messages found since %d. Skipping.\n", tID, currentTask.LastProcessedTimestamp)
							return
						}

						fmt.Printf("Task %d: Processing %d new messages...\n", tID, len(newMsgs))

						// contextMsgs for the LLM can be just the new ones, OR we provide some history + new ones.
						// The bot.ProcessTask uses 'context []string' and 'msg string'.
						// The 'msg' is usually the single prompt.
						// Now 'msg' should probably be the BLOCK of new messages.
						// And 'context' could be empty if we rely on the block, OR we still provide older history?
						// Bot logic: "Context: ... \n Message: ..."
						// If we pass many messages in Message, it works fine.
						// Let's pass the joined new messages as 'msg'.
						// And for 'context', maybe we still want *previous* context?
						// It's safer to get some *older* history as context if the new block is small.
						// But 'GetMessagesSince' returns the *full* text of the messages.
						// The LLM treats 'context' as history.
						// If we are incrementally processing, the LLM maintains state via 'memories'.
						// But short-term conversational context is 'context'.
						// If we don't pass 'context', it might lose track of the immediately preceding turn if it wasn't in the new block.
						// Let's fetch the *last 10 messages* overall as context, just in case.
						// BUT we must be careful not to duplicate what's in 'msg'.
						// Actually, simplicity: Just pass the new messages block as 'msg'.
						// The LLM should be able to handle "Here are new messages: [A, B, C]".
						// Memories handle long term.

						combinedMsg := strings.Join(newMsgs, "\n")

						// Send function for task conversation (no [Blady] prefix)
						sendToContact := func(msg string) {
							if whatsAppClient != nil {
								resp, err := whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
									Conversation: proto.String(msg),
								})
								if err != nil {
									fmt.Printf("Failed to send task message: %v\n", err)
								} else {
									err := historyStore.SaveMessage(resp.ID, cJID, "Me", msg, time.Now(), true)
									if err != nil {
										fmt.Printf("Failed to save task response to history: %v\n", err)
									}
								}
							}
						}

						// Setup button response for this execution
						chatJIDForContext := v.Info.Chat.String()
						taskBot.SendButtonResponseFunc = func(displayText, buttonID string) {
							if whatsAppClient != nil {
								if buttonID == "" {
									if rd, rb, found := buttonManager.Resolve(chatJIDForContext, displayText); found {
										displayText = rd
										buttonID = rb
									}
								}
								msgID, err := buttonManager.SendResponse(context.Background(), whatsAppClient, chatJIDForContext, displayText, buttonID)
								if err != nil {
									fmt.Printf("[ButtonResponse] Failed (Task), falling back to text: %v\n", err)
									whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{Conversation: proto.String(displayText)})
								} else {
									historyStore.SaveMessage(msgID, cJID, "Me", displayText, time.Now(), true)
								}
							}
						}

						// Process
						_, err = taskBot.ProcessTask(currentTask, combinedMsg, []string{}, sendToContact)
						if err != nil {
							fmt.Printf("Task processing failed: %v\n", err)
						}

						// 5. Update timestamp
						if err := taskBot.TaskManager.SetTaskProcessedTimestamp(tID, maxUnix); err != nil {
							fmt.Printf("Failed to update task timestamp: %v\n", err)
						}

					}(task.ID, chatJID)

				}
			}

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
	clientLog := waLog.Stdout("Client", "INFO", true)
	whatsAppClient = whatsmeow.NewClient(deviceStore, clientLog)
	client := whatsAppClient

	// Initialize LLM client based on provider selection
	// Initialize Batata
	batataKernel = batata.NewKernel("config")
	if err := batataKernel.Load(); err != nil {
		fmt.Println("No Batata config found (or error), starting in Setup Mode.")
		// We will trigger setup after connection
	}

	// Initialize LLM client based on Batata Config
	reinitLLM(types.EmptyJID)
	convManager = agent.NewConversationManager()
	taskLocks = locks.New()
	buttonManager = buttons.NewManager()

	historyStore, err = history.New("history.db")
	if err != nil {
		panic(err)
	}

	// Initialize task bot with StartTaskCallback
	// This callback is triggered when a task is confirmed via confirm_task action
	// SendMasterFunc sends to self-chat (Note to Self) - uses closure over whatsAppClient
	// Note: The bot itself adds [Blady][Task {ID}] or [Blady][Watcher] prefix as appropriate
	sendMasterFromTask := func(msg string) {
		if whatsAppClient != nil && whatsAppClient.Store.ID != nil {
			// Get own JID for self-chat
			ownJID := whatsAppClient.Store.ID.ToNonAD()
			_, err := whatsAppClient.SendMessage(context.Background(), ownJID, &waProto.Message{
				Conversation: proto.String(msg),
			})
			if err != nil {
				fmt.Printf("Failed to send master message from task: %v\n", err)
			}
		}
	}
	taskBot = bot.NewBot(llmClient, "config", nil, sendMasterFromTask, getAllContactsJSON(client))

	// Set up OnWatcherBlock callback to store withheld messages for LET IT BE override
	taskBot.OnWatcherBlock = func(blockedMsg string, targetChatID string, sendFunc func(string)) {
		lastWithheldMessage = &WithheldMessage{
			Message:  blockedMsg,
			SendFunc: sendFunc,
		}
		// Parse target JID to store for potential later use
		if targetJID, err := types.ParseJID(targetChatID); err == nil {
			lastWithheldMessage.TargetChatJID = targetJID
		}
		fmt.Printf("[Watcher] Stored withheld message for LET IT BE override: %s\n", blockedMsg)
	}

	taskBot.StartTaskCallback = func(task *tasks.Task) {
		fmt.Printf("[TaskManager] Starting task %d for contact %s\n", task.ID, task.Contact)

		// Parse contact JID
		contactJID, err := types.ParseJID(task.Contact)
		if err != nil {
			fmt.Printf("Failed to parse contact JID %s: %v\n", task.Contact, err)
			return
		}

		// Set ChatID to the contact JID initially (may be updated if bot responds from different JID)
		if err := taskBot.TaskManager.SetTaskChatID(task.ID, task.Contact); err != nil {
			fmt.Printf("Failed to set task chat ID: %v\n", err)
		}

		// Get conversation context for the task contact
		contextMsgs, err := historyStore.GetRecentMessages(task.Contact, 9)
		if err != nil {
			fmt.Printf("Failed to get context for task contact: %v\n", err)
			contextMsgs = []string{}
		}

		// Create send function for this contact
		sendToContact := func(msg string) {
			if whatsAppClient != nil {
				resp, err := whatsAppClient.SendMessage(context.Background(), contactJID, &waProto.Message{
					Conversation: proto.String(msg),
				})
				if err != nil {
					fmt.Printf("Failed to send task message: %v\n", err)
				} else {
					err := historyStore.SaveMessage(resp.ID, task.Contact, "Me", msg, time.Now(), true)
					if err != nil {
						fmt.Printf("Failed to save task response to history: %v\n", err)
					}
				}
			}
		}

		// Create button response function for this contact
		// Uses stored buttons context for full quotedMessage support
		contactChatID := task.Contact
		sendButtonResponse := func(displayText, buttonID string) {
			if whatsAppClient != nil {
				if buttonID == "" {
					if rd, rb, found := buttonManager.Resolve(contactChatID, displayText); found {
						displayText = rd
						buttonID = rb
					}
				}
				msgID, err := buttonManager.SendResponse(context.Background(), whatsAppClient, contactChatID, displayText, buttonID)
				if err != nil {
					fmt.Printf("[ButtonResponse] Failed (Task), falling back to text: %v\n", err)
					whatsAppClient.SendMessage(context.Background(), contactJID, &waProto.Message{Conversation: proto.String(displayText)})
				} else {
					historyStore.SaveMessage(msgID, task.Contact, "Me", displayText, time.Now(), true)
				}
			}
		}

		// Process task with empty message (initial prompt)
		go func() {
			taskBot.SendButtonResponseFunc = sendButtonResponse
			_, err := taskBot.ProcessTask(task, "", contextMsgs, sendToContact)
			if err != nil {
				fmt.Printf("Task initial processing failed: %v\n", err)
			}
		}()
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

	// Post-Connect: Check if we need to start Batata Setup
	// We need to wait a bit to ensure we can send messages?
	// Or we can just fire it.
	// To send to "Self", we need our own JID.
	if client.Store.ID != nil {
		selfJID := client.Store.ID.ToNonAD()
		sendToSelf := func(msg string) {
			client.SendMessage(context.Background(), selfJID, &waProto.Message{
				Conversation: proto.String(msg),
			})
		}

		// If config was missing, Load() returned error, but NewKernel gave defaults.
		// We can check if file exists or simply if we want to force setup.
		// Actually, Load() updates the config. If it failed, we are using defaults.
		// We should explicitly know if we need setup.
		// Let's retry Load to be sure, or track it.
		// Actually, NewKernel doesn't run Load. We ran Load above.
		// If Load failed, we assume we need setup.
		if _, err := os.Stat(batataKernel.ConfigPath); os.IsNotExist(err) {
			fmt.Println("Starting Batata Setup via WhatsApp...")
			go func() {
				time.Sleep(5 * time.Second) // Give time to connect fully
				batataKernel.StartSetup(sendToSelf)
			}()
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

	// Start Ticker
	go startScheduledTasksTicker()

	<-c

	client.Disconnect()
}

func startScheduledTasksTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	fmt.Println("Starting scheduled tasks ticker...")

	for range ticker.C {
		if taskBot != nil && taskBot.TaskManager != nil {
			startedTasks, err := taskBot.TaskManager.CheckScheduledTasks()
			if err != nil {
				fmt.Printf("Error checking scheduled tasks: %v\n", err)
				continue
			}

			for _, task := range startedTasks {
				// Trigger start callback for each started task
				if taskBot.StartTaskCallback != nil {
					taskBot.StartTaskCallback(task)
				}
			}
		}
	}
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

type ContactEntry struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

func getAllContactsJSON(client *whatsmeow.Client) string {
	var contacts []ContactEntry

	// Groups
	groups, err := client.GetJoinedGroups(context.Background())
	if err == nil {
		for _, g := range groups {
			contacts = append(contacts, ContactEntry{
				Name:   g.Name,
				Number: g.JID.String(),
			})
		}
	}

	// Contacts
	if client.Store.Contacts != nil {
		allContacts, err := client.Store.Contacts.GetAllContacts(context.Background())
		if err == nil {
			for jid, info := range allContacts {
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
				contacts = append(contacts, ContactEntry{
					Name:   name,
					Number: jid.String(),
				})
			}
		}
	}

	jsonData, _ := json.Marshal(contacts)
	return string(jsonData)
}

func reinitLLM(notifyJID types.JID) {
	cfg := batataKernel.Config
	fmt.Printf("Re-initializing LLM. Provider: %s\n", cfg.BrainProvider)

	var providerName, modelName string

	// Define Error Handler
	errorHandler := func(err error) {
		fmt.Printf("LLM Error Handler triggered: %v\n", err)
		batataKernel.ReportLLMError(err, func(msg string) {
			// Send to Self (User)
			if whatsAppClient != nil && whatsAppClient.Store.ID != nil {
				selfJID := whatsAppClient.Store.ID.ToNonAD()
				whatsAppClient.SendMessage(context.Background(), selfJID, &waProto.Message{
					Conversation: proto.String(msg),
				})
			}
		})
	}

	switch cfg.BrainProvider {
	case "cerebras":
		var err error
		key := cfg.CerebrasKey
		if key == "" {
			content, _ := os.ReadFile("config/keys/cerebras.txt")
			key = strings.TrimSpace(string(content))
		}
		model := cfg.CerebrasModel
		if model == "" {
			model = "gpt-oss-120b"
		}
		providerName = "Cerebras"
		modelName = model

		client, err := cerebras.NewClientWithKey(key, model)
		if err != nil {
			fmt.Printf("Failed to initialize Cerebras client: %v\n", err)
			llmClient = nil // Or a no-op client
		} else {
			client.ErrorHandler = errorHandler
			llmClient = client
		}
	case "ollama":
		host := cfg.OllamaHost
		if host == "" {
			host = "http://localhost"
		}
		port := cfg.OllamaPort
		if port == "" {
			port = "11434"
		}
		model := cfg.OllamaModel
		if model == "" {
			model = "qwen3:8b"
		}
		providerName = "Ollama"
		modelName = model

		// Construct URL
		url := host
		if !strings.HasPrefix(url, "http") {
			url = "http://" + url
		}
		fullURL := fmt.Sprintf("%s:%s", url, port)

		client := ollama.NewClient(fullURL, model)
		client.ErrorHandler = errorHandler
		llmClient = client
	case "none":
		fmt.Println("LLM Provider is NONE.")
		llmClient = nil
		providerName = "None"
	default:
		// Default to none
		llmClient = nil
		providerName = "None"
	}

	// Update TaskBot if it exists
	if taskBot != nil {
		taskBot.Client = llmClient
	}

	// Notify if JID is provided and provider is not "none"
	if !notifyJID.IsEmpty() && cfg.BrainProvider != "none" {
		msgTemplate := batata.GetString(cfg.Language, func(s batata.Strings) string { return s.BladyRunning })
		msgText := fmt.Sprintf(msgTemplate, providerName, modelName)

		go func() {
			// Small delay to ensure previous Batata messages are sent first if this was triggered by exiting Batata
			time.Sleep(500 * time.Millisecond)
			if whatsAppClient != nil {
				_, err := whatsAppClient.SendMessage(context.Background(), notifyJID, &waProto.Message{
					Conversation: proto.String(msgText),
				})
				if err != nil {
					fmt.Printf("Failed to send Blady activation message: %v\n", err)
				}
			}
		}()
	}
}
