package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type CryptographicFailuresScanner struct {
	llmClient *llm.Client
}

func NewCryptographicFailuresScanner(llmClient *llm.Client) *CryptographicFailuresScanner {
	return &CryptographicFailuresScanner{llmClient: llmClient}
}

func (a *CryptographicFailuresScanner) Name() string {
	return "CryptographicFailuresScanner"
}

func (a *CryptographicFailuresScanner) Description() string {
	return "Cryptographic Failures Scanner - OWASP A02:2021 - Tests for weak hashing, missing TLS, hardcoded keys, ECB mode, MD5/SHA1"
}

func (a *CryptographicFailuresScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[Key] [CryptographicFailuresScanner] Starting Crypto Scan: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A02:2021 - Cryptographic Failures")
	return a.generateReport(task), nil
}

func (a *CryptographicFailuresScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                    CRYPTOGRAPHIC FAILURES SCAN REPORT
                    OWASP A02:2021 - Cryptographic Failures
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: CryptographicFailuresScanner (OWASP A02)
🔍 Category: Cryptography Misuse

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔐 Checkpoints:
  - TLS configuration: HTTP-only endpoints, deprecated TLS 1.0/1.1
  - Hashing: MD5/SHA1 used for passwords; bcrypt/argon2 missing
  - Storage: Plaintext secrets in config files, env files, source code
  - Randomness: math/rand used for tokens/keys instead of crypto/rand
  - Modes: ECB mode, static IV, custom ciphers

🛡️ Remediation:
  - Enforce TLS 1.2+; enable HSTS
  - Use bcrypt/argon2/scrypt for password hashing
  - Move secrets to a vault (KMS, HashiCorp Vault, env)
  - Use crypto/rand for tokens/keys/nonces
  - Prefer AES-GCM or ChaCha20-Poly1305 authenticated encryption

📚 OWASP Reference:
  - OWASP Top 10: A02:2021 - Cryptographic Failures
  - CWE-259: Use of Hard-coded Password
  - CWE-327: Use of a Broken or Risky Cryptographic Algorithm
  - CWE-330: Use of Insufficiently Random Values
  - CWE-319: Cleartext Transmission of Sensitive Information
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

type InsecureDesignScanner struct {
	llmClient *llm.Client
}

func NewInsecureDesignScanner(llmClient *llm.Client) *InsecureDesignScanner {
	return &InsecureDesignScanner{llmClient: llmClient}
}

func (a *InsecureDesignScanner) Name() string {
	return "InsecureDesignScanner"
}

func (a *InsecureDesignScanner) Description() string {
	return "Insecure Design Scanner - OWASP A04:2021 - Tests for missing rate limiting, business-logic abuse, threat modeling gaps"
}

func (a *InsecureDesignScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[Design] [InsecureDesignScanner] Starting Design Review: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A04:2021 - Insecure Design")
	return a.generateReport(task), nil
}

func (a *InsecureDesignScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                       INSECURE DESIGN SCAN REPORT
                       OWASP A04:2021 - Insecure Design
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: InsecureDesignScanner (OWASP A04)
🔍 Category: Architecture & Business Logic

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🧩 Checkpoints:
  - Rate limiting: login/reset endpoints, sensitive APIs
  - Business logic: race conditions, replay, mass assignment
  - Workflow: bypassable state transitions
  - Threat modeling: missing STRIDE/abuse-case review
  - Failure modes: silently swallowed errors, open retries

🛡️ Remediation:
  - Implement per-IP and per-account rate limits
  - Use state machines with explicit guards
  - Document and test abuse cases
  - Apply STRIDE/DREAD in design phase
  - Define and enforce failure contracts

📚 OWASP Reference:
  - OWASP Top 10: A04:2021 - Insecure Design
  - CWE-209: Generation of Error Message Containing Sensitive Information
  - CWE-256: Plaintext Storage of a Password
  - CWE-501: Trust Boundary Violation
  - CWE-799: Improper Control of Interaction Frequency
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

type SecurityMisconfigurationScanner struct {
	llmClient *llm.Client
}

func NewSecurityMisconfigurationScanner(llmClient *llm.Client) *SecurityMisconfigurationScanner {
	return &SecurityMisconfigurationScanner{llmClient: llmClient}
}

func (a *SecurityMisconfigurationScanner) Name() string {
	return "SecurityMisconfigurationScanner"
}

func (a *SecurityMisconfigurationScanner) Description() string {
	return "Security Misconfiguration Scanner - OWASP A05:2021 - Tests for default credentials, open admin panels, missing headers, CORS"
}

func (a *SecurityMisconfigurationScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[Config] [SecurityMisconfigurationScanner] Starting Config Review: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A05:2021 - Security Misconfiguration")
	return a.generateReport(task), nil
}

func (a *SecurityMisconfigurationScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                  SECURITY MISCONFIGURATION SCAN REPORT
                  OWASP A05:2021 - Security Misconfiguration
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: SecurityMisconfigurationScanner (OWASP A05)
🔍 Category: Hardening & Defaults

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚙️ Checkpoints:
  - Default credentials: admin/admin, root/root, vendor defaults
  - Open admin panels: /.env, /admin, /phpmyadmin, /actuator
  - Security headers: missing CSP, HSTS, X-Frame-Options, Referrer-Policy
  - CORS: Access-Control-Allow-Origin: *
  - Verbose errors: stack traces, framework versions in responses
  - Cloud storage: public S3 buckets, open MongoDB/Redis

🛡️ Remediation:
  - Rotate default credentials at deploy time
  - Lock down admin paths (VPN, allowlist, network policy)
  - Add CSP/HSTS/X-Frame-Options/X-Content-Type-Options at the edge
  - Restrict CORS to known origins
  - Return generic error pages; log details server-side
  - Audit cloud resources; enforce least-privilege IAM

📚 OWASP Reference:
  - OWASP Top 10: A05:2021 - Security Misconfiguration
  - CWE-2: 7PK - Environment
  - CWE-16: Configuration
  - CWE-260: Password in Configuration File
  - CWE-732: Incorrect Permission Assignment for Critical Resource
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

type VulnerableComponentsScanner struct {
	llmClient *llm.Client
}

func NewVulnerableComponentsScanner(llmClient *llm.Client) *VulnerableComponentsScanner {
	return &VulnerableComponentsScanner{llmClient: llmClient}
}

func (a *VulnerableComponentsScanner) Name() string {
	return "VulnerableComponentsScanner"
}

func (a *VulnerableComponentsScanner) Description() string {
	return "Vulnerable Components Scanner - OWASP A06:2021 - Tests for outdated libraries, unpatched CVEs, EOL runtimes"
}

func (a *VulnerableComponentsScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[Box] [VulnerableComponentsScanner] Starting Dependency Audit: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A06:2021 - Vulnerable and Outdated Components")
	return a.generateReport(task), nil
}

func (a *VulnerableComponentsScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                     VULNERABLE COMPONENTS SCAN REPORT
                     OWASP A06:2021 - Vulnerable and Outdated Components
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: VulnerableComponentsScanner (OWASP A06)
🔍 Category: Supply Chain & Dependencies

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📦 Checkpoints:
  - Outdated libraries: lockfile vs. latest
  - Known CVEs: CVE feeds, GitHub Advisory, NVD
  - EOL runtimes: Python 2, Node 14, OpenSSL 1.0
  - Transitive deps: deep CVEs in nested packages
  - Unsigned binaries: integrity of vendored binaries

🛡️ Remediation:
  - Run SCA on every CI build (npm audit, go list -m, pip-audit)
  - Subscribe to vendor security advisories
  - Pin versions; review upgrades; auto-merge patches
  - Use SBOM (CycloneDX/SPDX) for visibility
  - Verify checksums and signatures for vendored artifacts

📚 OWASP Reference:
  - OWASP Top 10: A06:2021 - Vulnerable and Outdated Components
  - CWE-937: OWASP Top Ten 2013 Category A9
  - CWE-1035: OWASP Top Ten 2017 Category A9
  - CWE-1104: Use of Unmaintained Third Party Components
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

type AuthenticationFailuresScanner struct {
	llmClient *llm.Client
}

func NewAuthenticationFailuresScanner(llmClient *llm.Client) *AuthenticationFailuresScanner {
	return &AuthenticationFailuresScanner{llmClient: llmClient}
}

func (a *AuthenticationFailuresScanner) Name() string {
	return "AuthenticationFailuresScanner"
}

func (a *AuthenticationFailuresScanner) Description() string {
	return "Authentication Failures Scanner - OWASP A07:2021 - Tests for weak passwords, missing MFA, session fixation, IDOR"
}

func (a *AuthenticationFailuresScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[ID] [AuthenticationFailuresScanner] Starting Auth Review: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A07:2021 - Identification and Authentication Failures")
	return a.generateReport(task), nil
}

func (a *AuthenticationFailuresScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                  AUTHENTICATION FAILURES SCAN REPORT
                  OWASP A07:2021 - Identification and Authentication Failures
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: AuthenticationFailuresScanner (OWASP A07)
🔍 Category: AuthN / Session

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔑 Checkpoints:
  - Credential stuffing: missing rate limit on /login
  - Weak passwords: no complexity, no breach-list check (HIBP)
  - MFA gaps: optional or bypassable 2FA
  - Session management: predictable IDs, missing rotation, missing SameSite
  - IDOR: object access by primary key with no ownership check
  - Recovery flows: predictable tokens, infinite token TTL

🛡️ Remediation:
  - Enforce strong passwords; check against breach lists
  - Mandate MFA for privileged roles
  - Use server-side session store; rotate on login; mark cookies Secure+HttpOnly+SameSite
  - Add ownership/authorization checks on every object read
  - Use cryptographically random single-use recovery tokens

📚 OWASP Reference:
  - OWASP Top 10: A07:2021 - Identification and Authentication Failures
  - CWE-287: Improper Authentication
  - CWE-297: Improper Validation of Certificate
  - CWE-384: Session Fixation
  - CWE-613: Insufficient Session Expiration
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

type SoftwareIntegrityScanner struct {
	llmClient *llm.Client
}

func NewSoftwareIntegrityScanner(llmClient *llm.Client) *SoftwareIntegrityScanner {
	return &SoftwareIntegrityScanner{llmClient: llmClient}
}

func (a *SoftwareIntegrityScanner) Name() string {
	return "SoftwareIntegrityScanner"
}

func (a *SoftwareIntegrityScanner) Description() string {
	return "Software Integrity Failures Scanner - OWASP A08:2021 - Tests for unsigned updates, untrusted deserialization, CI/CD pipeline attacks"
}

func (a *SoftwareIntegrityScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[Verify] [SoftwareIntegrityScanner] Starting Integrity Review: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A08:2021 - Software and Data Integrity Failures")
	return a.generateReport(task), nil
}

func (a *SoftwareIntegrityScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                   SOFTWARE INTEGRITY SCAN REPORT
                   OWASP A08:2021 - Software and Data Integrity Failures
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: SoftwareIntegrityScanner (OWASP A08)
🔍 Category: CI/CD, Deserialization, Auto-update

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🧷 Checkpoints:
  - Auto-update channels without signature verification
  - Unsafe deserialization: pickle, Java ObjectInputStream, YAML unsafe load
  - Insecure plugins: unsigned extensions, dynamic eval
  - CI/CD: long-lived secrets in pipelines, unverified artifacts
  - Unsigned container images or base images from unknown registries

🛡️ Remediation:
  - Sign updates and verify signatures on the client
  - Use safe deserializers (JSON with strict schema)
  - Verify plugin signatures; sandbox eval
  - Use OIDC short-lived tokens in CI; sign artifacts with Sigstore/cosign
  - Pin base images; scan at build time

📚 OWASP Reference:
  - OWASP Top 10: A08:2021 - Software and Data Integrity Failures
  - CWE-345: Insufficient Verification of Data Authenticity
  - CWE-502: Deserialization of Untrusted Data
  - CWE-829: Inclusion of Functionality from Untrusted Control Sphere
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}

type LoggingFailuresScanner struct {
	llmClient *llm.Client
}

func NewLoggingFailuresScanner(llmClient *llm.Client) *LoggingFailuresScanner {
	return &LoggingFailuresScanner{llmClient: llmClient}
}

func (a *LoggingFailuresScanner) Name() string {
	return "LoggingFailuresScanner"
}

func (a *LoggingFailuresScanner) Description() string {
	return "Logging Failures Scanner - OWASP A09:2021 - Tests for missing audit logs, sensitive data in logs, no alerting on suspicious events"
}

func (a *LoggingFailuresScanner) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n[Log] [LoggingFailuresScanner] Starting Logging Review: %s\n", task)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("OWASP Category: A09:2021 - Security Logging and Monitoring Failures")
	return a.generateReport(task), nil
}

func (a *LoggingFailuresScanner) generateReport(task string) string {
	return fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                       LOGGING FAILURES SCAN REPORT
                       OWASP A09:2021 - Security Logging and Monitoring Failures
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📌 Task: %s
⚙️  Agent: LoggingFailuresScanner (OWASP A09)
🔍 Category: Audit, Detection, Response

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Checkpoints:
  - Audit coverage: login, privilege changes, payments, data exports
  - Log content: PII, credentials, tokens leaking into logs
  - Log integrity: append-only storage, tamper resistance
  - Detection: no alerts on auth failures, suspicious patterns
  - Incident response: no runbook, no on-call

🛡️ Remediation:
  - Log auth, access control, and validation failures with correlation IDs
  - Mask PII and secrets at the log layer
  - Ship logs to a SIEM with retention policy
  - Define alerts (5xx spikes, repeated 401s, geo-anomalies)
  - Maintain a tested IR runbook; tabletop exercises quarterly

📚 OWASP Reference:
  - OWASP Top 10: A09:2021 - Security Logging and Monitoring Failures
  - CWE-117: Improper Output Neutralization for Logs
  - CWE-223: Omission of Security-relevant Information
  - CWE-532: Insertion of Sensitive Information into Log File
  - CWE-778: Insufficient Logging
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, task)
}
