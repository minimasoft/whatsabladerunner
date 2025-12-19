package history

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type HistoryStore struct {
	db *sql.DB
}

func New(dbPath string) (*HistoryStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open history db: %w", err)
	}

	// Create table if not exists
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_jid TEXT,
		sender_jid TEXT,
		content TEXT,
		timestamp DATETIME,
		is_from_me BOOLEAN
	);
	CREATE INDEX IF NOT EXISTS idx_chat_jid ON messages(chat_jid);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON messages(timestamp);
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create messages table: %w", err)
	}

	return &HistoryStore{db: db}, nil
}

func (h *HistoryStore) SaveMessage(chatJID, senderJID, content string, timestamp time.Time, isFromMe bool) error {
	query := `INSERT INTO messages (chat_jid, sender_jid, content, timestamp, is_from_me) VALUES (?, ?, ?, ?, ?)`
	_, err := h.db.Exec(query, chatJID, senderJID, content, timestamp, isFromMe)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

func (h *HistoryStore) GetRecentMessages(chatJID string, limit int) ([]string, error) {
	query := `
	SELECT sender_jid, content, is_from_me 
	FROM messages 
	WHERE chat_jid = ? 
	ORDER BY timestamp DESC 
	LIMIT ?`

	rows, err := h.db.Query(query, chatJID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent messages: %w", err)
	}
	defer rows.Close()

	var messages []string
	// Use a slice to store reversed results since we query DESC but want context in chronological order?
	// Usually context is presented oldest to newest.
	// Since we query DESC (newest first), we will get: [Newest, ..., Oldest]
	// We should reverse this list before returning.

	var rawMessages []string
	for rows.Next() {
		var senderJID, content string
		var isFromMe bool
		if err := rows.Scan(&senderJID, &content, &isFromMe); err != nil {
			return nil, err
		}

		prefix := "User"
		if isFromMe {
			prefix = "Me" // Or Bot? "Me" is clearer for Note To Self.
		} else {
			// Maybe use a simplified ID?
			prefix = "User"
		}

		// Format: "User: Message"
		formatted := fmt.Sprintf("%s: %s", prefix, content)
		rawMessages = append(rawMessages, formatted)
	}

	// Reverse to get chronological order (Oldest -> Newest)
	for i := len(rawMessages) - 1; i >= 0; i-- {
		messages = append(messages, rawMessages[i])
	}

	return messages, nil
}
