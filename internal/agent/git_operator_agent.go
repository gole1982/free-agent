package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type GitOperatorAgent struct {
	workDir string
}

func NewGitOperatorAgent(workDir string) *GitOperatorAgent {
	return &GitOperatorAgent{workDir: workDir}
}

func (a *GitOperatorAgent) Name() string {
	return "GitOperator"
}

func (a *GitOperatorAgent) Description() string {
	return "Version control operations. Handles git commands like commit, push, pull, etc."
}

func (a *GitOperatorAgent) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", fmt.Errorf("no git command provided")
	}
	cmd := exec.CommandContext(ctx, "git", parts...)
	cmd.Dir = a.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git command failed: %v", err)
	}
	return "Git command executed successfully", nil
}
