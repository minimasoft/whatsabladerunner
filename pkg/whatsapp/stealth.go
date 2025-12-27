package whatsapp

import (
	"context"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// SendWithStealth sends a message with human-like delays and typing indicators
func SendWithStealth(ctx context.Context, client *whatsmeow.Client, target types.JID, msg *waProto.Message) (whatsmeow.SendResponse, error) {
	if client == nil {
		return whatsmeow.SendResponse{}, context.Canceled
	}

	// 1. Calculate and handle delays/indicators
	isInteractive := msg.ButtonsResponseMessage != nil || msg.ListResponseMessage != nil || msg.ProtocolMessage != nil

	textLen := 0
	if msg.Conversation != nil {
		textLen = len(*msg.Conversation)
	} else if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.Text != nil {
		textLen = len(*msg.ExtendedTextMessage.Text)
	}

	// Only send typing indicator if it's shared text and not a button click
	if !isInteractive && textLen > 0 {
		_ = client.SendChatPresence(ctx, target, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	}

	// Base delay:
	// - Interactive: very short (300-700ms) to simulate reaction time
	// - Text: 1-2 seconds + typing speed
	var delay time.Duration
	if isInteractive {
		delay = time.Duration(300+rand.Int63n(400)) * time.Millisecond
	} else {
		baseDelay := time.Duration(1000+rand.Int63n(1000)) * time.Millisecond
		charDelay := time.Duration(int64(textLen)*(30+rand.Int63n(40))) * time.Millisecond
		delay = baseDelay + charDelay
	}

	// Cap delay at 15 seconds to avoid timeouts or excessive waiting
	if delay > 15*time.Second {
		delay = 15 * time.Second
	}

	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return whatsmeow.SendResponse{}, ctx.Err()
	}

	// 3. Send Message
	resp, err := client.SendMessage(ctx, target, msg)

	// 4. Send Paused presence
	_ = client.SendChatPresence(ctx, target, types.ChatPresencePaused, types.ChatPresenceMediaText)

	return resp, err
}
