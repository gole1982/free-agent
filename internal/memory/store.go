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

type AgentTraits struct {
	AgentName    string
	Efficiency   float64
	Quality      float64
	Creativity   float64
	Collaboration float64
	UpdatedAt    time.Time
}

type SecurityPolicy struct {
	ID          int64
	PolicyType  string
	PolicyContent string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AgentExecution struct {
	ID              int64
	ConversationID  int64
	AgentName       string
	Task            string
	Result          string
	Status          string
	StartedAt       time.Time
	CompletedAt     *time.Time
}

func (s *Store) GetAgentTraits(agentName string) (*AgentTraits, error) {
	var traits AgentTraits
	var ts string
	err := s.db.QueryRow(
		`SELECT agent_name, efficiency, quality, creativity, collaboration, updated_at 
		 FROM agent_traits WHERE agent_name = ?`, agentName,
	).Scan(&traits.AgentName, &traits.Efficiency, &traits.Quality, &traits.Creativity, &traits.Collaboration, &ts)
	if err == sql.ErrNoRows {
		return &AgentTraits{
			AgentName:    agentName,
			Efficiency:   0.5,
			Quality:      0.5,
			Creativity:   0.5,
			Collaboration: 0.5,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	traits.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", ts)
	return &traits, nil
}

func (s *Store) SaveAgentTraits(traits *AgentTraits) error {
	_, err := s.db.Exec(
		`INSERT INTO agent_traits (agent_name, efficiency, quality, creativity, collaboration, updated_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(agent_name) DO UPDATE SET
		 efficiency = excluded.efficiency,
		 quality = excluded.quality,
		 creativity = excluded.creativity,
		 collaboration = excluded.collaboration,
		 updated_at = CURRENT_TIMESTAMP`,
		traits.AgentName, traits.Efficiency, traits.Quality, traits.Creativity, traits.Collaboration,
	)
	return err
}

func (s *Store) GetAllAgentTraits() ([]*AgentTraits, error) {
	rows, err := s.db.Query(
		`SELECT agent_name, efficiency, quality, creativity, collaboration, updated_at FROM agent_traits`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []*AgentTraits
	for rows.Next() {
		var t AgentTraits
		var ts string
		if err := rows.Scan(&t.AgentName, &t.Efficiency, &t.Quality, &t.Creativity, &t.Collaboration, &ts); err != nil {
			return nil, err
		}
		t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", ts)
		all = append(all, &t)
	}
	return all, nil
}

func (s *Store) SaveSecurityPolicy(policyType, policyContent string, enabled bool) error {
	_, err := s.db.Exec(
		`INSERT INTO security_policies (policy_type, policy_content, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		policyType, policyContent, boolToInt(enabled),
	)
	return err
}

func (s *Store) UpdateSecurityPolicy(policyType, policyContent string, enabled bool) error {
	_, err := s.db.Exec(
		`UPDATE security_policies SET policy_content = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE policy_type = ?`,
		policyContent, boolToInt(enabled), policyType,
	)
	return err
}

func (s *Store) GetSecurityPolicy(policyType string) (*SecurityPolicy, error) {
	var p SecurityPolicy
	var cts, uts string
	err := s.db.QueryRow(
		`SELECT id, policy_type, policy_content, enabled, created_at, updated_at FROM security_policies WHERE policy_type = ?`,
		policyType,
	).Scan(&p.ID, &p.PolicyType, &p.PolicyContent, &p.Enabled, &cts, &uts)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", cts)
	p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", uts)
	return &p, nil
}

func (s *Store) GetAllSecurityPolicies() ([]*SecurityPolicy, error) {
	rows, err := s.db.Query(
		`SELECT id, policy_type, policy_content, enabled, created_at, updated_at FROM security_policies WHERE enabled = 1`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []*SecurityPolicy
	for rows.Next() {
		var p SecurityPolicy
		var cts, uts string
		if err := rows.Scan(&p.ID, &p.PolicyType, &p.PolicyContent, &p.Enabled, &cts, &uts); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", cts)
		p.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", uts)
		all = append(all, &p)
	}
	return all, nil
}

func (s *Store) RecordAgentExecution(convID int64, agentName, task, result, status string) error {
	_, err := s.db.Exec(
		`INSERT INTO agent_executions (conversation_id, agent_name, task, result, status, started_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		convID, agentName, task, result, status,
	)
	return err
}

func (s *Store) CompleteAgentExecution(convID int64, agentName, result, status string) error {
	_, err := s.db.Exec(
		`UPDATE agent_executions SET result = ?, status = ?, completed_at = CURRENT_TIMESTAMP
		 WHERE conversation_id = ? AND agent_name = ? AND completed_at IS NULL
		 ORDER BY started_at DESC LIMIT 1`,
		result, status, convID, agentName,
	)
	return err
}

func (s *Store) GetAgentExecutions(convID int64) ([]*AgentExecution, error) {
	rows, err := s.db.Query(
		`SELECT id, conversation_id, agent_name, task, result, status, started_at, completed_at
		 FROM agent_executions WHERE conversation_id = ? ORDER BY started_at`,
		convID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []*AgentExecution
	for rows.Next() {
		var e AgentExecution
		var st, ct sql.NullString
		if err := rows.Scan(&e.ID, &e.ConversationID, &e.AgentName, &e.Task, &e.Result, &e.Status, &st, &ct); err != nil {
			return nil, err
		}
		e.StartedAt, _ = time.Parse("2006-01-02 15:04:05", st.String)
		if ct.Valid {
			t, _ := time.Parse("2006-01-02 15:04:05", ct.String)
			e.CompletedAt = &t
		}
		all = append(all, &e)
	}
	return all, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
