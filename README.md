# Free Agent

A harness-style multi-AI agent system for software engineering.

## Features

- 🤖 Multiple specialized AI agents
- 💾 SQLite-based memory and conversation history
- 🔌 Pluggable tool system
- 🚀 CLI-first interface
- 🎨 Interactive terminal UI
- 🔄 Multi-turn conversation support
- 📊 Real-time system status monitoring
- 🔒 Separate program and project configurations
- 🔄 **Executor/Observer/Evaluator Architecture** - Clear separation between hardcoded scheduling and agent intelligence
- 📝 **Skill-Driven Configuration** - Agent behaviors defined in SKILL.md files
- ⚙️ **Self-Learning System** - Agents optimize traits based on execution feedback

## Architecture Overview

Free Agent uses a clear boundary between hardcoded system functions and agent-driven intelligence:

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Scheduler (Hardcoded)                           │
│  System scheduling, timing, lifecycle management, channel management │
│  Delegates to TaskCoordinator for business routing                   │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│              TaskCoordinator (Agent)                                 │
│  Intent analysis, task decomposition, multi-agent coordination      │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
        ┌────────────────────────┼────────────────────────┐
        ↓                        ↓                        ↓
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Executors      │    │  Observer       │    │  Evaluator      │
│  (Agents)       │    │  (Agent)        │    │  (Agent)        │
│ - IntentAnalyzer│    │ - Intent check  │    │ - Result review │
│ - TaskPlanner   │    │ - Deadloop det  │    │ - Policy update │
│ - CodeGenerator │    │ - Honeypot det  │    │ - Trait adjust  │
│ - SecurityAssess│    │ - Malicious det │    │                 │
│ - SQLi/XSS/...  │    │ - Stop & guide  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Hardcoded vs Agent Boundary

| System Function | Implementation | Reason |
|-----------------|----------------|--------|
| Timing & Timeouts | **Hardcoded** | System infrastructure |
| Scheduled Tasks | **Hardcoded** | Fixed logic |
| Lifecycle Management | **Hardcoded** | Deterministic process |
| Channel/Flow Management | **Hardcoded** | Infrastructure |
| Intent Monitoring | **Agent** | Requires LLM judgment |
| Deadloop Detection | **Agent** | Context understanding needed |
| Honeypot Detection | **Agent** | Intelligent recognition |
| Result Review | **Agent** | Quality assessment needed |
| Policy Updates | **Agent** | Learning and adaptation |
| Business Execution | **Agent** | Domain intelligence |

### Skill Configuration

Agent behaviors are defined in `SKILL.md` files under the `skills/` directory:
- **Agent Character** - Role description
- **Core Capabilities** - What the agent can do
- **Workflow** - How the agent operates
- **Quality Metrics** - Efficiency, Quality, Creativity, Collaboration scores

## Quick Start

```bash
# Build
go build -o bin/free-agent.exe ./cmd/free-agent

# Launch UI with default session
.\bin\free-agent.exe

# Chat-only mode
.\bin\free-agent.exe -chat

# List saved sessions
.\bin\free-agent.exe -session

# Restore specific session
.\bin\free-agent.exe -session myproject

# Show help
.\bin\free-agent.exe -help
```

## Usage

### Command Line Arguments

| Command | Description |
|---------|-------------|
| `free-agent` | Start interactive UI with auto-generated session name (date+time+sequence) |
| `free-agent -chat` | Start chat-only mode (session: chat, project: chat-only) |
| `free-agent -session` | List all saved sessions with name and description |
| `free-agent -session <name>` | Restore a specific session by name |
| `free-agent -check` | Check system status and API connection |
| `free-agent -help` | Show help message with usage examples |

### System Check (`-check`)

The `-check` command performs three verification tasks:

1. **License Check**: Dynamically reads and displays the license from LICENSE file
2. **LLM Check**: Displays the configured LLM name
3. **API Connectivity Test**: Sends "What is today's date?" request and extracts date from response

**Output Format**:
```
=== System Check ===
1. License: MIT License
2. LLM: Free LLM
3. Remote Service: Normal/Abnormal
4. [extracted date string] / API Response: [response]
```

### Session Naming

- **Default**: `YYYYMMDD_HHMMSS_XX` (date + time + 2-digit sequence)
- **Chat mode**: `chat-only` (fixed project name)
- **Restored**: Uses session name from saved conversation

### In-chat Commands

- `/quit`: Save conversation and exit

## Configuration

### Program Configuration

Create `.env` file in the application directory with:

- `API_BYOK`: `"true"` to use custom API key, `"false"` for default API
- `API_URL`: LLM API endpoint URL
- `API_KEY`: API authentication key (required if BYOK=true)
- `DIRECTORY_STARTUP`: Parent directory for project sessions
- `PROGRAM_LOG_NAME`: Program log file name
- `PROGRAM_LOG_LEVEL`: Log level (debug, info, warn, error)

### Project Configuration

Automatically generated in project directory when creating sessions:

- `REPOSITORY_URL`: Remote git repository URL
- `REPOSITORY_KEY`: Git authentication key
- `STORAGE_PATH`: SQLite database path
- `PROCESS_LOG_NAME`: Session log file name
- `PROCESS_LOG_LEVEL`: Session log level

### Configuration Requirements

1. Program configuration must exist before running
2. Project configuration is auto-generated
3. All paths support relative and absolute formats

## Project Structure

```
free-agent/
├── .env                      # Program configuration
├── DESIGN.md                  # Complete system design and requirements
├── README.md                  # This file
├── AGENT_NAMING.md            # Agent naming conventions
├── .github/workflows/ci-cd.yml  # CI/CD pipeline
├── cmd/free-agent/main.go     # Main entry point
├── internal/
│   ├── agent/                 # AI agents
│   │   ├── scheduler.go       # Scheduler (hardcoded orchestration)
│   │   ├── task_coordinator_agent.go  # TaskCoordinator (business routing)
│   │   ├── observer_agent.go  # Observer (control agent)
│   │   ├── evaluator_agent.go # Evaluator (management agent)
│   │   ├── skill_loader.go    # SKILL.md parser
│   │   └── *_agent.go         # Executor agents (CodeGenerator, IntentAnalyzer, etc.)
│   ├── llm/                   # LLM client
│   ├── logger/                # Logging system
│   ├── memory/                # Conversation storage (SQLite)
│   ├── messaging/             # Message cleaning
│   ├── sandbox/               # Sandbox manager, policy engine, audit, snapshots
│   ├── tools/                 # Tool system
│   ├── vds/                   # Vulnerability Discovery System (6-phase framework)
│   │   └── tools/             # Tool adapters (sqlmap, ZAP, Nmap)
│   └── ui/                    # Terminal UI (Bubbletea)
├── skills/                    # Agent SKILL.md files
│   ├── coding/                # CodeGenerator, CodeReviewer, TestEngineer, DebugAnalyst
│   ├── control/               # Observer
│   ├── management/            # Evaluator, IntentAnalyzer, TaskCoordinator, FeedbackCollector
│   ├── planning/              # TaskPlanner, SolutionExplorer
│   ├── security/              # SecurityAssessor + OWASP scanners
│   ├── tools/                 # GitOperator
│   └── general/               # GeneralHandler
├── pkg/config/                # Configuration management
└── projects/                  # Project sessions directory
```

## CI/CD

Automated pipeline via GitHub Actions:
- Cross-platform builds (Linux, Windows, macOS)
- Automated testing
- GitHub Releases on main branch push

## License

MIT
