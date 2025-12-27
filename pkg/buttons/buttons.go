package buttons

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waAdv "go.mau.fi/whatsmeow/proto/waAdv"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"whatsabladerunner/pkg/whatsapp"
)

// ButtonsContext stores the context needed for button responses
type ButtonsContext struct {
	MessageID string
	ChatJID   types.JID // The origin chat (LID or Group or PN)
	SenderJID types.JID // The actual sender (LID or PN)
	SenderAlt types.JID // The alternative JID (PN if Sender is LID)
	Message   *waProto.Message
}

type Manager struct {
	lastMessages map[string]*ButtonsContext
}

func NewManager() *Manager {
	return &Manager{
		lastMessages: make(map[string]*ButtonsContext),
	}
}

func (m *Manager) Store(chatJID string, ctx *ButtonsContext) {
	if m.lastMessages == nil {
		m.lastMessages = make(map[string]*ButtonsContext)
	}
	m.lastMessages[chatJID] = ctx
}

func (m *Manager) Get(chatJID string) *ButtonsContext {
	if m.lastMessages == nil {
		return nil
	}
	return m.lastMessages[chatJID]
}

func (m *Manager) Resolve(chatJID string, text string) (resolvedDisplayText, resolvedButtonID string, found bool) {
	btnCtx := m.Get(chatJID)
	if btnCtx == nil || btnCtx.Message == nil {
		return "", "", false
	}
	if btnCtx.Message.ButtonsMessage != nil {
		for _, btn := range btnCtx.Message.ButtonsMessage.Buttons {
			if btn.ButtonText != nil && btn.ButtonText.DisplayText != nil {
				if strings.EqualFold(strings.TrimSpace(*btn.ButtonText.DisplayText), strings.TrimSpace(text)) {
					return *btn.ButtonText.DisplayText, *btn.ButtonID, true
				}
			}
		}
	} else if btnCtx.Message.ListMessage != nil {
		for _, section := range btnCtx.Message.ListMessage.Sections {
			for _, row := range section.Rows {
				if row.Title != nil {
					if strings.EqualFold(strings.TrimSpace(*row.Title), strings.TrimSpace(text)) {
						return *row.Title, *row.RowID, true
					}
				}
			}
		}
	}
	return "", "", false
}

func (m *Manager) SendResponse(ctx context.Context, client *whatsmeow.Client, chatJID string, displayText, buttonID string) (string, error) {
	btnCtx := m.Get(chatJID)
	if btnCtx == nil {
		return "", fmt.Errorf("no buttons context found for chat %s", chatJID)
	}

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
		// Conversation: proto.String("\t\t\t\t\t\t\t\t\t\t"), // REMOVED: Suspicious fingerprint
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
	fmt.Printf("[ButtonResponse] Proto Base64 Mirror V4: %s\n", base64.StdEncoding.EncodeToString(marshaled))
	fmt.Printf("[ButtonResponse] Sending Mirror Response (V4): Dest=%s, Part=%s, StanzaID=%s\n",
		target, partPN, btnCtx.MessageID)

	resp, err := whatsapp.SendWithStealth(ctx, client, target, protoMsg)
	if err != nil {
		fmt.Printf("[ButtonResponse] Failed with Dest=%s: %v\n", target, err)
		if target.String() != btnCtx.SenderAlt.String() && !btnCtx.SenderAlt.IsEmpty() {
			targetPN := btnCtx.SenderAlt
			fmt.Printf("[ButtonResponse] Retry with Dest=PN: %s\n", targetPN)
			resp, err = whatsapp.SendWithStealth(ctx, client, targetPN, protoMsg)
			if err != nil {
				return "", fmt.Errorf("failed after PN retry: %w", err)
			}
			return resp.ID, nil
		}
		return "", err
	}

	return resp.ID, nil
}

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
