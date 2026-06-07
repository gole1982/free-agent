package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/vibe-coding/free-agent/pkg/config"
)

// SecurityAssessor 安全评估Agent
// 负责协调安全测试，执行OWASP Top 10漏洞扫描
type SecurityAssessor struct {
	securityEnabled bool
	findings        []SecurityFinding
}

// SecurityFinding 安全发现结构
type SecurityFinding struct {
	ID          string
	Type        string // OWASP分类：A01-A10
	Severity    string // CRITICAL/HIGH/MEDIUM/LOW
	Description string
	Payload     string
	OWASPID     string
	Timestamp   string
}

// NewSecurityAssessor 创建SecurityAssessor Agent
func NewSecurityAssessor(progCfg *config.ProgramConfig) *SecurityAssessor {
	return &SecurityAssessor{
		securityEnabled: progCfg.Pentesting.Enabled,
		findings:        []SecurityFinding{},
	}
}

// Name 实现Agent接口
func (a *SecurityAssessor) Name() string {
	return "SecurityAssessor"
}

// Description 实现Agent接口
func (a *SecurityAssessor) Description() string {
	return "Security Assessor - OWASP Top 10 vulnerability scanning and security assessment"
}

// Execute SecurityAssessor的主执行逻辑（Agent接口）
func (a *SecurityAssessor) Execute(ctx context.Context, input string) (string, error) {
	if !a.securityEnabled {
		return "Security assessment mode is not enabled. Set PENTESTING_ENABLED=true in .env to enable.", nil
	}

	// 检查输入是否包含URL（用于实际扫描）
	if strings.Contains(input, "http://") || strings.Contains(input, "https://") || strings.Contains(input, "localhost") {
		return a.executeSecurityScan(ctx, input)
	}

	// 否则使用简单响应
	return "SecurityAssessor ready! Provide a target URL to scan.", nil
}

// executeSecurityScan 执行安全扫描
func (a *SecurityAssessor) executeSecurityScan(ctx context.Context, input string) (string, error) {
	// 从输入中提取URL
	targetURL := extractSecurityTargetURL(input)
	if targetURL == "" {
		targetURL = "http://localhost"
	}

	fmt.Printf("[SecurityAssessor] Starting security scan on: %s\n", targetURL)
	
	// 使用简单扫描器
	report, err := SimpleDVWAScan(targetURL)
	return report, err
}

// extractSecurityTargetURL 从输入中提取目标URL
func extractSecurityTargetURL(input string) string {
	words := strings.Fields(input)
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			return word
		}
		if strings.HasPrefix(word, "localhost") {
			return "http://" + word
		}
	}
	return ""
}

// analyzeInput 分析输入中的安全漏洞
func (a *SecurityAssessor) analyzeInput(input string) []SecurityFinding {
	var findings []SecurityFinding

	// OWASP Top 10 (2021) 漏洞模式
	payloadPatterns := []struct {
		name     string
		pattern  *regexp.Regexp
		severity string
		owaspID  string
	}{
		// A01: Broken Access Control
		{"Path Traversal", regexp.MustCompile(`\.\./`), "HIGH", "A01"},
		
		// A03: Injection
		{"SQL Injection", regexp.MustCompile(`('|")?OR\s+\d+=\d+`), "CRITICAL", "A03"},
		{"SQL Injection", regexp.MustCompile(`UNION\s+SELECT`), "CRITICAL", "A03"},
		{"XSS", regexp.MustCompile(`<script[^>]*>.*?</script>`), "HIGH", "A03"},
		{"XSS", regexp.MustCompile(`javascript:`), "HIGH", "A03"},
		{"Command Injection", regexp.MustCompile(`(;|\|\||&&)\s*(rm|ls|dir|cat)`), "CRITICAL", "A03"},
		{"File Include", regexp.MustCompile(`etc/passwd|windows/system32`), "CRITICAL", "A03"},
		
		// A10: Server-Side Request Forgery
		{"SSRF", regexp.MustCompile(`http://127\.0\.0\.1|http://localhost`), "HIGH", "A10"},
		
		// Other
		{"XXE", regexp.MustCompile(`<!DOCTYPE|<!ENTITY`), "CRITICAL", "A03"},
		{"CSRF", regexp.MustCompile(`csrf|token`), "MEDIUM", "A01"},
		{"RCE", regexp.MustCompile(`system\(|exec\(|shell_exec`), "CRITICAL", "A03"},
		{"Deserialization", regexp.MustCompile(`pickle|serialize|unserialize`), "CRITICAL", "A03"},
	}

	for _, pp := range payloadPatterns {
		if pp.pattern.MatchString(strings.ToLower(input)) {
			matches := pp.pattern.FindString(input)
			findings = append(findings, SecurityFinding{
				ID:          fmt.Sprintf("SA%03d", len(findings)+1),
				Type:        pp.name,
				Severity:    pp.severity,
				Description: fmt.Sprintf("Detected potential %s vulnerability (OWASP %s)", pp.name, pp.owaspID),
				Payload:     matches,
				OWASPID:     pp.owaspID,
				Timestamp:   "Now",
			})
		}
	}

	return findings
}

// GetFindings 获取安全发现
func (a *SecurityAssessor) GetFindings() []SecurityFinding {
	return a.findings
}

// ClearFindings 清除安全发现
func (a *SecurityAssessor) ClearFindings() {
	a.findings = []SecurityFinding{}
}

// IsEnabled 检查是否启用
func (a *SecurityAssessor) IsEnabled() bool {
	return a.securityEnabled
}

// SetEnabled 设置启用状态
func (a *SecurityAssessor) SetEnabled(enabled bool) {
	a.securityEnabled = enabled
}