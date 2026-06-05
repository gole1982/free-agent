package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GitAgent struct {
	workDir string
}

func NewGitAgent(workDir string) *GitAgent {
	return &GitAgent{workDir: workDir}
}

func (a *GitAgent) Name() string {
	return "Git"
}

func (a *GitAgent) Description() string {
	return "Version control operations. Handles git commands like commit, push, pull, etc."
}

func (a *GitAgent) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", fmt.Errorf("no git command provided")
	}

	cmd := exec.CommandContext(ctx, "git", parts...)
	cmd.Dir = a.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git command failed: %v", err)
	}

	return "Git command executed successfully", nil
}
