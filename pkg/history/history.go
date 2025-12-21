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
		message_id TEXT UNIQUE,
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

func (h *HistoryStore) SaveMessage(messageID, chatJID, senderJID, content string, timestamp time.Time, isFromMe bool) error {
	query := `INSERT OR IGNORE INTO messages (message_id, chat_jid, sender_jid, content, timestamp, is_from_me) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := h.db.Exec(query, messageID, chatJID, senderJID, content, timestamp, isFromMe)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}
	return nil
}

func (h *HistoryStore) GetRecentMessages(chatJID string, limit int) ([]string, error) {
	query := `
	SELECT sender_jid, content, timestamp, is_from_me 
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
	var rawMessages []string
	for rows.Next() {
		var senderJID, content string
		var timestamp time.Time
		var isFromMe bool
		if err := rows.Scan(&senderJID, &content, &timestamp, &isFromMe); err != nil {
			return nil, err
		}

		prefix := "User"
		if isFromMe {
			prefix = "Me"
		} else {
			prefix = "User"
		}

		// Format: "[2023-01-01 12:00:00] User: Message"
		formatted := fmt.Sprintf("[%s] %s: %s", timestamp.Format("2006-01-02 15:04:05"), prefix, content)
		rawMessages = append(rawMessages, formatted)
	}

	// Reverse to get chronological order (Oldest -> Newest)
	for i := len(rawMessages) - 1; i >= 0; i-- {
		messages = append(messages, rawMessages[i])
	}

	return messages, nil
}

// GetMessagesSince returns messages after the given unix timestamp
// Returns formatted messages list, the timestamp of the last message (maxUnix), and error
func (h *HistoryStore) GetMessagesSince(chatJID string, sinceUnix int64) ([]string, int64, error) {
	// If sinceUnix is 0, getting ALL messages might be too much, maybe default to last 10?
	// But the user request implies grabbing "new" messages.
	// If it's a new task, maybe start fresh?
	// Let's assume if 0 (never processed), we get recent ones to kickstart, OR we get all since 0.
	// User said: "When the task starts it may encounter no new messages... and that's ok"
	// But for the very FIRST run, we want the prompt/trigger message.
	// The trigger message is already saved.
	// If we return nothing for timestamp 0, the task sees nothing.
	// Strategy: If sinceUnix is 0, we treat it as "GetRecent(10)" to establish context,
	// and set timestamp to the latest.

	if sinceUnix == 0 {
		// Just reuse GetRecentMessages but we need the max timestamp too
		// Implementing a variation here
		query := `
		SELECT sender_jid, content, timestamp, is_from_me 
		FROM messages 
		WHERE chat_jid = ? 
		ORDER BY timestamp DESC 
		LIMIT 10`

		rows, err := h.db.Query(query, chatJID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to query initial messages: %w", err)
		}
		defer rows.Close()

		var rawMessages []string
		var maxUnix int64 = 0

		for rows.Next() {
			var sender string
			var content string
			var ts time.Time
			var isFromMe bool
			if err := rows.Scan(&sender, &content, &ts, &isFromMe); err != nil {
				return nil, 0, err
			}

			if ts.Unix() > maxUnix {
				maxUnix = ts.Unix()
			}

			prefix := "User"
			if isFromMe {
				prefix = "Me"
			}

			formatted := fmt.Sprintf("[%s] %s: %s", ts.Format("2006-01-02 15:04:05"), prefix, content)
			rawMessages = append(rawMessages, formatted)
		}

		// Reverse
		var messages []string
		for i := len(rawMessages) - 1; i >= 0; i-- {
			messages = append(messages, rawMessages[i])
		}
		return messages, maxUnix, nil
	}

	// Normal case: Get messages strictly > sinceUnix
	query := `
	SELECT sender_jid, content, timestamp, is_from_me 
	FROM messages 
	WHERE chat_jid = ? AND timestamp > ?
	ORDER BY timestamp ASC`

	sinceTime := time.Unix(sinceUnix, 0)
	rows, err := h.db.Query(query, chatJID, sinceTime)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query new messages: %w", err)
	}
	defer rows.Close()

	var messages []string
	var maxUnix int64 = sinceUnix

	for rows.Next() {
		var sender string
		var content string
		var ts time.Time
		var isFromMe bool
		if err := rows.Scan(&sender, &content, &ts, &isFromMe); err != nil {
			return nil, 0, err
		}

		if ts.Unix() > maxUnix {
			maxUnix = ts.Unix()
		}

		prefix := "User"
		if isFromMe {
			prefix = "Me"
		}

		formatted := fmt.Sprintf("[%s] %s: %s", ts.Format("2006-01-02 15:04:05"), prefix, content)
		messages = append(messages, formatted)
	}

	return messages, maxUnix, nil
}
