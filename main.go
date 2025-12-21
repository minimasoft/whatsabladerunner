package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"

	"go.mau.fi/whatsmeow"
	waAdv "go.mau.fi/whatsmeow/proto/waAdv"
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
	"whatsabladerunner/pkg/bot"
	"whatsabladerunner/pkg/cerebras"
	"whatsabladerunner/pkg/history"
	"whatsabladerunner/pkg/llm"
	"whatsabladerunner/pkg/locks"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/pkg/tasks"
	"whatsabladerunner/workflows"
)

// LLMProvider selects which LLM provider to use: "ollama" or "cerebras"
const LLMProvider = "cerebras" //"ollama"

var (
	llmClient      llm.Client
	convManager    *agent.ConversationManager
	whatsAppClient *whatsmeow.Client
	historyStore   *history.HistoryStore
	taskBot        *bot.Bot // Global bot instance for task handling
	taskLocks      *locks.KeyedMutex
)

// ButtonsContext stores the context needed for button responses
type ButtonsContext struct {
	MessageID string
	ChatJID   types.JID // The origin chat (LID or Group or PN)
	SenderJID types.JID // The actual sender (LID or PN)
	SenderAlt types.JID // The alternative JID (PN if Sender is LID)
	Message   *waProto.Message
}

// lastButtonsMessage stores the last buttons/list message per chat for context
var lastButtonsMessage = make(map[string]*ButtonsContext)

// WithheldMessage stores a blocked message for potential "LET IT BE" override
type WithheldMessage struct {
	Message       string       // The blocked message text
	TargetChatJID types.JID    // Where to send if unblocked
	SendFunc      func(string) // Function to use for sending
}

// lastWithheldMessage stores the most recent watcher-blocked message
var lastWithheldMessage *WithheldMessage

const BotPrefix = "[Blady] : "

// Helper to strip metadata from quoted messages
func getCleanQuotedMessage(orig *waProto.Message) *waProto.Message {
	if orig == nil {
		return nil
	}
	clean := &waProto.Message{}
	// Copy only content fields, ignore transport metadata or context info wrapper
	if orig.Conversation != nil {
		clean.Conversation = orig.Conversation
	}
	if orig.ExtendedTextMessage != nil {
		clean.ExtendedTextMessage = &waProto.ExtendedTextMessage{
			Text: orig.ExtendedTextMessage.Text,
			// Ignore ContextInfo, PreviewType, etc.
		}
	}
	if orig.ButtonsMessage != nil {
		// Deep copy ButtonsMessage content, EXCLUDING ContextInfo
		bm := orig.ButtonsMessage
		clean.ButtonsMessage = &waProto.ButtonsMessage{
			ContentText: bm.ContentText,
			FooterText:  bm.FooterText,
			Buttons:     bm.Buttons, // Buttons themselves are simple structs usually
			HeaderType:  bm.HeaderType,
			// explicit: ContextInfo: nil
		}
		if bm.HeaderType != nil && *bm.HeaderType != waProto.ButtonsMessage_EMPTY {
			// Copy header content if present (Text, Doc, Image etc)
			// But for safety, maybe just text?
			// The log showed headerType:EMPTY usually.
			// If we need media headers, we'd copy them.
		}
	}
	if orig.ListMessage != nil {
		lm := orig.ListMessage
		clean.ListMessage = &waProto.ListMessage{
			Title:       lm.Title,
			Description: lm.Description,
			ButtonText:  lm.ButtonText,
			ListType:    lm.ListType,
			Sections:    lm.Sections,
			FooterText:  lm.FooterText,
			// explicit: ContextInfo: nil
		}
	}
	// ... (other types simplified for now, focusing on text/buttons/list)

	// Ensure MessageContextInfo is nil or empty to avoid 479 on re-send
	return clean
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
				// Store RAW JIDs to allow strategy testing
				lastButtonsMessage[v.Info.Chat.String()] = &ButtonsContext{
					MessageID: v.Info.ID,
					ChatJID:   v.Info.Chat,
					SenderJID: v.Info.Sender,
					SenderAlt: v.Info.MessageSource.SenderAlt,
					Message:   v.Message,
				}
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
				lastButtonsMessage[v.Info.Chat.String()] = &ButtonsContext{
					MessageID: v.Info.ID,
					ChatJID:   v.Info.Chat,
					SenderJID: v.Info.Sender,
					SenderAlt: v.Info.MessageSource.SenderAlt,
					Message:   v.Message,
				}
				fmt.Printf("[ButtonsContext] Stored list message ID=%s from chat=%s sender=%s senderAlt=%s\n",
					v.Info.ID, v.Info.Chat, v.Info.Sender, v.Info.MessageSource.SenderAlt)
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
							// ... (Reuse existing button logic by wrapping or extracting?
							// The previous logic was closure-heavy. Let's duplicate or refactor.
							// For safety and speed in this specific 'replace' block, I will copy the logic
							// or better, define a shared helper if possible.
							// But since I can't easily refactor outside this block without more edits,
							// and the logic is complex (V4 mirror), I'll copy the core call back to the main client/logic.
							// Actually, I can use the same logic as before, just inline it or ensure 'v' is available.
							// 'v' IS available in the closure (eventHandler closure).
							// So I can just copy the inner function body.

							if whatsAppClient != nil {
								// Get stored buttons context
								btnCtx := lastButtonsMessage[chatJIDForContext]
								if btnCtx == nil {
									fmt.Printf("[ButtonResponse] No buttons context found for chat %s, falling back to text\n", chatJIDForContext)
									whatsAppClient.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{Conversation: proto.String(displayText)})
									return
								}
								// ... Full V4 Mirror Logic ...
								// To avoid huge code duplication risk in this diff, I will simplify or rely on the fact
								// that I can't easily copy 100 lines here.
								// Wait, the previous block I'm replacing HAD the V4 logic.
								// I should check if I can keep it.
								// I can't call the previous 'sendButtonResponse' because it was defined inside the 'convManager.StartWorkflow' block which I am removing.
								// So I MUST redefine it or extract it.
								// Given the 'replace' limit, I'll define it here.

								// Re-implementing V4 Minimal for brevity in this block, assuming similar context
								// Actually, I should probably copy the logic exactly to ensure stability.
								// It is verbose but safe.

								cleanQuote := getCleanQuotedMessage(btnCtx.Message)
								cleanQuote.MessageContextInfo = &waProto.MessageContextInfo{}
								var hash []byte
								var ts uint64 = uint64(time.Now().Unix())
								if btnCtx.Message.MessageContextInfo != nil && btnCtx.Message.MessageContextInfo.DeviceListMetadata != nil {
									hash = btnCtx.Message.MessageContextInfo.DeviceListMetadata.RecipientKeyHash
									if btnCtx.Message.MessageContextInfo.DeviceListMetadata.RecipientTimestamp != nil {
										ts = *btnCtx.Message.MessageContextInfo.DeviceListMetadata.RecipientTimestamp
									}
								}
								partPN := btnCtx.SenderAlt.ToNonAD().String()
								if btnCtx.SenderAlt.IsEmpty() {
									partPN = btnCtx.SenderJID.ToNonAD().String()
								}
								target := btnCtx.ChatJID
								protoMsg := &waProto.Message{
									Conversation: proto.String("\t\t\t\t\t\t\t\t\t\t"),
									MessageContextInfo: &waProto.MessageContextInfo{
										DeviceListMetadataVersion: proto.Int32(2),
										DeviceListMetadata: &waProto.DeviceListMetadata{
											SenderKeyHash: hash, SenderTimestamp: proto.Uint64(ts),
											SenderAccountType:   waAdv.ADVEncryptionType_E2EE.Enum(),
											ReceiverAccountType: waAdv.ADVEncryptionType_E2EE.Enum(),
										},
									},
									ButtonsResponseMessage: &waProto.ButtonsResponseMessage{
										SelectedButtonID: proto.String(buttonID),
										Response:         &waProto.ButtonsResponseMessage_SelectedDisplayText{SelectedDisplayText: displayText},
										ContextInfo: &waProto.ContextInfo{
											StanzaID: proto.String(btnCtx.MessageID), Participant: proto.String(partPN), QuotedMessage: cleanQuote,
										},
										Type: waProto.ButtonsResponseMessage_DISPLAY_TEXT.Enum(),
									},
								}

								// Send
								_, err := whatsAppClient.SendMessage(context.Background(), target, protoMsg)
								if err != nil {
									// Fallback retry
									if target.String() != btnCtx.SenderAlt.String() && !btnCtx.SenderAlt.IsEmpty() {
										whatsAppClient.SendMessage(context.Background(), btnCtx.SenderAlt, protoMsg)
									}
								}
								historyStore.SaveMessage("btn-resp", cJID, "Me", displayText, time.Now(), true)
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
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	whatsAppClient = whatsmeow.NewClient(deviceStore, clientLog)
	client := whatsAppClient

	// Initialize LLM client based on provider selection
	switch LLMProvider {
	case "cerebras":
		fmt.Println("Using Cerebras LLM provider")
		var err error
		llmClient, err = cerebras.NewClient("config/keys/cerebras.txt", "gpt-oss-120b")
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize Cerebras client: %v", err))
		}
	case "ollama":
		fallthrough
	default:
		fmt.Println("Using Ollama LLM provider")
		llmClient = ollama.NewClient("http://localhost:11434", "qwen3:8b")
	}
	convManager = agent.NewConversationManager()
	taskLocks = locks.New()

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
				// Get stored buttons context
				btnCtx := lastButtonsMessage[contactChatID]
				if btnCtx == nil {
					fmt.Printf("[ButtonResponse] No buttons context found for chat %s, falling back to text\n", contactChatID)
					// Fallback to regular text response
					whatsAppClient.SendMessage(context.Background(), contactJID, &waProto.Message{
						Conversation: proto.String(displayText),
					})
					return
				}

				fmt.Printf("[ButtonResponse] Sending: displayText=%s, buttonID=%s, stanzaID=%s\n",
					displayText, buttonID, btnCtx.MessageID)

				// Final Strategy: Full Mirror (V4)
				cleanQuote := getCleanQuotedMessage(btnCtx.Message)
				cleanQuote.MessageContextInfo = &waProto.MessageContextInfo{}

				var hash []byte
				var ts uint64 = uint64(time.Now().Unix())
				if btnCtx.Message.MessageContextInfo != nil && btnCtx.Message.MessageContextInfo.DeviceListMetadata != nil {
					hash = btnCtx.Message.MessageContextInfo.DeviceListMetadata.RecipientKeyHash
					if btnCtx.Message.MessageContextInfo.DeviceListMetadata.RecipientTimestamp != nil {
						ts = *btnCtx.Message.MessageContextInfo.DeviceListMetadata.RecipientTimestamp
					}
				}

				partPN := btnCtx.SenderAlt.ToNonAD().String()
				if btnCtx.SenderAlt.IsEmpty() {
					partPN = btnCtx.SenderJID.ToNonAD().String()
				}

				target := btnCtx.ChatJID

				protoMsg := &waProto.Message{
					Conversation: proto.String("\t\t\t\t\t\t\t\t\t\t"),
					MessageContextInfo: &waProto.MessageContextInfo{
						DeviceListMetadataVersion: proto.Int32(2),
						DeviceListMetadata: &waProto.DeviceListMetadata{
							SenderKeyHash:       hash,
							SenderTimestamp:     proto.Uint64(ts),
							SenderAccountType:   waAdv.ADVEncryptionType_E2EE.Enum(),
							ReceiverAccountType: waAdv.ADVEncryptionType_E2EE.Enum(),
						},
					},
					ButtonsResponseMessage: &waProto.ButtonsResponseMessage{
						SelectedButtonID: proto.String(buttonID),
						Response: &waProto.ButtonsResponseMessage_SelectedDisplayText{
							SelectedDisplayText: displayText,
						},
						ContextInfo: &waProto.ContextInfo{
							StanzaID:      proto.String(btnCtx.MessageID),
							Participant:   proto.String(partPN),
							QuotedMessage: cleanQuote,
						},
						Type: waProto.ButtonsResponseMessage_DISPLAY_TEXT.Enum(),
					},
				}

				marshaled, _ := proto.Marshal(protoMsg)
				fmt.Printf("[ButtonResponse] Proto Base64 Mirror V4 (Task): %s\n", base64.StdEncoding.EncodeToString(marshaled))

				fmt.Printf("[ButtonResponse] Sending Mirror Response (V4 Task): Dest=%s, Part=%s, StanzaID=%s\n",
					target, partPN, btnCtx.MessageID)

				resp, err := whatsAppClient.SendMessage(context.Background(), target, protoMsg)
				if err != nil {
					fmt.Printf("[ButtonResponse] Failed (Task) with Dest=%s: %v\n", target, err)
					if target.String() != btnCtx.SenderAlt.String() && !btnCtx.SenderAlt.IsEmpty() {
						targetPN := btnCtx.SenderAlt
						fmt.Printf("[ButtonResponse] Retry (Task) with Dest=PN: %s\n", targetPN)
						resp, err = whatsAppClient.SendMessage(context.Background(), targetPN, protoMsg)
						if err == nil {
							fmt.Printf("[ButtonResponse] Success (Task) with Dest=PN! ID: %s\n", resp.ID)
							historyStore.SaveMessage(resp.ID, task.Contact, "Me", displayText, time.Now(), true)
						} else {
							fmt.Printf("[ButtonResponse] Failed (Task) with Dest=PN: %v\n", err)
						}
					}
				} else {
					fmt.Printf("[ButtonResponse] Success (Task)! ID: %s\n", resp.ID)
					historyStore.SaveMessage(resp.ID, task.Contact, "Me", displayText, time.Now(), true)
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

	data, err := json.Marshal(contacts)
	if err != nil {
		return "[]"
	}
	return string(data)
}
