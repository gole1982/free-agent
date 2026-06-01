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
