package vds

import (
	"context"
	"fmt"
	"time"

	"github.com/vibe-coding/free-agent/internal/logger"
)

// PhaseStatus 阶段状态
type PhaseStatus string

const (
	PhasePending    PhaseStatus = "pending"
	PhaseRunning    PhaseStatus = "running"
	PhaseCompleted  PhaseStatus = "completed"
	PhaseFailed     PhaseStatus = "failed"
	PhaseSkipped    PhaseStatus = "skipped"
)

// PhaseResult 阶段执行结果
type PhaseResult struct {
	Phase    string
	Status   PhaseStatus
	Duration time.Duration
	Error    error
	Data     interface{} // Phase-specific output
}

// VDSOrchestrator VDS 协调器 — 管理 6 个阶段的工作流
type VDSOrchestrator struct {
	config   *VDSConfig
	registry *PluginRegistry
	results  []PhaseResult
}

// NewVDSOrchestrator 创建 VDS 协调器
func NewVDSOrchestrator(config *VDSConfig, registry *PluginRegistry) *VDSOrchestrator {
	if config == nil {
		config = DefaultConfig()
	}
	if registry == nil {
		registry = NewPluginRegistry()
	}
	return &VDSOrchestrator{
		config:   config,
		registry: registry,
	}
}

// Run 执行完整的 VDS 扫描流程
func (o *VDSOrchestrator) Run(ctx context.Context, targetURL string) (*ScanReport, error) {
	fmt.Printf("[VDS] Starting vulnerability discovery for: %s\n", targetURL)
	startTime := time.Now()

	// Phase 1: 侦察与信息收集
	profile, err := o.runReconnaissance(ctx, targetURL)
	if err != nil {
		return nil, fmt.Errorf("reconnaissance phase failed: %w", err)
	}

	// Phase 2: 应用映射
	surface, err := o.runMapping(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("mapping phase failed: %w", err)
	}

	// Phase 3: 漏洞发现
	findings, err := o.runDiscovery(ctx, surface)
	if err != nil {
		return nil, fmt.Errorf("discovery phase failed: %w", err)
	}

	// Phase 4: 漏洞利用（默认禁用）
	var exploitResults []ExploitResult
	if o.config.Phases.Exploitation.Enabled {
		exploitResults, err = o.runExploitation(ctx, findings)
		if err != nil {
			logger.ProcessWarn("[VDS] Exploitation phase failed (non-fatal): %v", err)
		}
	}

	// Phase 5: 报告生成
	report := o.buildReport(targetURL, findings, exploitResults, time.Since(startTime))

	// Phase 6: 修复验证（默认禁用）
	if o.config.Phases.Remediation.Enabled {
		o.runRemediation(ctx, findings)
	}

	fmt.Printf("[VDS] Scan completed in %v, %d findings\n", time.Since(startTime), len(findings))
	return report, nil
}

// Results 获取所有阶段结果
func (o *VDSOrchestrator) Results() []PhaseResult {
	return o.results
}

func (o *VDSOrchestrator) recordResult(phase string, status PhaseStatus, duration time.Duration, err error, data interface{}) {
	o.results = append(o.results, PhaseResult{
		Phase:    phase,
		Status:   status,
		Duration: duration,
		Error:    err,
		Data:     data,
	})
}

func (o *VDSOrchestrator) buildReport(target string, findings []VulnerabilityFinding, exploits []ExploitResult, duration time.Duration) *ScanReport {
	report := &ScanReport{
		ScanID:      fmt.Sprintf("scan-%d", time.Now().Unix()),
		Target:      target,
		GeneratedAt: time.Now(),
		Findings:    findings,
		ExploitResults: exploits,
	}

	// 构建 ATT&CK 覆盖矩阵
	coverageMap := make(map[string]*ATTCKCoverage)
	for _, f := range findings {
		if f.ATTCKTechnique != "" {
			if _, exists := coverageMap[f.ATTCKTechnique]; !exists {
				coverageMap[f.ATTCKTechnique] = &ATTCKCoverage{
					TechniqueID: f.ATTCKTechnique,
					Tactic:      f.ATTCKTactic,
					Attempted:   true,
					Detected:    true,
					Blocked:     false,
					Evidence:    f.Type,
				}
			}
		}
	}
	for _, c := range coverageMap {
		report.ATTCKCoverageMatrix = append(report.ATTCKCoverageMatrix, *c)
	}

	// 构建摘要
	summary := ReportSummary{TotalFindings: len(findings)}
	owaspSet := make(map[string]bool)
	attckSet := make(map[string]bool)
	for _, f := range findings {
		switch f.Severity {
		case "CRITICAL":
			summary.Critical++
		case "HIGH":
			summary.High++
		case "MEDIUM":
			summary.Medium++
		case "LOW":
			summary.Low++
		}
		if f.OWASPCategory != "" {
			owaspSet[f.OWASPCategory] = true
		}
		if f.ATTCKTechnique != "" {
			attckSet[f.ATTCKTechnique] = true
		}
	}
	for k := range owaspSet {
		summary.OWASPCoverage = append(summary.OWASPCoverage, k)
	}
	for k := range attckSet {
		summary.ATTCKCoverage = append(summary.ATTCKCoverage, k)
	}
	// 简单风险评分：CRITICAL=10, HIGH=7, MEDIUM=4, LOW=1
	total := float64(summary.Critical*10 + summary.High*7 + summary.Medium*4 + summary.Low*1)
	if total > 100 {
		total = 100
	}
	summary.RiskScore = total
	report.Summary = summary

	return report
}
