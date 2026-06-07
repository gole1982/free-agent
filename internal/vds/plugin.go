package vds

import "context"

// Plugin 插件基础接口
type Plugin interface {
	Name() string
	Version() string
	Initialize(config map[string]interface{}) error
}

// ScannerPlugin 扫描器插件接口
type ScannerPlugin interface {
	Plugin
	// OWASP 映射
	OWASPCategory() string // OWASP 分类，如 "A03:2021-Injection"
	OWASPID() string       // OWASP ID，如 "WSTG-INPV-01"
	// ATT&CK 映射
	ATTCKTechniques() []string // ATT&CK 技术 ID 列表，如 ["T1190", "T1059"]
	ATTCKTactics() []string    // ATT&CK 战术列表，如 ["Initial Access", "Execution"]
	// 执行
	Scan(ctx context.Context, surface *AttackSurface) ([]VulnerabilityFinding, error)
}

// ExploitPlugin 利用插件接口
type ExploitPlugin interface {
	Plugin
	ATTCKTechnique() string // 对应的 ATT&CK 技术
	Exploit(ctx context.Context, finding *VulnerabilityFinding) (*ExploitResult, error)
}

// ReconPlugin 侦察插件接口
type ReconPlugin interface {
	Plugin
	Recon(ctx context.Context, target string) (*TargetProfile, error)
}

// MappingPlugin 映射插件接口
type MappingPlugin interface {
	Plugin
	Map(ctx context.Context, profile *TargetProfile) (*AttackSurface, error)
}

// ReportPlugin 报告插件接口
type ReportPlugin interface {
	Plugin
	Generate(ctx context.Context, report *ScanReport) ([]byte, error)
	Format() string // "html", "json", "markdown"
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	scanners  []ScannerPlugin
	exploits  []ExploitPlugin
	recons    []ReconPlugin
	mappers   []MappingPlugin
	reporters []ReportPlugin
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{}
}

// RegisterScanner 注册扫描器插件
func (r *PluginRegistry) RegisterScanner(p ScannerPlugin) {
	r.scanners = append(r.scanners, p)
}

// RegisterExploit 注册利用插件
func (r *PluginRegistry) RegisterExploit(p ExploitPlugin) {
	r.exploits = append(r.exploits, p)
}

// RegisterRecon 注册侦察插件
func (r *PluginRegistry) RegisterRecon(p ReconPlugin) {
	r.recons = append(r.recons, p)
}

// RegisterMapper 注册映射插件
func (r *PluginRegistry) RegisterMapper(p MappingPlugin) {
	r.mappers = append(r.mappers, p)
}

// RegisterReporter 注册报告插件
func (r *PluginRegistry) RegisterReporter(p ReportPlugin) {
	r.reporters = append(r.reporters, p)
}

// Scanners 获取所有扫描器
func (r *PluginRegistry) Scanners() []ScannerPlugin { return r.scanners }

// Exploits 获取所有利用插件
func (r *PluginRegistry) Exploits() []ExploitPlugin { return r.exploits }

// Recons 获取所有侦察插件
func (r *PluginRegistry) Recons() []ReconPlugin { return r.recons }

// Mappers 获取所有映射插件
func (r *PluginRegistry) Mappers() []MappingPlugin { return r.mappers }

// Reporters 获取所有报告插件
func (r *PluginRegistry) Reporters() []ReportPlugin { return r.reporters }
