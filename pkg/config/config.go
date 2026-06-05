package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type ProgramConfig struct {
	API struct {
		BYOK bool
		URL  string
		Key  string
		Name string
	}
	User struct {
		Name          string
		GreetingFormat string
	}
	Directory struct {
		Startup string
	}
	ProgramLog struct {
		Name  string
		Level string
	}
	Pentesting struct {
		Enabled bool
	}
}

type ProjectConfig struct {
	Repository struct {
		URL string
		Key string
	}
	Storage struct {
		Path string
	}
	ProcessLog struct {
		Name  string
		Level string
	}
}

var programConfig *ProgramConfig
var projectConfig *ProjectConfig

func LoadProgramConfig() (*ProgramConfig, error) {
	if programConfig != nil {
		return programConfig, nil
	}

	cfg := &ProgramConfig{
		API: struct {
			BYOK bool
			URL  string
			Key  string
			Name string
		}{
			BYOK: false,
			URL:  "https://818233.xyz",
			Key:  "",
			Name: "Free LLM",
		},
		User: struct {
			Name          string
			GreetingFormat string
		}{
			Name:          "User",
			GreetingFormat: "Hi $name! Today is $day, $date. Nice to meet you!",
		},
		Directory: struct {
			Startup string
		}{
			Startup: "./projects",
		},
		ProgramLog: struct {
			Name  string
			Level string
		}{
			Name:  "free-agent.log",
			Level: "info",
		},
		Pentesting: struct {
			Enabled bool
		}{
			Enabled: false,
		},
	}

	envPath := ".env"
	if _, err := os.Stat(envPath); err == nil {
		envMap, err := godotenv.Read(envPath)
		if err != nil {
			return nil, err
		}

		if val, ok := envMap["API_BYOK"]; ok {
			cfg.API.BYOK = strings.ToLower(val) == "true"
		}
		if val, ok := envMap["API_URL"]; ok {
			cfg.API.URL = val
		}
		if val, ok := envMap["API_KEY"]; ok {
			cfg.API.Key = val
		}
		if val, ok := envMap["API_NAME"]; ok {
			cfg.API.Name = val
		}
		if val, ok := envMap["USER_NAME"]; ok {
			cfg.User.Name = val
		}
		if val, ok := envMap["GREETING_FORMAT"]; ok {
			cfg.User.GreetingFormat = val
		}
		if val, ok := envMap["DIRECTORY_STARTUP"]; ok {
			cfg.Directory.Startup = val
		}
		if val, ok := envMap["PROGRAM_LOG_NAME"]; ok {
			cfg.ProgramLog.Name = val
		}
		if val, ok := envMap["PROGRAM_LOG_LEVEL"]; ok {
			cfg.ProgramLog.Level = val
		}
		if val, ok := envMap["PENTESTING_ENABLED"]; ok {
			cfg.Pentesting.Enabled = strings.ToLower(val) == "true"
		}
	}

	programConfig = cfg
	return cfg, nil
}

// GetProgramStoragePath 获取程序默认存储路径
func GetProgramStoragePath() string {
	return "./data/free-agent.db"
}

// GetLicense 获取许可证信息
func GetLicense() (string, error) {
	data, err := os.ReadFile("LICENSE")
	if err != nil {
		return "", err
	}
	
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		return lines[0], nil
	}
	return "MIT License", nil
}

func LoadProjectConfig(projectDir string) (*ProjectConfig, error) {
	if projectConfig != nil {
		return projectConfig, nil
	}

	cfg := &ProjectConfig{
		Repository: struct {
			URL string
			Key string
		}{
			URL: "",
			Key: "",
		},
		Storage: struct {
			Path string
		}{
			Path: "./data/free-agent.db",
		},
		ProcessLog: struct {
			Name  string
			Level string
		}{
			Name:  "session.log",
			Level: "info",
		},
	}

	envPath := filepath.Join(projectDir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		envMap, err := godotenv.Read(envPath)
		if err != nil {
			return nil, err
		}

		if val, ok := envMap["REPOSITORY_URL"]; ok {
			cfg.Repository.URL = val
		}
		if val, ok := envMap["REPOSITORY_KEY"]; ok {
			cfg.Repository.Key = val
		}
		if val, ok := envMap["STORAGE_PATH"]; ok {
			cfg.Storage.Path = val
		}
		if val, ok := envMap["PROCESS_LOG_NAME"]; ok {
			cfg.ProcessLog.Name = val
		}
		if val, ok := envMap["PROCESS_LOG_LEVEL"]; ok {
			cfg.ProcessLog.Level = val
		}
	}

	projectConfig = cfg
	return cfg, nil
}

func CreateProjectConfig(projectDir string) error {
	envPath := filepath.Join(projectDir, ".env")
	content := `# Project Configuration
REPOSITORY_URL=
REPOSITORY_KEY=
STORAGE_PATH=./data/free-agent.db
PROCESS_LOG_NAME=session.log
PROCESS_LOG_LEVEL=info
`
	return os.WriteFile(envPath, []byte(content), 0644)
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".free-agent", "config.yaml")
}
