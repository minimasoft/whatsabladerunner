package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/mdp/qrterminal/v3"

	_ "github.com/mattn/go-sqlite3"
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("------------------------------------------------")
		fmt.Printf("Received a message!\n")
		fmt.Printf("ID: %s\n", v.Info.ID)
		fmt.Printf("Time: %s\n", v.Info.Timestamp)
		fmt.Printf("Sender: %s (PushName: %s)\n", v.Info.Sender, v.Info.PushName)
		fmt.Printf("Chat: %s\n", v.Info.Chat)
		if v.Message != nil {
			fmt.Printf("Content: %+v\n", v.Message)
		}
		fmt.Println("------------------------------------------------")
	case *events.HistorySync:
		id := v.Data.GetSyncType()
		if id == waHistorySync.HistorySync_FULL || id == waHistorySync.HistorySync_RECENT {
			fmt.Printf("Received History Sync (Type: %s)\n", id)

			conversations := v.Data.GetConversations()
			// Sort by LastMsgTimestamp descending
			sort.Slice(conversations, func(i, j int) bool {
				return conversations[i].GetLastMsgTimestamp() > conversations[j].GetLastMsgTimestamp()
			})

			for _, conv := range conversations {
				fmt.Printf("Conversation: %s (%s)\n", conv.GetID(), conv.GetName())
				if conv.GetUnreadCount() > 0 {
					fmt.Printf("  Unread: %d\n", conv.GetUnreadCount())
				}
				if conv.GetIsParentGroup() || conv.GetIsDefaultSubgroup() {
					fmt.Println("  [Group/Community]")
				}

				// List participants if available (HistorySync might not always have full participant list for all convs, but let's check)
				// Note: Conversation proto in HistorySync has 'participant' field
				participants := conv.GetParticipant()
				if len(participants) > 0 {
					fmt.Println("  Participants:")
					for _, p := range participants {
						fmt.Printf("    - %s\n", p.GetUserJID())
					}
				}
			}
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
	clientLog := waLog.Stdout("Client", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(eventHandler)

	if client.Store.ID == nil {
		// No ID stored, new login
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
