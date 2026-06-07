package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibe-coding/free-agent/internal/logger"
)

type SandboxConfig struct {
	Image          string
	WorkDir        string
	HostWorkDir    string
	NetworkMode    string
	MemoryLimit    string
	CPUQuota       int64
	ReadOnlyRootFS bool
	Timeout        time.Duration
	EnvVars        map[string]string
	Volumes        map[string]string
	Capabilities   []string
	SecurityOpts   []string
}

type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

type Sandbox struct {
	config SandboxConfig
}

func NewSandbox(config SandboxConfig) *Sandbox {
	if config.Image == "" {
		config.Image = "alpine:latest"
	}
	if config.WorkDir == "" {
		config.WorkDir = "/workspace"
	}
	if config.NetworkMode == "" {
		config.NetworkMode = "none"
	}
	if config.MemoryLimit == "" {
		config.MemoryLimit = "512m"
	}
	if config.CPUQuota == 0 {
		config.CPUQuota = 50000
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	return &Sandbox{config: config}
}

func (s *Sandbox) Execute(ctx context.Context, command string, args ...string) *ExecutionResult {
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	containerName := fmt.Sprintf("free-agent-sandbox-%d", time.Now().UnixNano())
	
	dockerArgs := s.buildDockerArgs(containerName, command, args...)
	
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &ExecutionResult{Error: fmt.Errorf("stdout pipe: %w", err), Duration: time.Since(start)}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &ExecutionResult{Error: fmt.Errorf("stderr pipe: %w", err), Duration: time.Since(start)}
	}

	if err := cmd.Start(); err != nil {
		return &ExecutionResult{Error: fmt.Errorf("start: %w", err), Duration: time.Since(start)}
	}

	stdoutBytes, _ := io.ReadAll(stdout)
	stderrBytes, _ := io.ReadAll(stderr)

	err = cmd.Wait()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	s.cleanup(containerName)

	return &ExecutionResult{
		Stdout:   string(stdoutBytes),
		Stderr:   string(stderrBytes),
		ExitCode: exitCode,
		Duration: duration,
		Error:    err,
	}
}

func (s *Sandbox) buildDockerArgs(containerName, command string, args ...string) []string {
	dockerArgs := []string{
		"run",
		"--rm",
		"--name", containerName,
		"--network", s.config.NetworkMode,
		"--memory", s.config.MemoryLimit,
		"--cpus", fmt.Sprintf("%.2f", float64(s.config.CPUQuota)/100000.0),
		"--workdir", s.config.WorkDir,
	}

	if s.config.ReadOnlyRootFS {
		dockerArgs = append(dockerArgs, "--read-only")
		dockerArgs = append(dockerArgs, "--tmpfs", "/tmp:rw,noexec,nosuid,size=100m")
		dockerArgs = append(dockerArgs, "--tmpfs", "/workspace:rw,noexec,nosuid,size=500m")
	}

	for _, cap := range s.config.Capabilities {
		dockerArgs = append(dockerArgs, "--cap-add", cap)
	}

	for _, opt := range s.config.SecurityOpts {
		dockerArgs = append(dockerArgs, "--security-opt", opt)
	}

	if s.config.HostWorkDir != "" {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:%s", s.config.HostWorkDir, s.config.WorkDir))
	}

	for hostPath, containerPath := range s.config.Volumes {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	for k, v := range s.config.EnvVars {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	dockerArgs = append(dockerArgs, s.config.Image)

	if command != "" {
		dockerArgs = append(dockerArgs, command)
		dockerArgs = append(dockerArgs, args...)
	}

	return dockerArgs
}

func (s *Sandbox) cleanup(containerName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerName)
	cmd.Run()
}

func (s *Sandbox) ExecuteScript(ctx context.Context, script string, lang string) *ExecutionResult {
	var command string
	var args []string
	var ext string

	switch strings.ToLower(lang) {
	case "bash", "sh":
		command = "sh"
		args = []string{"-c", script}
		ext = ".sh"
	case "python", "python3":
		command = "python3"
		args = []string{"-c", script}
		ext = ".py"
	case "javascript", "js", "node":
		command = "node"
		args = []string{"-e", script}
		ext = ".js"
	default:
		command = "sh"
		args = []string{"-c", script}
		ext = ".sh"
	}

	if s.config.HostWorkDir != "" {
		scriptPath := filepath.Join(s.config.HostWorkDir, fmt.Sprintf("script_%d%s", time.Now().UnixNano(), ext))
		if err := os.WriteFile(scriptPath, []byte(script), 0755); err == nil {
			defer os.Remove(scriptPath)
			containerScriptPath := filepath.Join(s.config.WorkDir, filepath.Base(scriptPath))
			command = "sh"
			args = []string{containerScriptPath}
		}
	}

	return s.Execute(ctx, command, args...)
}

func (s *Sandbox) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "version")
	return cmd.Run() == nil
}

func DefaultSandbox() *Sandbox {
	return NewSandbox(SandboxConfig{
		Image:       "free-agent-runner:latest",
		WorkDir:     "/workspace",
		NetworkMode: "none",
		MemoryLimit: "1g",
		CPUQuota:    100000,
		Timeout:     10 * time.Minute,
		Capabilities: []string{
			"CAP_NET_RAW",
			"CAP_NET_ADMIN",
		},
		SecurityOpts: []string{
			"no-new-privileges:true",
		},
	})
}

func BuildRunnerImage() error {
	dockerfile := `
FROM alpine:latest
RUN apk add --no-cache \
    bash \
    python3 \
    python3-dev \
    py3-pip \
    nodejs \
    npm \
    git \
    curl \
    wget \
    nmap \
    netcat-openbsd \
    bind-tools \
    iputils \
    ca-certificates \
    && pip3 install --no-cache-dir requests beautifulsoup4 lxml \
    && npm install -g npm@latest
WORKDIR /workspace
`
	tmpDir, err := os.MkdirTemp("", "free-agent-runner-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", "free-agent-runner:latest", tmpDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logger.ProcessInfo("Building sandbox runner image...")
	return cmd.Run()
}
