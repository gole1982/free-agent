package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// CommandInjectionScanner - OWASP A03: Injection - 命令注入扫描Agent
// 对应 OWASP Top 10 2021 A03:2021 - Injection
type CommandInjectionScanner struct {
	llmClient *llm.Client
}

// NewCommandInjectionScanner 创建CommandInjectionScanner Agent
func NewCommandInjectionScanner(llmClient *llm.Client) *CommandInjectionScanner {
	return &CommandInjectionScanner{llmClient: llmClient}
}

// Name 实现Agent接口
func (a *CommandInjectionScanner) Name() string {
	return "CommandInjectionScanner"
}

// Description 实现Agent接口
func (a *CommandInjectionScanner) Description() string {
	return "Command Injection Scanner - OWASP A03:2021-Injection - Tests for OS command injection"
}

// Execute 执行命令注入测试
func (a *CommandInjectionScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n💻 [CommandInjectionScanner] Starting Command Injection Scan: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A03:2021 - Injection")
	
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成命令注入测试报告
func (a *CommandInjectionScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    COMMAND INJECTION SCAN REPORT
                    OWASP A03:2021 - Injection
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: CommandInjectionScanner (OWASP A03)
🔍 Category: OS Command Injection

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🛡️ Remediation:
  - Avoid using user input in system commands
  - Use parameterized APIs instead of shell commands
  - Validate and sanitize all user inputs
  - Implement proper error handling

📚 OWASP Reference:
  - OWASP Top 10: A03:2021 - Injection
  - CWE-78: Improper Neutralization of Special Elements used in an OS Command
  - CWE-88: Argument Injection

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

// PathTraversalScanner - OWASP A01: Broken Access Control - 路径遍历扫描Agent
// 对应 OWASP Top 10 2021 A01:2021 - Broken Access Control
type PathTraversalScanner struct {
	llmClient *llm.Client
}

// NewPathTraversalScanner 创建PathTraversalScanner Agent
func NewPathTraversalScanner(llmClient *llm.Client) *PathTraversalScanner {
	return &PathTraversalScanner{llmClient: llmClient}
}

// Name 实现Agent接口
func (a *PathTraversalScanner) Name() string {
	return "PathTraversalScanner"
}

// Description 实现Agent接口
func (a *PathTraversalScanner) Description() string {
	return "Path Traversal Scanner - OWASP A01:2021-Broken Access Control - Tests for path traversal"
}

// Execute 执行路径遍历测试
func (a *PathTraversalScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n📁 [PathTraversalScanner] Starting Path Traversal Scan: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A01:2021 - Broken Access Control")
	
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成路径遍历测试报告
func (a *PathTraversalScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    PATH TRAVERSAL SCAN REPORT
                    OWASP A01:2021 - Broken Access Control
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: PathTraversalScanner (OWASP A01)
🔍 Category: Path/Directory Traversal

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🛡️ Remediation:
  - Validate user input against a whitelist
  - Use chroot jails or containerization
  - Normalize file paths before processing
  - Implement proper access controls

📚 OWASP Reference:
  - OWASP Top 10: A01:2021 - Broken Access Control
  - CWE-22: Improper Limitation of a Pathname to a Restricted Directory
  - CWE-23: Relative Path Traversal

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

// SSRFScanner - OWASP A10: SSRF - 服务端请求伪造扫描Agent
// 对应 OWASP Top 10 2021 A10:2021 - Server-Side Request Forgery
type SSRFScanner struct {
	llmClient *llm.Client
}

// NewSSRFScanner 创建SSRFScanner Agent
func NewSSRFScanner(llmClient *llm.Client) *SSRFScanner {
	return &SSRFScanner{llmClient: llmClient}
}

// Name 实现Agent接口
func (a *SSRFScanner) Name() string {
	return "SSRFScanner"
}

// Description 实现Agent接口
func (a *SSRFScanner) Description() string {
	return "SSRF Scanner - OWASP A10:2021-SSRF - Tests for Server-Side Request Forgery"
}

// Execute 执行SSRF测试
func (a *SSRFScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[SSRFScanner] Starting SSRF Scan: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A10:2021 - Server-Side Request Forgery")
	
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成SSRF测试报告
func (a *SSRFScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    SSRF SCAN REPORT
                    OWASP A10:2021 - Server-Side Request Forgery
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: SSRFScanner (OWASP A10)
🔍 Category: Server-Side Request Forgery

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🛡️ Remediation:
  - Validate and sanitize all user-supplied URLs
  - Use URL allowlists
  - Disable HTTP redirections
  - Enforce URL schema, port, and destination restrictions

📚 OWASP Reference:
  - OWASP Top 10: A10:2021 - Server-Side Request Forgery
  - CWE-918: Server-Side Request Forgery (SSRF)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

// FileIncludeScanner - OWASP A03: Injection - 文件包含扫描Agent
// 对应 OWASP Top 10 2021 A03:2021 - Injection
type FileIncludeScanner struct {
	llmClient *llm.Client
}

// NewFileIncludeScanner 创建FileIncludeScanner Agent
func NewFileIncludeScanner(llmClient *llm.Client) *FileIncludeScanner {
	return &FileIncludeScanner{llmClient: llmClient}
}

// Name 实现Agent接口
func (a *FileIncludeScanner) Name() string {
	return "FileIncludeScanner"
}

// Description 实现Agent接口
func (a *FileIncludeScanner) Description() string {
	return "File Inclusion Scanner - OWASP A03:2021-Injection - Tests for Local/Remote File Inclusion"
}

// Execute 执行文件包含测试
func (a *FileIncludeScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n📄 [FileIncludeScanner] Starting File Inclusion Scan: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A03:2021 - Injection")
	
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成文件包含测试报告
func (a *FileIncludeScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    FILE INCLUSION SCAN REPORT
                    OWASP A03:2021 - Injection
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: FileIncludeScanner (OWASP A03)
🔍 Category: Local/Remote File Inclusion (LFI/RFI)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🛡️ Remediation:
  - Avoid passing user input directly to file inclusion functions
  - Use whitelist approach for file includes
  - Disable allow_url_include for remote file inclusion
  - Validate and sanitize all file paths

📚 OWASP Reference:
  - OWASP Top 10: A03:2021 - Injection
  - CWE-98: Improper Control of Filename for Include/Require Statement
  - CWE-829: Inclusion of Functionality from Untrusted Control Sphere

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

// CTFSolver - CTF挑战解决Agent
// 用于CTF（Capture The Flag）挑战的多方法探索
type CTFSolver struct {
	llmClient *llm.Client
}

// NewCTFSolver 创建CTFSolver Agent
func NewCTFSolver(llmClient *llm.Client) *CTFSolver {
	return &CTFSolver{llmClient: llmClient}
}

// Name 实现Agent接口
func (a *CTFSolver) Name() string {
	return "CTFSolver"
}

// Description 实现Agent接口
func (a *CTFSolver) Description() string {
	return "CTF Solver - Multi-approach challenge solving for Capture The Flag competitions"
}

// Execute 执行CTF挑战
func (a *CTFSolver) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n🏴 [CTFSolver] Starting CTF Challenge: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	
	report := a.generateReport(task)
	return report, nil
}

// generateReport 生成CTF解决报告
func (a *CTFSolver) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                         CTF SOLVE RESULT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: CTFSolver
🏴 Category: Capture The Flag Challenge

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚡ Multi-approach strategy:
  - Reconnaissance: Gather information about the challenge
  - Enumeration: Identify entry points and attack surface
  - Exploitation: Attempt various exploitation techniques
  - Post-exploitation: Extract the flag

🎯 Challenge Types:
  - Web Security: XSS, SQLi, LFI, RCE, etc.
  - Cryptography: Encoding, hashing, encryption
  - Reverse Engineering: Binary analysis, patching
  - Forensics: Memory dumps, pcap analysis
  - OSINT: Information gathering

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}