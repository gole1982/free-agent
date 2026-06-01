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
