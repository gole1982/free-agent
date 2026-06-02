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
	db, err := sql.Open("sqlite", path)
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

func (s *Store) GetConversations() ([]Conversation, error) {
	rows, err := s.db.Query("SELECT id, title, created_at FROM conversations ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		var ts string
		if err := rows.Scan(&c.ID, &c.Title, &ts); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", ts)
		conversations = append(conversations, c)
	}
	return conversations, nil
}

func (s *Store) GetConversation(id int64) (*Conversation, error) {
	var c Conversation
	var ts string
	err := s.db.QueryRow("SELECT id, title, created_at FROM conversations WHERE id = ?", id).Scan(&c.ID, &c.Title, &ts)
	if err != nil {
		return nil, err
	}
	c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", ts)
	return &c, nil
}

func (s *Store) DeleteConversation(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec("DELETE FROM messages WHERE conversation_id = ?", id); err != nil {
		tx.Rollback()
		return err
	}

	if _, err := tx.Exec("DELETE FROM conversations WHERE id = ?", id); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *Store) UpdateConversationTitle(id int64, title string) error {
	_, err := s.db.Exec("UPDATE conversations SET title = ? WHERE id = ?", title, id)
	return err
}

func (s *Store) GetConversationCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&count)
	return count, err
}

func (s *Store) Close() error {
	return s.db.Close()
}
