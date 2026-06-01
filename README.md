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
