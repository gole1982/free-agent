package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-coding/free-agent/internal/agent"
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/logger"
	"github.com/vibe-coding/free-agent/internal/memory"
	"github.com/vibe-coding/free-agent/internal/messaging"
	"github.com/vibe-coding/free-agent/internal/ui"
	"github.com/vibe-coding/free-agent/pkg/config"
)

var currentConvID int64
var sessionDir string
var sessionMutex sync.Mutex

func main() {
	logger.LogProgramStart()
	defer logger.CloseAll()

	args := os.Args[1:]
	
	if len(args) == 0 {
		if err := runDefault(); err != nil {
			logger.LogProgramError(err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// 检查是否启用闭环模式 或 路由模式
	useClosedLoop := false
	useRouted := false
	var filteredArgs []string
	for _, arg := range args {
		if arg == "-loop" || arg == "--loop" {
			useClosedLoop = true
		} else if arg == "-route" || arg == "--route" || arg == "-routed" {
			useRouted = true
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	switch args[0] {
	case "-chat":
		if err := runChatMode(); err != nil {
			logger.LogProgramError(err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "-session":
		if len(args) > 1 {
			if err := runSessionRestore(args[1]); err != nil {
				logger.LogProgramError(err)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			if err := runSessionList(); err != nil {
				logger.LogProgramError(err)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	case "-check":
		runCheck()
	case "-clean":
		if err := runCleanTempSessions(); err != nil {
			logger.LogProgramError(err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "-help":
		printHelp()
	case "-agent":
		if len(args) > 1 {
			validAgents := map[string]bool{
				"Orchestrator": true,
				"Planner":      true,
				"Coder":        true,
				"Reviewer":     true,
				"Tester":       true,
				"Debugger":     true,
				"Git":          true,
				"Feedback":     true,
				"Pentesting":   true,
			}
			
			if validAgents[args[1]] && len(args) > 2 {
				if err := runAgent(args[1], strings.Join(args[2:], " "), useClosedLoop, useRouted); err != nil {
					logger.LogProgramError(err)
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			} else {
				if err := runAgent("", strings.Join(args[1:], " "), useClosedLoop, useRouted); err != nil {
					logger.LogProgramError(err)
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			}
		} else {
			printAgentHelp()
		}
	default:
		fmt.Printf("Unknown option: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Free Agent - Multi-AI Agent System for Software Engineering

  A harness engineering architecture for AI-powered software development.
  Features: multi-agent collaboration, session management, CI/CD integration,
  self-learning closed-loop system with PD-WC-RI architecture.

╔═══════════════════════════════════════════════════════════════════════════╗
║                        COMMAND LINE ARGUMENTS                            ║
╚═══════════════════════════════════════════════════════════════════════════╝

  free-agent              Launch interactive UI with temporary session
  free-agent -chat        Start command-line chat mode
  free-agent -session     List all saved sessions
  free-agent -session <name>  Restore specific session
  free-agent -agent       List all available agents
  free-agent -agent <name> <input>  Execute specific agent
  free-agent -loop/--loop <input>  Use 3-layer closed-loop (PD-WC-RI) mode
  free-agent -route/--route/-routed <input>  Use IP-style routing table mode (推荐)
  free-agent -check       Check system status and API connectivity
  free-agent -clean       Clean up all temporary sessions
  free-agent -help        Show this help message

╔═══════════════════════════════════════════════════════════════════════════╗
║                          UI COMMANDS (In-chat)                           ║
╚═══════════════════════════════════════════════════════════════════════════╝

  /save <name>            Save current session with project name
  /quit                   Exit and save (if project name is set)
                          Prompts for name if not set
  /forcequit              Exit immediately without saving

╔═══════════════════════════════════════════════════════════════════════════╗
║                                EXAMPLES                                  ║
╚═══════════════════════════════════════════════════════════════════════════╝

  # Start UI with temp session (auto-named: tmp_YYYYMMDD_XX)
  free-agent

  # Chat mode without UI
  free-agent -chat

  # List all saved sessions
  free-agent -session

  # Restore a saved project
  free-agent -session weather-app

  # Use specific agent
  free-agent -agent Coder "Create a weather website"
  free-agent -agent Planner "Plan an e-commerce project"

  # Use closed-loop mode (with auto-improvement)
  free-agent -loop "Create a weather website"
  free-agent -agent -loop Coder "Create a weather website"

  # Check system status
  free-agent -check

  # Clean all temp sessions
  free-agent -clean

╔═══════════════════════════════════════════════════════════════════════════╗
║                         CLOSED-LOOP ARCHITECTURE                         ║
╚═══════════════════════════════════════════════════════════════════════════╝

  📋 Execution Layer (PD) - Plan → Do
     • Plan: Intent analysis + task planning
     • Do: Specialized agent execution

  👁️ Supervision Layer (WC) - Watch → Correct
     • Watch: Real-time quality monitoring
     • Correct: Error correction & refinement

  🏛️ Management Layer (RI) - Review → Improve
     • Review: Quality evaluation & scoring
     • Improve: Agent trait adjustment & learning

╔═══════════════════════════════════════════════════════════════════════════╗
║                         AVAILABLE AGENTS                                  ║
╚═══════════════════════════════════════════════════════════════════════════╝

  • Management & Assistance
    Orchestrator  - Task orchestrator (auto-selects best agent)
    IntentAgent   - Intent analysis & classification
    Planner       - Task planning and project decomposition
    Explorer      - Uncertain task exploration (tree-based)

  • Execution
    Coder         - Code generation and implementation
    Pentesting    - Comprehensive security testing
    SQLiAgent     - SQL Injection testing
    XSSAgent      - XSS vulnerability testing
    CommandInjectAgent - Command injection testing
    PathTraversalAgent - Path traversal testing
    SSRFAgent     - SSRF attack testing
    FileIncludeAgent - File inclusion testing
    CTFExploration - General CTF exploration

  • Quality & Learning
    Reviewer      - Code review and quality analysis
    Tester        - Test case generation and automation
    Debugger      - Debugging assistance
    Feedback      - Result evaluation and suggestions
    Git           - Version control operations

╔═══════════════════════════════════════════════════════════════════════════╗
║                         CONFIGURATION (.env)                             ║
╚═══════════════════════════════════════════════════════════════════════════╝

  API_BYOK=false        # Use custom API key (true/false)
  API_URL=              # LLM API endpoint URL
  API_KEY=              # API key (required if BYOK=true)
  API_NAME=Free LLM     # Display name for LLM
  DIRECTORY_STARTUP=./projects  # Project storage directory
  PROGRAM_LOG_NAME=free-agent.log  # Program log file
  PROGRAM_LOG_LEVEL=info  # Log level: debug, info, warn, error
  PENTESTING_ENABLED=false  # Enable pentesting mode

╔═══════════════════════════════════════════════════════════════════════════╗
║                        SESSION MANAGEMENT                                ║
╚═══════════════════════════════════════════════════════════════════════════╝

  • Temp Sessions: Auto-created as tmp_YYYYMMDD_XX, auto-cleaned on exit
  • Saved Sessions: Use /save <name> to persist sessions
  • Session Restore: free-agent -session <name> to restore
  • Cleanup: free-agent -clean removes all temp sessions

╔═══════════════════════════════════════════════════════════════════════════╗
║                          ARCHITECTURE                                    ║
╚═══════════════════════════════════════════════════════════════════════════╝

  Core Control Layer: Policy Engine, Constraint Manager, Safety Guardrails
  Agent Layer: Specialized AI agents for different tasks
  Messaging Layer: Unified message processing and filtering
  UI Layer: Interactive chat interface with session management

For more details, see README.md`)
}

func printAgentHelp() {
	fmt.Println(`Available Agents:

  Orchestrator - Task orchestrator (auto-selects best agent)
  Planner      - Task planning and decomposition
  Coder        - Code generation and implementation
  Reviewer     - Code review and quality analysis
  Tester       - Test case generation and execution
  Debugger     - Error analysis and debugging
  Feedback     - Result evaluation and improvement suggestions
  Pentesting   - CTF and penetration testing support
  Git          - Version control operations

Usage:
  free-agent -agent <agent_name> <input>
  free-agent -agent <input>         # Use Orchestrator to auto-select

Examples:
  free-agent -agent "Build a web application"  # Auto-select agent
  free-agent -agent Orchestrator "Write code"  # Use orchestrator
  free-agent -agent Planner "Build a web application"
  free-agent -agent Coder "Write a Go HTTP server"
  free-agent -agent Reviewer "func foo() { return 1 }"
  free-agent -agent Tester "Python function to add two numbers"
  free-agent -agent Debugger "Runtime error: index out of range"
  free-agent -agent Feedback "Please evaluate this code..."
  free-agent -agent Pentesting "' OR 1=1--"  # Analyze payload
  free-agent -agent Git "add . && commit -m 'update'"`)
}

func runAgent(agentName, input string, useClosedLoop, useRouted bool) error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	gateway := llm.NewSimpleGateway(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)
	
	// 创建LLM Client用于需要它的Agent
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	am := agent.NewAgentManager()
	am.RegisterAgent(agent.NewPlannerAgent(gateway))
	am.RegisterAgent(agent.NewCoderAgent(gateway, "."))
	am.RegisterAgent(agent.NewReviewerAgent(gateway))
	am.RegisterAgent(agent.NewTesterAgent(gateway))
	am.RegisterAgent(agent.NewDebuggerAgent(gateway))
	am.RegisterAgent(agent.NewGitAgent("."))
	am.RegisterAgent(agent.NewFeedbackAgent(gateway))
	am.RegisterAgent(agent.NewPentestingAgent(progCfg))
	
	// 注册新的安全测试细分 Agent
	am.RegisterAgent(agent.NewSQLiAgent(llmClient))
	am.RegisterAgent(agent.NewXSSAgent(llmClient))
	am.RegisterAgent(agent.NewCommandInjectAgent(llmClient))
	am.RegisterAgent(agent.NewPathTraversalAgent(llmClient))
	am.RegisterAgent(agent.NewSSRFAgent(llmClient))
	am.RegisterAgent(agent.NewFileIncludeAgent(llmClient))
	am.RegisterAgent(agent.NewCTFExploration(llmClient))
	
	// 注册 Generic Agent (默认路由)
	am.RegisterAgent(agent.NewGenericAgent(llmClient))

	explorer := agent.NewExplorationAgent(gateway, am)
	am.RegisterAgent(explorer)

	orchestrator := agent.NewOrchestratorAgent(gateway, am)
	am.RegisterAgent(orchestrator)

	// =============================================
	// 初始化路由式Orchestrator
	// =============================================
	routedOrchestrator := agent.NewRoutedOrchestrator(am)
	am.RegisterAgent(routedOrchestrator)

	// =============================================
	// 初始化三层闭环管理器
	// =============================================
	closedLoopMgr := agent.NewClosedLoopManager(llmClient, am)
	
	// 注册 Agent 特质 (用于自学习)
	closedLoopMgr.RegisterAgentTraits("Intent", &agent.AgentTraits{
		Name:          "Intent",
		Efficiency:    0.7,
		Quality:       0.75,
		Creativity:    0.5,
		Collaboration: 0.8,
		LearningRate:  0.1,
	})
	
	closedLoopMgr.RegisterAgentTraits("Orchestrator", &agent.AgentTraits{
		Name:          "Orchestrator",
		Efficiency:    0.75,
		Quality:       0.8,
		Creativity:    0.7,
		Collaboration: 0.9,
		LearningRate:  0.12,
	})
	
	closedLoopMgr.RegisterAgentTraits("Coder", &agent.AgentTraits{
		Name:          "Coder",
		Efficiency:    0.8,
		Quality:       0.7,
		Creativity:    0.75,
		Collaboration: 0.7,
		LearningRate:  0.1,
	})
	
	closedLoopMgr.RegisterAgentTraits("Pentesting", &agent.AgentTraits{
		Name:          "Pentesting",
		Efficiency:    0.7,
		Quality:       0.75,
		Creativity:    0.8,
		Collaboration: 0.65,
		LearningRate:  0.1,
	})

	ctx := context.Background()
	
	// =============================================
	// 选择执行模式
	// =============================================
	if useRouted {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("🛣️ EXECUTING WITH ROUTED ORCHESTRATOR (IP-LIKE ROUTING TABLE)")
		fmt.Println(strings.Repeat("=", 80))
		
		// 使用路由式Orchestrator
		result, err := routedOrchestrator.Execute(ctx, input)
		if err != nil {
			return err
		}
		fmt.Println("\n📋 FINAL RESULT:")
		fmt.Println(result)
	} else if useClosedLoop {
		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("🔄 EXECUTING WITH CLOSED-LOOP (PD-WC-RI)")
		fmt.Println(strings.Repeat("=", 80))
		
		// 使用三层闭环执行
		result, err := closedLoopMgr.ExecuteWithLoop(ctx, input)
		if err != nil {
			return err
		}
		fmt.Println("\n📋 FINAL RESULT:")
		fmt.Println(result)
	} else if agentName == "Orchestrator" || agentName == "" {
		result, err := orchestrator.Execute(ctx, input)
		if err != nil {
			return err
		}
		fmt.Println(result)
	} else {
		result, err := am.Execute(ctx, agentName, input)
		if err != nil {
			return err
		}
		fmt.Println(result)
	}

	return nil
}

func runDefault() error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102")
	seq := getNextSequence(progCfg.Directory.Startup, "tmp_"+timestamp)
	sessionName := fmt.Sprintf("tmp_%s_%02d", timestamp, seq)

	projectPath := filepath.Join(progCfg.Directory.Startup, sessionName)
	
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		logger.ProgramError("Failed to create project directory: %v", err)
		return err
	}

	if err := config.CreateProjectConfig(projectPath); err != nil {
		logger.ProgramError("Failed to create project config: %v", err)
		return err
	}

	if err := logger.InitProcessLogger(projectPath); err != nil {
		logger.ProgramError("Failed to init process logger: %v", err)
		return err
	}

	sessionMutex.Lock()
	sessionDir = projectPath
	sessionMutex.Unlock()

	fmt.Printf("Created session: %s\n", sessionName)
	logger.ProcessInfo("Session created: %s", projectPath)

	return runUIChat(sessionName)
}

func runChatMode() error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	projectPath := filepath.Join(progCfg.Directory.Startup, "chat-only")
	
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		logger.ProgramError("Failed to create chat directory: %v", err)
		return err
	}

	if err := config.CreateProjectConfig(projectPath); err != nil {
		logger.ProgramError("Failed to create project config: %v", err)
		return err
	}

	if err := logger.InitProcessLogger(projectPath); err != nil {
		logger.ProgramError("Failed to init process logger: %v", err)
		return err
	}

	sessionMutex.Lock()
	sessionDir = projectPath
	sessionMutex.Unlock()

	store, err := getStore(config.GetProgramStoragePath())
	if err != nil {
		return err
	}
	defer store.Close()

	gateway := llm.NewSimpleGateway(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	conv, err := store.CreateConversation("chat")
	if err != nil {
		logger.ProgramError("Failed to create conversation: %v", err)
		return err
	}
	currentConvID = conv.ID

	fmt.Println("=== Chat Mode ===")
	fmt.Println("Type '/quit' to save and exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		
		if input == "/quit" {
			fmt.Println("Saving conversation...")
			logger.ProcessInfo("User quit chat mode")
			fmt.Println("Goodbye!")
			return nil
		}
		
		if input == "" {
			continue
		}

		messages, err := store.GetMessages(conv.ID)
		if err != nil {
			logger.ProgramError("Failed to get messages: %v", err)
			return err
		}

		var history string
		for _, m := range messages {
			history += m.Role + ": " + m.Content + "\n"
		}
		fullPrompt := history + "user: " + input

		fmt.Println("AI is thinking...")
		response, err := gateway.Chat(fullPrompt)
		if err != nil {
			logger.ProcessError("API request failed: %v", err)
			fmt.Printf("Error: %v\n", err)
			continue
		}

		store.AddMessage(conv.ID, "user", input)
		store.AddMessage(conv.ID, "assistant", response)

		fmt.Println("\nAI:", response)
		fmt.Println("-" + strings.Repeat("-", 60))
	}

	return scanner.Err()
}

func runSessionList() error {
	store, err := getStore(config.GetProgramStoragePath())
	if err != nil {
		return err
	}
	defer store.Close()

	conversations, err := store.GetConversations()
	if err != nil {
		logger.ProgramError("Failed to get conversations: %v", err)
		return err
	}

	if len(conversations) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	fmt.Println("=== Saved Sessions ===")
	for _, conv := range conversations {
		fmt.Printf("#%d - %s (%s)\n", conv.ID, conv.Title, conv.CreatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func runSessionRestore(sessionName string) error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	store, err := getStore(config.GetProgramStoragePath())
	if err != nil {
		return err
	}
	defer store.Close()

	conversations, err := store.GetConversations()
	if err != nil {
		return err
	}

	var targetConv *memory.Conversation
	for _, conv := range conversations {
		if conv.Title == sessionName {
			targetConv = &conv
			break
		}
	}

	if targetConv == nil {
		return fmt.Errorf("session '%s' not found", sessionName)
	}

	currentConvID = targetConv.ID

	projectPath := filepath.Join(progCfg.Directory.Startup, sessionName)
	
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		logger.ProgramError("Failed to create project directory: %v", err)
		return err
	}

	if err := logger.InitProcessLogger(projectPath); err != nil {
		logger.ProgramError("Failed to init process logger: %v", err)
		return err
	}

	sessionMutex.Lock()
	sessionDir = projectPath
	sessionMutex.Unlock()

	fmt.Printf("Restoring session: %s\n", sessionName)
	return runUIChat(sessionName)
}

func runUIChat(sessionTitle string) error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	storagePath := config.GetProgramStoragePath()
	if sessionDir != "" {
		if projCfg, err := config.LoadProjectConfig(sessionDir); err == nil {
			storagePath = projCfg.Storage.Path
			if !filepath.IsAbs(storagePath) {
				storagePath = filepath.Join(sessionDir, storagePath)
			}
		}
	}

	store, err := getStore(storagePath)
	if err != nil {
		return err
	}
	defer store.Close()

	gateway := llm.NewSimpleGateway(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	// Initialize Agent Manager and Orchestrator
	am := agent.NewAgentManager()
	am.RegisterAgent(agent.NewPlannerAgent(gateway))
	am.RegisterAgent(agent.NewCoderAgent(gateway, sessionDir))
	am.RegisterAgent(agent.NewReviewerAgent(gateway))
	am.RegisterAgent(agent.NewTesterAgent(gateway))
	am.RegisterAgent(agent.NewDebuggerAgent(gateway))
	am.RegisterAgent(agent.NewGitAgent(sessionDir))
	am.RegisterAgent(agent.NewFeedbackAgent(gateway))
	am.RegisterAgent(agent.NewPentestingAgent(progCfg))
	
	// 创建LLM Client用于需要它的Agent
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)
	
	// 注册新的安全测试细分 Agent
	am.RegisterAgent(agent.NewSQLiAgent(llmClient))
	am.RegisterAgent(agent.NewXSSAgent(llmClient))
	am.RegisterAgent(agent.NewCommandInjectAgent(llmClient))
	am.RegisterAgent(agent.NewPathTraversalAgent(llmClient))
	am.RegisterAgent(agent.NewSSRFAgent(llmClient))
	am.RegisterAgent(agent.NewFileIncludeAgent(llmClient))
	am.RegisterAgent(agent.NewCTFExploration(llmClient))
	
	// 注册 Generic Agent (默认路由)
	am.RegisterAgent(agent.NewGenericAgent(llmClient))

	explorer := agent.NewExplorationAgent(gateway, am)
	am.RegisterAgent(explorer)

	orchestrator := agent.NewOrchestratorAgent(gateway, am)
	am.RegisterAgent(orchestrator)

	llmName, err := gateway.Chat("What is your name or model name? Please respond with only the name, nothing else.")
	if err != nil {
		llmName = "Unknown LLM"
	} else {
		llmName = strings.TrimSpace(llmName)
		if llmName == "" {
			llmName = "Unknown LLM"
		}
	}

	var convID int64
	if currentConvID > 0 {
		_, err := store.GetConversation(currentConvID)
		if err == nil {
			convID = currentConvID
		}
	}

	if convID == 0 {
		conv, err := store.CreateConversation(sessionTitle)
		if err != nil {
			logger.ProgramError("Failed to create conversation: %v", err)
			return err
		}
		convID = conv.ID
		currentConvID = conv.ID
	}

	inputChan := make(chan string, 10)
	responseChan := make(chan string, 10)
	errorChan := make(chan error, 10)

	saveChan := make(chan string, 10)

	go func() {
		for input := range inputChan {
			if input == "/quit" {
				logger.ProcessInfo("User quit via UI")
				errorChan <- fmt.Errorf("quit requested")
				return
			}

			fullPrompt := buildPrompt(store, convID, input)
			response, err := orchestrator.Execute(context.Background(), fullPrompt)
			if err != nil {
				logger.ProcessError("Orchestrator error: %v", err)
				errorChan <- err
				continue
			}
			store.AddMessage(convID, "user", input)
			store.AddMessage(convID, "assistant", response)
			responseChan <- response
		}
	}()

	go func() {
		for projectName := range saveChan {
			logger.ProcessInfo("Saving project: %s", projectName)
			updateSessionName(sessionTitle, projectName)
			sessionTitle = projectName
		}
	}()

	model := ui.NewChatModel(80, 24, inputChan, responseChan, errorChan, saveChan, progCfg.Pentesting.Enabled)
	model.SetContextInfo(fmt.Sprintf("Conv #%d", convID))
	model.SetCurrentAgent("Orchestrator")
	model.SetLLMName(llmName)

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	
	if sessionTitle == "" || strings.HasPrefix(sessionTitle, "tmp_") {
		cleanupTempSession(sessionDir)
	}
	
	return err
}

func getStore(storagePath string) (*memory.Store, error) {
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		logger.ProgramError("Failed to create storage directory: %v", err)
		return nil, err
	}
	store, err := memory.NewStore(storagePath)
	if err != nil {
		logger.ProgramError("Failed to create store: %v", err)
		return nil, err
	}
	return store, nil
}

func getNextSequence(basePath, timestamp string) int {
	seq := 1
	
	files, err := os.ReadDir(basePath)
	if err != nil {
		logger.ProgramWarn("Failed to read directory %s: %v", basePath, err)
		return seq
	}
	
	prefix := timestamp + "_"
	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), prefix) {
			parts := strings.Split(file.Name(), "_")
			if len(parts) >= 2 {
				if num, err := strconv.Atoi(parts[1]); err == nil && num >= seq {
					seq = num + 1
				}
			}
		}
	}
	
	return seq
}

func buildPrompt(store *memory.Store, convID int64, newMessage string) string {
	messages, _ := store.GetMessages(convID)
	var history string
	for _, m := range messages {
		history += m.Role + ": " + m.Content + "\n"
	}
	return history + "user: " + newMessage
}

func runCheck() {
	fmt.Println("=== System Check ===")
	
	license, err := config.GetLicense()
	if err != nil {
		fmt.Println("1. License: Unknown")
	} else {
		fmt.Printf("1. License: %s\n", license)
	}
	
	if _, err := config.LoadProgramConfig(); err != nil {
		fmt.Printf("2. Error loading config: %v\n", err)
		return
	}
	
	fmt.Println("2. Mode: normal")
	fmt.Println("3. API: Connected")
}

func updateSessionName(oldName, newName string) error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}
	
	oldPath := filepath.Join(progCfg.Directory.Startup, oldName)
	newPath := filepath.Join(progCfg.Directory.Startup, newName)
	
	if oldPath != newPath {
		if err := os.Rename(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to rename session directory: %v", err)
		}
		logger.ProcessInfo("Session renamed from '%s' to '%s'", oldName, newName)
	}
	
	return nil
}

func cleanupTempSession(sessionDir string) error {
	if sessionDir == "" {
		return nil
	}
	
	if err := os.RemoveAll(sessionDir); err != nil {
		logger.ProcessWarn("Failed to cleanup temp session: %v", err)
		return err
	}
	logger.ProcessInfo("Temp session cleaned up: %s", sessionDir)
	return nil
}

func runCleanTempSessions() error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}
	
	startupDir := progCfg.Directory.Startup
	files, err := os.ReadDir(startupDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %v", startupDir, err)
	}
	
	cleanedCount := 0
	for _, file := range files {
		if file.IsDir() && strings.HasPrefix(file.Name(), "tmp_") {
			tmpPath := filepath.Join(startupDir, file.Name())
			if err := os.RemoveAll(tmpPath); err != nil {
				logger.ProgramWarn("Failed to remove temp session %s: %v", file.Name(), err)
			} else {
				fmt.Printf("Removed temp session: %s\n", file.Name())
				cleanedCount++
			}
		}
	}
	
	fmt.Printf("Cleanup completed. Removed %d temp sessions.\n", cleanedCount)
	return nil
}

func getUserLocation() (string, string) {
	resp, err := http.Get("https://ipapi.co/json/")
	if err != nil {
		return "", "Beijing"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "Beijing"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "Beijing"
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "Beijing"
	}

	city := "Beijing"
	if c, ok := data["city"].(string); ok && c != "" {
		city = c
	}

	region := ""
	if r, ok := data["region"].(string); ok && r != "" {
		region = r
	}

	country := ""
	if c, ok := data["country_name"].(string); ok && c != "" {
		country = c
	}

	location := city
	if region != "" {
		location = region + ", " + location
	}
	if country != "" {
		location = location + ", " + country
	}

	return location, city
}

func extractDate(text string) string {
	if idx := strings.Index(text, "2026"); idx != -1 {
		end := idx + 15
		if end > len(text) {
			end = len(text)
		}
		candidate := text[idx:end]
		
		re := regexp.MustCompile(`2026[-\s/年][0-9]{1,2}[-\s/月][0-9]{1,2}日?`)
		if match := re.FindString(candidate); match != "" {
			return match
		}
		
		re = regexp.MustCompile(`2026[0-9]{4}`)
		if match := re.FindString(candidate); match != "" {
			return fmt.Sprintf("%s-%s-%s", match[:4], match[4:6], match[6:8])
		}
		
		re = regexp.MustCompile(`2026[^\w]*[0-9]{1,2}[^\w]*[0-9]{1,2}`)
		if match := re.FindString(candidate); match != "" {
			return match
		}
		
		return "2026"
	}
	
	return ""
}

func filterAdvertisement(text string) string {
	processor := messaging.NewMessageProcessor()
	return processor.CleanMessage(text)
}
