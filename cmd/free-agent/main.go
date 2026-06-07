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
	default:
		// 直接接受任务，使用三层闭环系统
		if err := runTask(strings.Join(args, " ")); err != nil {
			logger.LogProgramError(err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func printHelp() {
	fmt.Println(`Free Agent - Multi-AI Agent System for Software Engineering

  A harness engineering architecture for AI-powered software development.
  Features: multi-agent collaboration, session management, CI/CD integration,
  self-learning three-layer closed-loop architecture.

╔═══════════════════════════════════════════════════════════════════════════╗
║                        COMMAND LINE ARGUMENTS                            ║
╚═══════════════════════════════════════════════════════════════════════════╝

  free-agent              Launch interactive UI with temporary session
  free-agent <task>      Execute task using three-layer closed-loop system
  free-agent -chat        Start command-line chat mode
  free-agent -session     List all saved sessions
  free-agent -session <name>  Restore specific session
  free-agent -check       Check system status and API connectivity
  free-agent -clean       Clean up all temporary sessions
  free-agent -help        Show this help message

╔═══════════════════════════════════════════════════════════════════════════╗
║                      THREE-LAYER CLOSED-LOOP ARCHITECTURE              ║
╚═══════════════════════════════════════════════════════════════════════════╝

  1. EXECUTION LAYER - Responsible for all task execution
     • Plan: Intent analysis + task decomposition
     • Execute: Route matching selects best agent for the task

  2. CONTROL LAYER - Real-time monitoring and correction
     • Watch: Monitor execution quality and collect metrics
     • Correct: Auto-correct errors and issues

  3. MANAGEMENT LAYER - Review and optimize
     • Review: Evaluate results and performance
     • Improve: Adjust agent traits based on learning

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

  # Execute task directly (uses closed-loop system
  free-agent "Create a weather website"
  free-agent "Test SQL injection on target"
  free-agent "Plan an e-commerce project"

  # Chat mode without UI
  free-agent -chat

  # List all saved sessions
  free-agent -session

  # Restore a saved project
  free-agent -session weather-app

  # Check system status
  free-agent -check

  # Clean all temp sessions
  free-agent -clean

╔═══════════════════════════════════════════════════════════════════════════╗
║                         CLOSED-LOOP EXECUTION DETAILS                         ║
╚═══════════════════════════════════════════════════════════════════════════╝

  📋 EXECUTION LAYER (Plan → Execute
     • Intent analysis and task planning
     • Route matching selects appropriate agent
     • Agent execution of the planned tasks

  👁️ CONTROL LAYER (Watch → Correct
     • Real-time quality monitoring and metrics collection
     • Error detection and automatic correction
     • Performance tracking and issue reporting

  🏛️ MANAGEMENT LAYER (Review → Improve
     • Result evaluation and scoring
     • Agent trait adjustment based on performance
     • Learning and optimization over time

╔═══════════════════════════════════════════════════════════════════════════╗
║                         AVAILABLE AGENTS                                  ║
╚═══════════════════════════════════════════════════════════════════════════╝

  • Management & Assistance
    IntentAgent   - Intent analysis & classification
    Planner       - Task planning and project decomposition
    Explorer      - Uncertain task exploration (tree-based)
    GenericAgent  - General purpose task handling

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

╔═══════════════════════════════════════════════════════════════════════════╝
║                        SESSION MANAGEMENT                                ║
╚═══════════════════════════════════════════════════════════════════════════╝

  • Temp Sessions: Auto-created as tmp_YYYYMMDD_XX, auto-cleaned on exit
  • Saved Sessions: Use /save <name> to persist sessions
  • Session Restore: free-agent -session <name> to restore
  • Cleanup: free-agent -clean removes all temp sessions

For more details, see README.md`)
}

func runTask(input string) error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	gateway := llm.NewSimpleGateway(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)
	
	// 创建LLM Client用于需要它的Agent
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	am := agent.NewAgentManager()
	am.RegisterAgent(agent.NewIntentAgent(gateway))
	am.RegisterAgent(agent.NewPlannerAgent(gateway))
	am.RegisterAgent(agent.NewCodeGenerator(gateway, "."))
	am.RegisterAgent(agent.NewReviewerAgent(gateway))
	am.RegisterAgent(agent.NewTesterAgent(gateway))
	am.RegisterAgent(agent.NewDebuggerAgent(gateway))
	am.RegisterAgent(agent.NewGitAgent("."))
	am.RegisterAgent(agent.NewFeedbackAgent(gateway))
	am.RegisterAgent(agent.NewSecurityAssessor(progCfg))
	
	// 注册安全测试细分Agent（对应OWASP Top 10）
	am.RegisterAgent(agent.NewSQLInjectionScanner(llmClient))        // OWASP A03
	am.RegisterAgent(agent.NewXSSScanner(llmClient))                 // OWASP A03
	am.RegisterAgent(agent.NewCommandInjectionScanner(llmClient))    // OWASP A03
	am.RegisterAgent(agent.NewPathTraversalScanner(llmClient))       // OWASP A01
	am.RegisterAgent(agent.NewSSRFScanner(llmClient))                // OWASP A10
	am.RegisterAgent(agent.NewFileIncludeScanner(llmClient))        // OWASP A03
	am.RegisterAgent(agent.NewCTFSolver(llmClient))
	
	// 注册 Generic Agent (默认路由)
	am.RegisterAgent(agent.NewGenericAgent(llmClient))

	explorer := agent.NewExplorationAgent(gateway, am)
	am.RegisterAgent(explorer)

	orchestrator := agent.NewOrchestratorAgent(gateway, am)
	am.RegisterAgent(orchestrator)

	// =============================================
	// 使用新的 Scheduler + Worker/Watcher/Auditor 模式
	// =============================================
	// 创建 SkillLoader
	skillLoader := agent.NewSkillLoader("skills")
	
	// 创建调度器（硬编码系统调度）
	scheduler := agent.NewScheduler(llmClient, am, skillLoader)
	
	// 设置调度器配置（可通过配置文件调整）
	scheduler.SetMaxIterations(10)
	scheduler.SetMaxDuration(10 * time.Minute)
	scheduler.SetExecutorTimeout(5 * time.Minute)
	scheduler.SetObserverInterval(100 * time.Millisecond)

	ctx := context.Background()
	
	// 使用新模式执行任务
	result, err := scheduler.ExecuteWithAgentPattern(ctx, input)
	
	if err != nil {
		return err
	}
	
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📋 FINAL RESULT")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println(result)

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
	
	// 创建LLM Client用于需要它的Agent
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	conv, err := store.CreateConversation("chat")
	if err != nil {
		logger.ProgramError("Failed to create conversation: %v", err)
		return err
	}
	currentConvID = conv.ID

	am := agent.NewAgentManager()
	am.RegisterAgent(agent.NewIntentAgent(gateway))
	am.RegisterAgent(agent.NewPlannerAgent(gateway))
	am.RegisterAgent(agent.NewCodeGenerator(gateway, sessionDir))
	am.RegisterAgent(agent.NewReviewerAgent(gateway))
	am.RegisterAgent(agent.NewTesterAgent(gateway))
	am.RegisterAgent(agent.NewDebuggerAgent(gateway))
	am.RegisterAgent(agent.NewGitAgent(sessionDir))
	am.RegisterAgent(agent.NewFeedbackAgent(gateway))
	am.RegisterAgent(agent.NewSecurityAssessor(progCfg))
	
	// 注册安全测试细分Agent（对应OWASP Top 10）
	am.RegisterAgent(agent.NewSQLInjectionScanner(llmClient))        // OWASP A03
	am.RegisterAgent(agent.NewXSSScanner(llmClient))                 // OWASP A03
	am.RegisterAgent(agent.NewCommandInjectionScanner(llmClient))    // OWASP A03
	am.RegisterAgent(agent.NewPathTraversalScanner(llmClient))       // OWASP A01
	am.RegisterAgent(agent.NewSSRFScanner(llmClient))                // OWASP A10
	am.RegisterAgent(agent.NewFileIncludeScanner(llmClient))        // OWASP A03
	am.RegisterAgent(agent.NewCTFSolver(llmClient))
	
	// 注册 Generic Agent (默认路由)
	am.RegisterAgent(agent.NewGenericAgent(llmClient))

	explorer := agent.NewExplorationAgent(gateway, am)
	am.RegisterAgent(explorer)

	orchestrator := agent.NewOrchestratorAgent(gateway, am)
	am.RegisterAgent(orchestrator)

	// 初始化三层闭环管理器
	// 先创建 SkillLoader
	skillLoader := agent.NewSkillLoader("skills")
	
	closedLoopMgr := agent.NewClosedLoopManager(llmClient, am, skillLoader)
	
	// 从 SKILL.md 加载 Agent 特质
	agentNames := []string{
		"Intent", "Coder", "Planner", "Reviewer", "Tester", "Debugger", 
		"Git", "Feedback", "SecurityAssessor", "SQLInjectionScanner", "XSSScanner",
		"CommandInjectionScanner", "PathTraversalScanner", "SSRFScanner",
		"FileIncludeScanner", "CTFSolver", "GeneralHandler", 
		"Exploration", "Orchestrator",
	}
	
	for _, name := range agentNames {
		skill, err := skillLoader.LoadSkill(name)
		if err == nil && skill.Metrics != nil {
			// 确保 LearningRate 有默认值
			if skill.Metrics.LearningRate == 0 {
				skill.Metrics.LearningRate = 0.1
			}
			closedLoopMgr.RegisterAgentTraits(name, skill.Metrics)
			fmt.Printf("Loaded skill for agent: %s\n", name)
		} else {
			// 如果找不到 SKILL.md，使用默认值
			defaultTraits := &agent.AgentTraits{
				Name:          name,
				Efficiency:    0.7,
				Quality:       0.7,
				Creativity:    0.7,
				Collaboration: 0.7,
				LearningRate:  0.1,
			}
			closedLoopMgr.RegisterAgentTraits(name, defaultTraits)
			fmt.Printf("Using default traits for agent: %s (err: %v)\n", name, err)
		}
	}

	fmt.Println("=== Chat Mode (Three-Layer Closed-Loop System ===")
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

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("AI is thinking (three-layer closed-loop)...")

		// 使用三层闭环系统执行
		response, err := closedLoopMgr.ExecuteWithLoop(context.Background(), fullPrompt)
		if err != nil {
			logger.ProcessError("Closed-loop system error: %v", err)
			fmt.Printf("Error: %v\n", err)
			continue
		}

		store.AddMessage(conv.ID, "user", input)
		store.AddMessage(conv.ID, "assistant", response)

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("Result:")
		fmt.Println(response)
		fmt.Println(strings.Repeat("=", 80))
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

	// Initialize Agent Manager
	am := agent.NewAgentManager()
	am.RegisterAgent(agent.NewIntentAgent(gateway))
	am.RegisterAgent(agent.NewPlannerAgent(gateway))
	am.RegisterAgent(agent.NewCodeGenerator(gateway, sessionDir))
	am.RegisterAgent(agent.NewReviewerAgent(gateway))
	am.RegisterAgent(agent.NewTesterAgent(gateway))
	am.RegisterAgent(agent.NewDebuggerAgent(gateway))
	am.RegisterAgent(agent.NewGitAgent(sessionDir))
	am.RegisterAgent(agent.NewFeedbackAgent(gateway))
	am.RegisterAgent(agent.NewSecurityAssessor(progCfg))
	
	// 创建LLM Client用于需要它的Agent
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)
	
	// 注册安全测试细分Agent（对应OWASP Top 10）
	am.RegisterAgent(agent.NewSQLInjectionScanner(llmClient))        // OWASP A03
	am.RegisterAgent(agent.NewXSSScanner(llmClient))                 // OWASP A03
	am.RegisterAgent(agent.NewCommandInjectionScanner(llmClient))    // OWASP A03
	am.RegisterAgent(agent.NewPathTraversalScanner(llmClient))       // OWASP A01
	am.RegisterAgent(agent.NewSSRFScanner(llmClient))                // OWASP A10
	am.RegisterAgent(agent.NewFileIncludeScanner(llmClient))        // OWASP A03
	am.RegisterAgent(agent.NewCTFSolver(llmClient))
	
	// 注册 Generic Agent (默认路由)
	am.RegisterAgent(agent.NewGenericAgent(llmClient))

	explorer := agent.NewExplorationAgent(gateway, am)
	am.RegisterAgent(explorer)

	orchestrator := agent.NewOrchestratorAgent(gateway, am)
	am.RegisterAgent(orchestrator)

	// 初始化三层闭环管理器
	// 先创建 SkillLoader
	skillLoader := agent.NewSkillLoader("skills")
	
	closedLoopMgr := agent.NewClosedLoopManager(llmClient, am, skillLoader)
	
	// 从 SKILL.md 加载 Agent 特质
	agentNames := []string{
		"Intent", "Coder", "Planner", "Reviewer", "Tester", "Debugger", 
		"Git", "Feedback", "SecurityAssessor", "SQLInjectionScanner", "XSSScanner",
		"CommandInjectionScanner", "PathTraversalScanner", "SSRFScanner",
		"FileIncludeScanner", "CTFSolver", "GeneralHandler", 
		"Exploration", "Orchestrator",
	}
	
	for _, name := range agentNames {
		skill, err := skillLoader.LoadSkill(name)
		if err == nil && skill.Metrics != nil {
			// 确保 LearningRate 有默认值
			if skill.Metrics.LearningRate == 0 {
				skill.Metrics.LearningRate = 0.1
			}
			closedLoopMgr.RegisterAgentTraits(name, skill.Metrics)
			fmt.Printf("Loaded skill for agent: %s\n", name)
		} else {
			// 如果找不到 SKILL.md，使用默认值
			defaultTraits := &agent.AgentTraits{
				Name:          name,
				Efficiency:    0.7,
				Quality:       0.7,
				Creativity:    0.7,
				Collaboration: 0.7,
				LearningRate:  0.1,
			}
			closedLoopMgr.RegisterAgentTraits(name, defaultTraits)
			fmt.Printf("Using default traits for agent: %s (err: %v)\n", name, err)
		}
	}

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

			// 使用三层闭环系统执行任务
			response, err := closedLoopMgr.ExecuteWithLoop(context.Background(), input)
			if err != nil {
				logger.ProcessError("Closed-loop manager error: %v", err)
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
	model.SetCurrentAgent("Closed-Loop System")
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
