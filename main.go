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
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"whatsabladerunner/pkg/agent"
	"whatsabladerunner/pkg/ollama"
	"whatsabladerunner/workflows"
)

var (
	ollamaClient *ollama.Client
	convManager  *agent.ConversationManager
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("------------------------------------------------")
		fmt.Printf("Received a message!\n")
		//fmt.Printf("ID: %s\n", v.Info.ID)
		fmt.Printf("Time: %s\n", v.Info.Timestamp)
		fmt.Printf("Sender: %s (PushName: %s)\n", v.Info.Sender, v.Info.PushName)
		fmt.Printf("Chat: %s\n", v.Info.Chat)

		// Check if the message is sent to self (Note to Self)
		if v.Info.IsFromMe && v.Info.Chat.User == v.Info.Sender.User {
			fmt.Println("it's you - triggering workflow")

			fmt.Printf("DEBUG: Message Struct: %+v\n", v.Message)
			if v.Message != nil {
				text := ""
				if v.Message.ExtendedTextMessage != nil {
					text = *v.Message.ExtendedTextMessage.Text
				} else if v.Message.Conversation != nil {
					text = *v.Message.Conversation
				}

				if text != "" {
					fmt.Printf("DEBUG: Extracted text: %s\n", text)
					// Start workflow in background, managed by ConversationManager
					chatID := v.Info.Chat.String()
					convManager.StartWorkflow(chatID, func(ctx context.Context) {
						wf := workflows.NewDemoWorkflow(ollamaClient)
						wf.Run(ctx, text)
					})
				} else {
					fmt.Println("DEBUG: No text found in message")
				}
			}
		} else {
			if v.Message != nil && v.Message.ExtendedTextMessage != nil {
				fmt.Printf("Content: %+v\n", v.Message.ExtendedTextMessage.Text)
			}
		}
		fmt.Println("------------------------------------------------")
	case *events.HistorySync:
		// ... existing history sync code ...
		id := v.Data.GetSyncType()
		if id == waHistorySync.HistorySync_FULL || id == waHistorySync.HistorySync_RECENT {
			fmt.Printf("Received History Sync (Type: %s)\n", id)
		}
	}
}

func main() {
	// |------------------------------------------------------------------------------------------------------|
	// | NOTE: You must also import the appropriate DB connector, e.g. github.com/mattn/go-sqlite3 for SQLite |
	// |------------------------------------------------------------------------------------------------------|

	dbLog := waLog.Stdout("Database", "INFO", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Initialize custom services
	ollamaClient = ollama.NewClient("http://localhost:11434", "qwen3:8b")
	convManager = agent.NewConversationManager()

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
