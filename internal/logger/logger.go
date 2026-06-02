package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vibe-coding/free-agent/pkg/config"
)

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Logger struct {
	file     *os.File
	log      *log.Logger
	logLevel LogLevel
	mu       sync.Mutex
}

var programLogger *Logger
var processLogger *Logger

func init() {
	cfg, err := config.LoadProgramConfig()
	if err != nil {
		fmt.Printf("Warning: Failed to load program config for logger: %v\n", err)
		return
	}

	logPath := cfg.ProgramLog.Name
	if !filepath.IsAbs(logPath) {
		exePath, _ := os.Executable()
		logPath = filepath.Join(filepath.Dir(exePath), logPath)
	}

	programLogger = NewLogger(logPath, parseLogLevel(cfg.ProgramLog.Level))
}

func parseLogLevel(levelStr string) LogLevel {
	switch levelStr {
	case "debug":
		return LevelDebug
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

func NewLogger(logPath string, level LogLevel) *Logger {
	err := os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		fmt.Printf("Warning: Failed to create log directory: %v\n", err)
	}

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Failed to open log file %s: %v\n", logPath, err)
		return &Logger{
			log:      log.New(os.Stderr, "", log.LstdFlags),
			logLevel: level,
		}
	}

	return &Logger{
		file:     file,
		log:      log.New(file, "", log.LstdFlags),
		logLevel: level,
	}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	if l.logLevel > LevelDebug {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.log.Printf("[DEBUG] "+format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	if l.logLevel > LevelInfo {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.log.Printf("[INFO] "+format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	if l.logLevel > LevelWarn {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.log.Printf("[WARN] "+format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.log.Printf("[ERROR] "+format, v...)
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func ProgramDebug(format string, v ...interface{}) {
	if programLogger != nil {
		programLogger.Debug(format, v...)
	}
}

func ProgramInfo(format string, v ...interface{}) {
	if programLogger != nil {
		programLogger.Info(format, v...)
	}
}

func ProgramWarn(format string, v ...interface{}) {
	if programLogger != nil {
		programLogger.Warn(format, v...)
	}
}

func ProgramError(format string, v ...interface{}) {
	if programLogger != nil {
		programLogger.Error(format, v...)
	}
}

func InitProcessLogger(projectDir string) error {
	cfg, err := config.LoadProjectConfig(projectDir)
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	logPath := filepath.Join(projectDir, cfg.ProcessLog.Name)
	
	err = os.MkdirAll(projectDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	processLogger = NewLogger(logPath, parseLogLevel(cfg.ProcessLog.Level))
	processLogger.Info("Process logger initialized for session: %s", projectDir)
	return nil
}

func ProcessDebug(format string, v ...interface{}) {
	if processLogger != nil {
		processLogger.Debug(format, v...)
	}
}

func ProcessInfo(format string, v ...interface{}) {
	if processLogger != nil {
		processLogger.Info(format, v...)
	}
}

func ProcessWarn(format string, v ...interface{}) {
	if processLogger != nil {
		processLogger.Warn(format, v...)
	}
}

func ProcessError(format string, v ...interface{}) {
	if processLogger != nil {
		processLogger.Error(format, v...)
	}
}

func CloseAll() {
	if programLogger != nil {
		programLogger.Close()
	}
	if processLogger != nil {
		processLogger.Close()
	}
}

func LogProgramStart() {
	ProgramInfo("========================================")
	ProgramInfo("Free Agent started at %s", time.Now().Format(time.RFC3339))
	ProgramInfo("========================================")
}

func LogProgramError(err error) {
	ProgramError("Program error: %v", err)
}

func LogProcessInput(input string) {
	ProcessInfo("User input: %s", truncate(input, 100))
}

func LogProcessAPIError(err error) {
	ProcessError("API error: %v", err)
}

func LogProcessInvalidInput(input string) {
	ProcessWarn("Invalid user input: %s", truncate(input, 100))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
