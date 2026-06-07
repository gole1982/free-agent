package vds

import (
	"encoding/json"
	"fmt"
	"os"
)

// VDSConfig VDS 系统配置
type VDSConfig struct {
	Target     TargetConfig     `json:"target"`
	Frameworks FrameworksConfig `json:"frameworks"`
	Phases     PhasesConfig     `json:"phases"`
}

// TargetConfig 目标配置
type TargetConfig struct {
	URL   string      `json:"url"`
	Scope ScopeConfig `json:"scope"`
}

// ScopeConfig 范围配置
type ScopeConfig struct {
	IncludeDomains []string `json:"include_domains"`
	ExcludePaths   []string `json:"exclude_paths"`
	MaxDepth       int      `json:"max_depth"`
}

// FrameworksConfig 双框架配置
type FrameworksConfig struct {
	OWASP OWASPConfig  `json:"owasp"`
	ATTCK ATTCKConfig  `json:"attck"`
}

// OWASPConfig OWASP 配置
type OWASPConfig struct {
	Enabled    bool     `json:"enabled"`
	Categories []string `json:"categories"`
}

// ATTCKConfig ATT&CK 配置
type ATTCKConfig struct {
	Enabled    bool     `json:"enabled"`
	Matrix     string   `json:"matrix"`
	Tactics    []string `json:"tactics"`
	Techniques []string `json:"techniques"`
}

// PhasesConfig 各阶段配置
type PhasesConfig struct {
	Reconnaissance PhaseConfig `json:"reconnaissance"`
	Mapping        PhaseConfig `json:"mapping"`
	Discovery      PhaseConfig `json:"discovery"`
	Exploitation   PhaseConfig `json:"exploitation"`
	Reporting      PhaseConfig `json:"reporting"`
	Remediation    PhaseConfig `json:"remediation"`
}

// PhaseConfig 单个阶段配置
type PhaseConfig struct {
	Enabled         bool     `json:"enabled"`
	ATTCKTechniques []string `json:"attck_techniques"`
	Scanners        []string `json:"scanners,omitempty"`
}

// LoadConfig 从 YAML 文件加载 VDS 配置
func LoadConfig(path string) (*VDSConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read vds config: %w", err)
	}
	var cfg VDSConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse vds config: %w", err)
	}
	return &cfg, nil
}

// DefaultConfig 返回默认 VDS 配置
func DefaultConfig() *VDSConfig {
	return &VDSConfig{
		Frameworks: FrameworksConfig{
			OWASP: OWASPConfig{
				Enabled:    true,
				Categories: []string{"INFO", "CONF", "AUTHN", "AUTHZ", "INPV", "BUSL"},
			},
			ATTCK: ATTCKConfig{
				Enabled: true,
				Matrix:  "enterprise",
				Tactics: []string{
					"Reconnaissance", "Initial Access", "Execution",
					"Credential Access", "Discovery", "Exfiltration",
				},
				Techniques: []string{"T1595", "T1190", "T1059", "T1110", "T1078"},
			},
		},
		Phases: PhasesConfig{
			Reconnaissance: PhaseConfig{
				Enabled:         true,
				ATTCKTechniques: []string{"T1595", "T1591", "T1592"},
			},
			Mapping: PhaseConfig{
				Enabled:         true,
				ATTCKTechniques: []string{"T1595.002", "T1087", "T1069"},
			},
			Discovery: PhaseConfig{
				Enabled:         true,
				ATTCKTechniques: []string{"T1190", "T1059", "T1110", "T1078"},
				Scanners:        []string{"sqli", "xss", "command_injection", "path_traversal"},
			},
			Exploitation: PhaseConfig{
				Enabled:         false,
				ATTCKTechniques: []string{"T1190", "T1078", "T1003", "T1567"},
			},
			Reporting: PhaseConfig{
				Enabled: true,
			},
			Remediation: PhaseConfig{
				Enabled: false,
			},
		},
	}
}
