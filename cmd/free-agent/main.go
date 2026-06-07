package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-coding/free-agent/internal/agent"
	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/logger"
	"github.com/vibe-coding/free-agent/internal/memory"
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
		if err := runTask(strings.Join(args, " ")); err != nil {
			logger.LogProgramError(err)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func printHelp() {
	fmt.Println(`Free Agent - Multi-AI Agent System for Software Engineering

Usage:
  free-agent                    Launch interactive TUI with auto-generated temp session
  free-agent <task>             Execute a single task using the Scheduler
  free-agent -chat              REPL chat mode (no UI)
  free-agent -session [name]    List or restore a saved session
  free-agent -check             System check + API connectivity
  free-agent -clean             Remove all temp sessions
  free-agent -help              Show this help

Architecture (Scheduler + Worker/Observer/Evaluator):
  Scheduler    - hardcoded system scheduling, lifecycle, channels
  Worker       - business agent (intent-routed)
  Observer     - control agent (intent alignment, dead loop, honeypot)
  Evaluator    - management agent (result review, policy update)

Available Agents:
  IntentAnalyzer, TaskPlanner, CodeGenerator, CodeReviewer,
  TestEngineer, DebugAnalyst, GitOperator, FeedbackCollector,
  SecurityAssessor, SQLInjectionScanner, XSSScanner,
  CommandInjectionScanner, PathTraversalScanner, SSRFScanner,
  FileIncludeScanner, CryptographicFailuresScanner,
  InsecureDesignScanner, SecurityMisconfigurationScanner,
  VulnerableComponentsScanner, AuthenticationFailuresScanner,
  SoftwareIntegrityScanner, LoggingFailuresScanner,
  CTFSolver, SolutionExplorer, GeneralHandler, TaskCoordinator

For more details, see README.md and DESIGN.md.`)
}

func registerAgents(am *agent.AgentManager, gateway *llm.SimpleGateway, llmClient *llm.Client, workDir string) {
	am.RegisterAgent(agent.NewIntentAnalyzerAgent(gateway))
	am.RegisterAgent(agent.NewTaskPlannerAgent(gateway))
	am.RegisterAgent(agent.NewCodeGeneratorAgent(gateway, workDir))
	am.RegisterAgent(agent.NewCodeReviewerAgent(gateway))
	am.RegisterAgent(agent.NewTestEngineerAgent(gateway))
	am.RegisterAgent(agent.NewDebugAnalystAgent(gateway))
	am.RegisterAgent(agent.NewGitOperatorAgent(workDir))
	am.RegisterAgent(agent.NewFeedbackCollectorAgent(gateway))
	am.RegisterAgent(agent.NewSecurityAssessorAgent(am, llmClient))

	am.RegisterAgent(agent.NewSQLInjectionScanner(llmClient))
	am.RegisterAgent(agent.NewXSSScanner(llmClient))
	am.RegisterAgent(agent.NewCommandInjectionScanner(llmClient))
	am.RegisterAgent(agent.NewPathTraversalScanner(llmClient))
	am.RegisterAgent(agent.NewSSRFScanner(llmClient))
	am.RegisterAgent(agent.NewFileIncludeScanner(llmClient))
	am.RegisterAgent(agent.NewCryptographicFailuresScanner(llmClient))
	am.RegisterAgent(agent.NewInsecureDesignScanner(llmClient))
	am.RegisterAgent(agent.NewSecurityMisconfigurationScanner(llmClient))
	am.RegisterAgent(agent.NewVulnerableComponentsScanner(llmClient))
	am.RegisterAgent(agent.NewAuthenticationFailuresScanner(llmClient))
	am.RegisterAgent(agent.NewSoftwareIntegrityScanner(llmClient))
	am.RegisterAgent(agent.NewLoggingFailuresScanner(llmClient))
	am.RegisterAgent(agent.NewCTFSolver(llmClient))

	am.RegisterAgent(agent.NewGeneralHandlerAgent(llmClient))
	am.RegisterAgent(agent.NewSolutionExplorerAgent(gateway, am))
	am.RegisterAgent(agent.NewTaskCoordinatorAgent(gateway, am))
}

func newScheduler(llmClient *llm.Client, am *agent.AgentManager, store *memory.Store) *agent.Scheduler {
	skillLoader := agent.NewSkillLoader("skills")
	scheduler := agent.NewScheduler(llmClient, am, skillLoader, store)
	scheduler.SetMaxIterations(10)
	scheduler.SetMaxDuration(10 * time.Minute)
	scheduler.SetExecutorTimeout(5 * time.Minute)
	scheduler.SetObserverInterval(100 * time.Millisecond)
	return scheduler
}

func runTask(input string) error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	gateway := llm.NewSimpleGateway(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	store, err := getStore(config.GetProgramStoragePath())
	if err != nil {
		return err
	}
	defer store.Close()

	am := agent.NewAgentManager()
	registerAgents(am, gateway, llmClient, ".")

	scheduler := newScheduler(llmClient, am, store)
	result, err := scheduler.ExecuteWithAgentPattern(context.Background(), input)
	if err != nil {
		return err
	}
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
		return err
	}
	if err := config.CreateProjectConfig(projectPath); err != nil {
		return err
	}
	if err := logger.InitProcessLogger(projectPath); err != nil {
		return err
	}

	sessionMutex.Lock()
	sessionDir = projectPath
	sessionMutex.Unlock()

	fmt.Printf("Created session: %s\n", sessionName)
	return runUIChat(sessionName)
}

func runChatMode() error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	projectPath := filepath.Join(progCfg.Directory.Startup, "chat-only")
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return err
	}
	if err := config.CreateProjectConfig(projectPath); err != nil {
		return err
	}
	if err := logger.InitProcessLogger(projectPath); err != nil {
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
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	am := agent.NewAgentManager()
	registerAgents(am, gateway, llmClient, projectPath)

	scheduler := newScheduler(llmClient, am, store)

	conv, err := store.CreateConversation("chat")
	if err != nil {
		return err
	}
	currentConvID = conv.ID

	fmt.Println("=== Chat Mode (Scheduler + Worker/Observer/Evaluator) ===")
	fmt.Println("Type '/quit' to exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "/quit" {
			return nil
		}
		if input == "" {
			continue
		}

		messages, _ := store.GetMessages(conv.ID)
		var history strings.Builder
		for _, m := range messages {
			history.WriteString(m.Role + ": " + m.Content + "\n")
		}
		fullPrompt := history.String() + "user: " + input

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Println("AI is thinking...")

		response, err := scheduler.ExecuteWithAgentPattern(context.Background(), fullPrompt)
		if err != nil {
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

	for _, conv := range conversations {
		if conv.Title == sessionName {
			currentConvID = conv.ID
			fmt.Printf("Restoring session: %s\n", sessionName)
			break
		}
	}

	projectPath := filepath.Join(progCfg.Directory.Startup, sessionName)
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return err
	}
	if err := config.CreateProjectConfig(projectPath); err != nil {
		return err
	}
	if err := logger.InitProcessLogger(projectPath); err != nil {
		return err
	}

	sessionMutex.Lock()
	sessionDir = projectPath
	sessionMutex.Unlock()

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
	llmClient := llm.NewClient(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)

	am := agent.NewAgentManager()
	registerAgents(am, gateway, llmClient, sessionDir)

	scheduler := newScheduler(llmClient, am, store)

	llmName, err := gateway.Chat("What is your name or model name? Respond with only the name.")
	if err != nil || strings.TrimSpace(llmName) == "" {
		llmName = "Unknown LLM"
	} else {
		llmName = strings.TrimSpace(llmName)
	}

	var convID int64
	if currentConvID > 0 {
		if _, err := store.GetConversation(currentConvID); err == nil {
			convID = currentConvID
		}
	}
	if convID == 0 {
		conv, err := store.CreateConversation(sessionTitle)
		if err != nil {
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
				errorChan <- fmt.Errorf("quit requested")
				return
			}
			response, err := scheduler.ExecuteWithAgentPattern(context.Background(), input)
			if err != nil {
				logger.ProcessError("Scheduler error: %v", err)
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
			sessionTitle = projectName
		}
	}()

	model := ui.NewChatModel(80, 24, inputChan, responseChan, errorChan, saveChan, progCfg.Pentesting.Enabled)
	model.SetContextInfo(fmt.Sprintf("Conv #%d", convID))
	model.SetCurrentAgent("Scheduler")
	model.SetLLMName(llmName)
	model.SetProjectName(sessionTitle)

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()

	if sessionTitle == "" || strings.HasPrefix(sessionTitle, "tmp_") {
		cleanupTempSession(sessionDir)
	}
	return err
}

func getStore(storagePath string) (*memory.Store, error) {
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		return nil, err
	}
	return memory.NewStore(storagePath)
}

func getNextSequence(basePath, timestamp string) int {
	seq := 1
	files, err := os.ReadDir(basePath)
	if err != nil {
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

func runCheck() {
	fmt.Println("=== System Check ===")

	license, err := config.GetLicense()
	if err != nil {
		fmt.Println("1. License: Unknown")
	} else {
		fmt.Printf("1. License: %s\n", license)
	}

	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	userName := progCfg.User.Name
	if userName == "" {
		userName = "Guest"
	}

	gateway := llm.NewSimpleGateway(progCfg.API.URL, progCfg.API.Key, progCfg.API.BYOK)
	llmName, err := gateway.Chat("What is your name or model name? Respond with only the name.")
	if err != nil || strings.TrimSpace(llmName) == "" {
		llmName = "Unknown LLM"
	}
	llmName = strings.TrimSpace(llmName)

	dateFromLLM, err := gateway.Chat("What is today's date in YYYY-MM-DD format? Only respond with the date.")
	if err != nil || dateFromLLM == "" {
		dateFromLLM = time.Now().Format("2006-01-02")
	}
	dateFromLLM = strings.TrimSpace(dateFromLLM)

	fmt.Printf("3. Nice to meet you, %s. I am %s, today is %s.\n", userName, llmName, dateFromLLM)
}

func cleanupTempSession(sessionDir string) error {
	if sessionDir == "" {
		return nil
	}
	if err := os.RemoveAll(sessionDir); err != nil {
		logger.ProcessWarn("Failed to cleanup temp session: %v", err)
		return err
	}
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

var _ = http.Get
