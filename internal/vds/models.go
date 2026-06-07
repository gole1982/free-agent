package vds

import "time"

// Target 扫描目标
type Target struct {
	ID             string
	URL            string
	Name           string
	Scope          Scope
	Status         string
	CreatedAt      time.Time
	ATTCKTactics   []string // 适用的 ATT&CK 战术
}

// Scope 扫描范围
type Scope struct {
	IncludeDomains []string
	ExcludePaths   []string
	MaxDepth       int
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Server     string
	Technology string
	OS         string
	Headers    map[string]string
}

// EntryPoint 入口点
type EntryPoint struct {
	URL        string
	Method     string
	Parameters []Parameter
}

// TargetProfile 目标画像（Phase 1 输出）
type TargetProfile struct {
	URL          string
	Domains      []string
	Ports        []int
	Technologies []string
	ServerInfo   ServerInfo
	EntryPoints  []EntryPoint
}

// Endpoint 端点
type Endpoint struct {
	ID                string
	URL               string
	Method            string
	Parameters        []Parameter
	ContentType       string
	AuthRequired      bool
	ATTCKTechniques   []string // 可能涉及的 ATT&CK 技术
}

// Parameter 参数
type Parameter struct {
	Name         string
	Type         string // query/form/json/cookie
	Location     string
	Example      string
	AttackVector string // 可能的攻击向量
}

// Cookie Cookie
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Secure   bool
	HttpOnly bool
}

// API API 端点
type API struct {
	URL        string
	Method     string
	AuthType   string
	Parameters []Parameter
}

// AttackSurface 攻击面（Phase 2 输出）
type AttackSurface struct {
	Endpoints  []Endpoint
	Parameters []Parameter
	Cookies    []Cookie
	APIs       []API
}

// VulnerabilityFinding 漏洞发现（Phase 3 输出）
type VulnerabilityFinding struct {
	ID          string
	Type        string // 漏洞类型
	Severity    string // CRITICAL/HIGH/MEDIUM/LOW
	Description string
	Evidence    []byte
	Payload     string
	Location    string
	Confidence  int // 0-100

	// 双框架映射
	OWASPCategory  string // OWASP 分类，如 "A03:2021-Injection"
	OWASPID        string // OWASP ID，如 "WSTG-INPV-01"
	ATTCKTechnique string // ATT&CK 技术 ID，如 "T1190"
	ATTCKTactic    string // ATT&CK 战术，如 "Initial Access"

	// 利用信息
	Exploitable    bool
	ExploitProof   []byte
	BusinessImpact string
}

// ExploitResult 漏洞利用结果（Phase 4 输出）
type ExploitResult struct {
	Finding    *VulnerabilityFinding
	Successful bool
	Impact     string
	Proof      []byte
	Chain      []string // 漏洞链
}

// ATTCKCoverage ATT&CK 覆盖条目
type ATTCKCoverage struct {
	TechniqueID   string `json:"technique_id"`
	TechniqueName string `json:"technique_name"`
	Tactic        string `json:"tactic"`
	Attempted     bool   `json:"attempted"`
	Detected      bool   `json:"detected"`
	Blocked       bool   `json:"blocked"`
	Evidence      string `json:"evidence"`
}

// ScanReport 扫描报告（Phase 5 输出）
type ScanReport struct {
	ScanID              string
	Target              string
	GeneratedAt         time.Time
	Findings            []VulnerabilityFinding
	ExploitResults      []ExploitResult
	ATTCKCoverageMatrix []ATTCKCoverage
	Summary             ReportSummary
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalFindings  int
	Critical       int
	High           int
	Medium         int
	Low            int
	OWASPCoverage  []string
	ATTCKCoverage  []string
	RiskScore      float64
}

// RemediationResult 修复验证结果（Phase 6 输出）
type RemediationResult struct {
	FindingID  string
	Verified   bool
	Details    string
	RetestAt   time.Time
}
