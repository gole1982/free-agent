# Free Agent MVP Implementation Plan

&gt; **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a minimum viable multi-AI agent system that can use the free chat API to write code.

**Architecture:** Go-based CLI application with SQLite storage, modular agent design, and clean separation of concerns.

**Tech Stack:**
- Go 1.21+
- SQLite (mattn/go-sqlite3)
- Cobra (CLI framework)
- Bubble Tea (interactive UI)

---

## Project Structure

```
free-agent/
├── cmd/
│   └── free-agent/
│       └── main.go
├── internal/
│   ├── llm/
│   │   ├── gateway.go
│   │   └── client.go
│   ├── memory/
│   │   ├── store.go
│   │   └── schema.go
│   ├── agent/
│   │   ├── base.go
│   │   └── coder.go
│   └── tools/
│       └── filesystem.go
├── pkg/
│   └── config/
│       └── config.go
├── data/
├── config/
│   └── config.yaml
├── go.mod
└── go.sum
```

---

## Task 1: Initialize Go Project

**Files:**
- Create: `go.mod`
- Create: `go.sum`

- [ ] **Step 1: Initialize Go module**

Run: `go mod init github.com/vibe-coding/free-agent`
Expected: Creates `go.mod` with module name

- [ ] **Step 2: Add initial dependencies**

Run:
```
go get github.com/spf13/cobra@latest
go get github.com/mattn/go-sqlite3@latest
go get gopkg.in/yaml.v3@latest
```
Expected: Dependencies added to `go.mod`

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: initialize go module"
```

---

## Task 2: Configuration Management

**Files:**
- Create: `pkg/config/config.go`
- Create: `config/config.yaml`

- [ ] **Step 1: Write config package**

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM struct {
		BaseURL string `yaml:"base_url"`
	} `yaml:"llm"`
	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".free-agent", "config.yaml")
}
```

- [ ] **Step 2: Write default config.yaml**

```yaml
llm:
  base_url: "https://818233.xyz"
storage:
  path: "./data/free-agent.db"
```

- [ ] **Step 3: Commit**

```bash
git add pkg/config/config.go config/config.yaml
git commit -m "feat: add configuration management"
```

---

## Task 3: LLM Gateway

**Files:**
- Create: `internal/llm/client.go`
- Create: `internal/llm/gateway.go`

- [ ] **Step 1: Write LLM client**

```go
package llm

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *Client) Chat(prompt string) (string, error) {
	encodedPrompt := url.PathEscape(prompt)
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, encodedPrompt)

	resp, err := c.http.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	return string(body), nil
}
```

- [ ] **Step 2: Write gateway interface**

```go
package llm

type Gateway interface {
	Chat(prompt string) (string, error)
}

type SimpleGateway struct {
	client *Client
}

func NewSimpleGateway(baseURL string) *SimpleGateway {
	return &SimpleGateway{
		client: NewClient(baseURL),
	}
}

func (g *SimpleGateway) Chat(prompt string) (string, error) {
	return g.client.Chat(prompt)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/llm/client.go internal/llm/gateway.go
git commit -m "feat: add llm gateway"
```

---

## Task 4: SQLite Memory Store

**Files:**
- Create: `internal/memory/schema.go`
- Create: `internal/memory/store.go`

- [ ] **Step 1: Write schema and migrations**

```go
package memory

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func initDB(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS conversations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
`
	_, err := db.Exec(schema)
	return err
}
```

- [ ] **Step 2: Write store implementation**

```go
package memory

import (
	"database/sql"
	"time"
)

type Message struct {
	ID            int64
	ConversationID int64
	Role          string
	Content       string
	Timestamp     time.Time
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
		ID:            id,
		ConversationID: convID,
		Role:          role,
		Content:       content,
		Timestamp:     time.Now(),
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
```

- [ ] **Step 3: Commit**

```bash
git add internal/memory/schema.go internal/memory/store.go
git commit -m "feat: add sqlite memory store"
```

---

## Task 5: Base Agent and Coder Agent

**Files:**
- Create: `internal/agent/base.go`
- Create: `internal/agent/coder.go`

- [ ] **Step 1: Write base agent**

```go
package agent

import (
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
)

type BaseAgent struct {
	name     string
	gateway  llm.Gateway
	store    *memory.Store
	convID   int64
}

func NewBaseAgent(name string, gateway llm.Gateway, store *memory.Store) (*BaseAgent, error) {
	conv, err := store.CreateConversation(name + " conversation")
	if err != nil {
		return nil, err
	}

	return &BaseAgent{
		name:    name,
		gateway: gateway,
		store:   store,
		convID:  conv.ID,
	}, nil
}

func (a *BaseAgent) Think(prompt string) (string, error) {
	fullPrompt := a.buildPrompt(prompt)
	response, err := a.gateway.Chat(fullPrompt)
	if err != nil {
		return "", err
	}

	a.store.AddMessage(a.convID, "user", fullPrompt)
	a.store.AddMessage(a.convID, "assistant", response)

	return response, nil
}

func (a *BaseAgent) buildPrompt(userPrompt string) string {
	messages, _ := a.store.GetMessages(a.convID)
	var history string
	for _, m := range messages {
		history += m.Role + ": " + m.Content + "\n"
	}
	return history + "user: " + userPrompt
}
```

- [ ] **Step 2: Write coder agent**

```go
package agent

import (
	"fmt"
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
)

type CoderAgent struct {
	base *BaseAgent
}

func NewCoderAgent(gateway llm.Gateway, store *memory.Store) (*CoderAgent, error) {
	base, err := NewBaseAgent("Coder", gateway, store)
	if err != nil {
		return nil, err
	}
	return &CoderAgent{base: base}, nil
}

func (c *CoderAgent) WriteCode(task string) (string, error) {
	prompt := fmt.Sprintf(`You are a professional software engineer. Please write high-quality code for the following task.

Task: %s

Please provide:
1. The complete code
2. Explanation of how it works
3. Usage examples

Use markdown for code blocks.`, task)

	return c.base.Think(prompt)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/agent/base.go internal/agent/coder.go
git commit -m "feat: add base agent and coder agent"
```

---

## Task 6: Filesystem Tools

**Files:**
- Create: `internal/tools/filesystem.go`

- [ ] **Step 1: Write filesystem tools**

```go
package tools

import (
	"os"
	"path/filepath"
)

type FileSystem struct{}

func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

func (fs *FileSystem) WriteFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func (fs *FileSystem) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (fs *FileSystem) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tools/filesystem.go
git commit -m "feat: add filesystem tools"
```

---

## Task 7: CLI Entry Point

**Files:**
- Create: `cmd/free-agent/main.go`

- [ ] **Step 1: Write main.go with Cobra**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vibe-coding/free-agent/internal/agent"
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
	"github.com/vibe-coding/free-agent/pkg/config"
)

var rootCmd = &cobra.Command{
	Use:   "free-agent",
	Short: "A multi-AI agent system for software engineering",
	Long:  `Free Agent is a harness-style multi-AI agent system for coding, testing, and managing git repositories.`,
}

var chatCmd = &cobra.Command{
	Use:   "chat [prompt]",
	Short: "Chat with the AI",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runChat,
}

var codeCmd = &cobra.Command{
	Use:   "code [task]",
	Short: "Ask the Coder agent to write code",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCode,
}

func init() {
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(codeCmd)
}

func loadConfig() (*config.Config, error) {
	cfgPath := config.DefaultPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return &config.Config{
			LLM: struct {
				BaseURL string `yaml:"base_url"`
			}{BaseURL: "https://818233.xyz"},
			Storage: struct {
				Path string `yaml:"path"`
			}{Path: "./data/free-agent.db"},
		}, nil
	}
	return config.Load(cfgPath)
}

func runChat(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	gateway := llm.NewSimpleGateway(cfg.LLM.BaseURL)
	response, err := gateway.Chat(args[0])
	if err != nil {
		return err
	}

	fmt.Println(response)
	return nil
}

func runCode(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.Storage.Path), 0755); err != nil {
		return err
	}

	store, err := memory.NewStore(cfg.Storage.Path)
	if err != nil {
		return err
	}
	defer store.Close()

	gateway := llm.NewSimpleGateway(cfg.LLM.BaseURL)
	coder, err := agent.NewCoderAgent(gateway, store)
	if err != nil {
		return err
	}

	response, err := coder.WriteCode(args[0])
	if err != nil {
		return err
	}

	fmt.Println(response)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build and test**

Run: `go build -o bin/free-agent.exe ./cmd/free-agent`
Expected: Builds executable in `bin/`

Run: `.\bin\free-agent.exe chat "Hello"`
Expected: Returns a greeting

- [ ] **Step 3: Commit**

```bash
git add cmd/free-agent/main.go
git commit -m "feat: add cli entry point"
```

---

## Task 8: Project Documentation

**Files:**
- Create: `README.md`
- Create: `.gitignore`

- [ ] **Step 1: Write .gitignore**

```
bin/
data/
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
go.work
.vscode/
.idea/
```

- [ ] **Step 2: Write README.md**

```markdown
# Free Agent

A harness-style multi-AI agent system for software engineering.

## Features

- 🤖 Multiple specialized AI agents
- 💾 SQLite-based memory and conversation history
- 🔌 Pluggable tool system
- 🚀 CLI-first interface

## Quick Start

```bash
# Build
go build -o bin/free-agent.exe ./cmd/free-agent

# Chat
.\bin\free-agent.exe chat "Hello, world!"

# Ask Coder agent
.\bin\free-agent.exe code "Write a function to reverse a string in Go"
```

## Configuration

Create `~/.free-agent/config.yaml`:

```yaml
llm:
  base_url: "https://818233.xyz"
storage:
  path: "./data/free-agent.db"
```

## License

MIT
```

- [ ] **Step 3: Commit**

```bash
git add README.md .gitignore
git commit -m "docs: add readme and gitignore"
```

---

## Final Check

- [ ] Verify all files are created
- [ ] Run full build
- [ ] Test both `chat` and `code` commands
- [ ] Celebrate! 🎉

---

## Next Steps (Post-MVP)

1. Add more agents (Planner, Reviewer, Tester, etc.)
2. Implement orchestrator for multi-agent collaboration
3. Add Git integration
4. Add context compression
5. Add browser automation

