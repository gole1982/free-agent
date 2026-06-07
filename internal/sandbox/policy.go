package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// PolicyEngine 安全策略引擎 — 管理命令过滤、网络策略、文件系统策略
type PolicyEngine struct {
	mu               sync.RWMutex
	commandPolicy    CommandPolicy
	networkPolicy    NetworkPolicy
	filesystemPolicy FilesystemPolicy
	alertConfig      AlertConfig
}

// CommandPolicy 命令过滤策略
type CommandPolicy struct {
	AllowedCommands []string `json:"allowed_commands"` // 允许的命令白名单
	BlockedPatterns []string `json:"blocked_patterns"` // 禁止的命令模式
	Mode            string   `json:"mode"`             // "whitelist" | "blacklist" | "disabled"
}

// NetworkPolicy 网络策略
type NetworkPolicy struct {
	IsolationMode  string   `json:"isolation_mode"`  // "vpn" | "vlan" | "none"
	AllowInternet  bool     `json:"allow_internet"`  // 是否允许访问互联网
	BlockedPorts   []int    `json:"blocked_ports"`   // 禁止的端口
	AllowedDomains []string `json:"allowed_domains"` // 允许的域名（白名单）
	RateLimitMbps  int      `json:"rate_limit_mbps"` // 速率限制
}

// FilesystemPolicy 文件系统策略
type FilesystemPolicy struct {
	ReadOnly      bool     `json:"readonly"`       // 是否只读
	AllowedPaths  []string `json:"allowed_paths"`  // 允许访问的路径
	BlockedPaths  []string `json:"blocked_paths"`  // 禁止访问的路径
	MaxDiskUsage  string   `json:"max_disk_usage"` // 最大磁盘使用
}

// AlertConfig 告警配置
type AlertConfig struct {
	Enabled           bool     `json:"enabled"`
	Channels          []string `json:"channels"`            // "email" | "webhook" | "log"
	SeverityThreshold string   `json:"severity_threshold"` // "low" | "medium" | "high" | "critical"
}

// PolicyCheckResult 策略检查结果
type PolicyCheckResult struct {
	Allowed bool     `json:"allowed"`
	Reason  string   `json:"reason,omitempty"`
	Rules   []string `json:"rules,omitempty"` // 命中的规则
}

// DefaultPolicyEngine 创建默认策略引擎
func DefaultPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		commandPolicy: CommandPolicy{
			Mode: "blacklist",
			AllowedCommands: []string{
				"nmap", "sqlmap", "nikto", "hydra", "curl", "wget",
				"python3", "python", "node", "bash", "sh", "cat",
				"grep", "awk", "sed", "find", "ls", "pwd", "whoami",
				"id", "uname", "ping", "traceroute", "dig", "nslookup",
				"netcat", "nc", "ssh", "scp", "git",
			},
			BlockedPatterns: []string{
				"rm -rf /",
				"dd if=/dev/zero",
				":(){ :|:& };:", // Fork 炸弹
				"mkfs",
				"fdisk",
				"parted",
				"chmod 777 /",
				"> /dev/sda",
				"wget | sh",
				"curl | sh",
				"eval(",
				"exec(",
			},
		},
		networkPolicy: NetworkPolicy{
			IsolationMode:  "none",
			AllowInternet:  false,
			BlockedPorts:   []int{22, 3389, 445, 135, 139},
			AllowedDomains: []string{"*.exploit-db.com", "*.securityfocus.com", "*.cve.org"},
			RateLimitMbps:  100,
		},
		filesystemPolicy: FilesystemPolicy{
			ReadOnly:     true,
			AllowedPaths: []string{"/tools", "/reports", "/workspace", "/tmp"},
			BlockedPaths: []string{"/etc/shadow", "/etc/passwd", "/root/.ssh", "/boot", "/proc", "/sys"},
			MaxDiskUsage: "10g",
		},
		alertConfig: AlertConfig{
			Enabled:           true,
			Channels:          []string{"log"},
			SeverityThreshold: "medium",
		},
	}
}

// LoadPolicyFromFile 从 JSON 文件加载策略配置
func LoadPolicyFromFile(path string) (*PolicyEngine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file: %w", err)
	}

	var config struct {
		Command    CommandPolicy    `json:"command"`
		Network    NetworkPolicy    `json:"network"`
		Filesystem FilesystemPolicy `json:"filesystem"`
		Alert      AlertConfig      `json:"alert"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse policy file: %w", err)
	}

	return &PolicyEngine{
		commandPolicy:    config.Command,
		networkPolicy:    config.Network,
		filesystemPolicy: config.Filesystem,
		alertConfig:      config.Alert,
	}, nil
}

// CheckCommand 检查命令是否被策略允许
func (pe *PolicyEngine) CheckCommand(command string, args []string) PolicyCheckResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	fullCmd := command
	if len(args) > 0 {
		fullCmd = command + " " + strings.Join(args, " ")
	}

	// 检查黑名单模式
	if pe.commandPolicy.Mode == "blacklist" || pe.commandPolicy.Mode == "whitelist" {
		// 先检查阻止模式
		for _, pattern := range pe.commandPolicy.BlockedPatterns {
			if strings.Contains(fullCmd, pattern) {
				return PolicyCheckResult{
					Allowed: false,
					Reason:  fmt.Sprintf("command matches blocked pattern: %s", pattern),
					Rules:   []string{"blocked_pattern"},
				}
			}
		}
	}

	// 白名单模式：检查命令是否在允许列表中
	if pe.commandPolicy.Mode == "whitelist" {
		baseCmd := command
		if idx := strings.LastIndex(command, "/"); idx >= 0 {
			baseCmd = command[idx+1:]
		}

		allowed := false
		for _, ac := range pe.commandPolicy.AllowedCommands {
			if baseCmd == ac {
				allowed = true
				break
			}
		}
		if !allowed {
			return PolicyCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("command not in whitelist: %s", baseCmd),
				Rules:   []string{"whitelist"},
			}
		}
	}

	return PolicyCheckResult{Allowed: true}
}

// CheckNetworkAccess 检查网络访问是否被策略允许
func (pe *PolicyEngine) CheckNetworkAccess(port int, domain string) PolicyCheckResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var rules []string

	// 检查端口
	for _, bp := range pe.networkPolicy.BlockedPorts {
		if port == bp {
			return PolicyCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("port %d is blocked", port),
				Rules:   []string{"blocked_port"},
			}
		}
	}

	// 检查域名白名单（如果不允许互联网且域名不在白名单中）
	if !pe.networkPolicy.AllowInternet && domain != "" {
		allowed := false
		for _, ad := range pe.networkPolicy.AllowedDomains {
			if matchDomain(domain, ad) {
				allowed = true
				break
			}
		}
		if !allowed {
			rules = append(rules, "domain_not_whitelisted")
			return PolicyCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("domain not in whitelist: %s", domain),
				Rules:   rules,
			}
		}
	}

	return PolicyCheckResult{Allowed: true}
}

// CheckFilesystemAccess 检查文件系统访问是否被策略允许
func (pe *PolicyEngine) CheckFilesystemAccess(path string, write bool) PolicyCheckResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// 只读模式下禁止写入
	if pe.filesystemPolicy.ReadOnly && write {
		// 允许写入特定路径
		for _, ap := range pe.filesystemPolicy.AllowedPaths {
			if strings.HasPrefix(path, ap) {
				return PolicyCheckResult{Allowed: true}
			}
		}
		return PolicyCheckResult{
			Allowed: false,
			Reason:  fmt.Sprintf("filesystem is read-only, write to %s denied", path),
			Rules:   []string{"readonly"},
		}
	}

	// 检查禁止路径
	for _, bp := range pe.filesystemPolicy.BlockedPaths {
		if strings.HasPrefix(path, bp) {
			return PolicyCheckResult{
				Allowed: false,
				Reason:  fmt.Sprintf("path is blocked: %s", bp),
				Rules:   []string{"blocked_path"},
			}
		}
	}

	return PolicyCheckResult{Allowed: true}
}

// CommandPolicy 获取命令策略
func (pe *PolicyEngine) CommandPolicy() CommandPolicy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.commandPolicy
}

// NetworkPolicy 获取网络策略
func (pe *PolicyEngine) NetworkPolicy() NetworkPolicy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.networkPolicy
}

// FilesystemPolicy 获取文件系统策略
func (pe *PolicyEngine) FilesystemPolicy() FilesystemPolicy {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.filesystemPolicy
}

// AlertConfig 获取告警配置
func (pe *PolicyEngine) AlertConfig() AlertConfig {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.alertConfig
}

// matchDomain 域名通配符匹配（支持 *.example.com）
func matchDomain(domain, pattern string) bool {
	if domain == pattern {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		if strings.HasSuffix(domain, suffix) || domain == pattern[2:] {
			return true
		}
	}
	return false
}
