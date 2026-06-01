package memory

import (
	"database/sql"
	"time"
)

type Message struct {
	ID             int64
	ConversationID int64
	Role           string
	Content        string
	Timestamp      time.Time
}

type Conversation struct {
	ID        int64
	Title     string
	CreatedAt time.Time
}

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := initDB(db); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) CreateConversation(title string) (*Conversation, error) {
	result, err := s.db.Exec("INSERT INTO conversations (title) VALUES (?)", title)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Conversation{
		ID:        id,
		Title:     title,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Store) AddMessage(convID int64, role, content string) (*Message, error) {
	result, err := s.db.Exec(
		"INSERT INTO messages (conversation_id, role, content) VALUES (?, ?, ?)",
		convID, role, content,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &Message{
		ID:             id,
		ConversationID: convID,
		Role:           role,
		Content:        content,
		Timestamp:      time.Now(),
	}, nil
}

func (s *Store) GetMessages(convID int64) ([]Message, error) {
	rows, err := s.db.Query(
		"SELECT id, conversation_id, role, content, timestamp FROM messages WHERE conversation_id = ? ORDER BY timestamp",
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		var ts string
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &ts); err != nil {
			return nil, err
		}
		m.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		messages = append(messages, m)
	}
	return messages, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
