package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	case "-help":
		printHelp()
	default:
		fmt.Printf("Unknown option: %s\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Free Agent - Multi-AI Agent System for Software Engineering

Usage:
  free-agent              Start interactive UI with default session
  free-agent -chat        Start chat-only mode
  free-agent -session     List all saved sessions
  free-agent -session <name>  Restore a specific session
  free-agent -help        Show this help message

Examples:
  free-agent              # Launch UI with auto-generated session name
  free-agent -chat        # Start command-line chat mode
  free-agent -session     # List all sessions
  free-agent -session myproject  # Restore 'myproject' session

Configuration:
  Create .env file with:
    API_BYOK=false        # Use custom API key (true/false)
    API_URL=              # LLM API endpoint
    API_KEY=              # API key (required if BYOK=true)
    DIRECTORY_STARTUP=./projects  # Project parent directory

For more details, see README.md`)
}

func runDefault() error {
	progCfg, err := config.LoadProgramConfig()
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	seq := getNextSequence(progCfg.Directory.Startup, timestamp)
	sessionName := fmt.Sprintf("%s_%02d", timestamp, seq)

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

	go func() {
		for input := range inputChan {
			if input == "/quit" {
				logger.ProcessInfo("User quit via UI")
				errorChan <- fmt.Errorf("quit requested")
				return
			}

			fullPrompt := buildPrompt(store, convID, input)
			response, err := gateway.Chat(fullPrompt)
			if err != nil {
				logger.ProcessError("API error: %v", err)
				errorChan <- err
				continue
			}
			store.AddMessage(convID, "user", input)
			store.AddMessage(convID, "assistant", response)
			responseChan <- response
		}
	}()

	model := ui.NewChatModel(80, 24, inputChan, responseChan, errorChan)
	model.SetContextInfo(fmt.Sprintf("Conv #%d", convID))
	model.SetCurrentAgent("Default")

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
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
