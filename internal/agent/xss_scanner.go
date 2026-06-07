package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// XSSScanner - OWASP A03: Injection - XSS扫描Agent
// 对应 OWASP Top 10 2021 A03:2021 - Injection
type XSSScanner struct {
	llmClient *llm.Client
}

// NewXSSScanner 创建XSSScanner Agent
func NewXSSScanner(llmClient *llm.Client) *XSSScanner {
	return &XSSScanner{
		llmClient: llmClient,
	}
}

// Name 实现Agent接口
func (a *XSSScanner) Name() string {
	return "XSSScanner"
}

// Description 实现Agent接口
func (a *XSSScanner) Description() string {
	return "XSS Scanner - OWASP A03:2021-Injection - Tests for Cross-Site Scripting vulnerabilities"
}

// Execute 执行XSS测试
func (a *XSSScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n🎯 [XSSScanner] Starting XSS Scan: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A03:2021 - Injection")

	// OWASP XSS测试流程
	workflow := []struct {
		step string
		desc string
	}{
		{"1. Identify Input Points", "Find all possible user input locations"},
		{"2. Reflected XSS Testing", "Test <script>alert(1)</script>"},
		{"3. Stored XSS Testing", "Test if malicious payload can be saved"},
		{"4. DOM-Based XSS Testing", "Test location.hash, document.write, etc."},
		{"5. WAF Bypass Attempts", "Encoding, obfuscation bypass techniques"},
		{"6. Context Analysis", "Determine proper context for exploitation"},
		{"7. Generate Report", "Document findings"},
	}

	for i, wf := range workflow {
		fmt.Printf("\n📋 Step %d: %s\n", i+1, wf.step)
		fmt.Printf("   Description: %s\n", wf.desc)
	}

	// 模拟执行过程
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成XSS测试报告
func (a *XSSScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                          XSS SCAN REPORT
                    OWASP A03:2021 - Injection
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
📅 Time: Current Time
⚙️  Agent: XSSScanner (OWASP A03)
🔍 Category: Cross-Site Scripting (XSS)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ Test Findings (Example):
  1. Reflected XSS Detected - MEDIUM Risk
  2. Stored XSS Detected - HIGH Risk
  3. Affected Areas: Login page, Comments, Search
  4. Impact: Cookie theft, Session hijacking, Phishing redirect

🛡️ Remediation:
  - Output Encoding (HTML Entity Encoding)
  - Content Security Policy (CSP)
  - HttpOnly and Secure flags for cookies
  - Input validation and filtering
  - Use safe JavaScript APIs

📚 OWASP Reference:
  - OWASP Top 10: A03:2021 - Injection
  - CWE-79: Cross-site Scripting (XSS)
  - CWE-80: Improper Neutralization of Script-Related HTML Tags
  - OWASP XSS Prevention Cheat Sheet

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}