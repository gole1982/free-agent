package vds

import (
	"context"
	"fmt"
	"time"
)

// =============================================
// Phase 1: 侦察与信息收集
// ATT&CK: T1595, T1591, T1592
// =============================================

func (o *VDSOrchestrator) runReconnaissance(ctx context.Context, target string) (*TargetProfile, error) {
	fmt.Printf("[VDS:Recon] Starting reconnaissance for: %s\n", target)
	start := time.Now()

	profile := &TargetProfile{URL: target}

	for _, plugin := range o.registry.Recons() {
		select {
		case <-ctx.Done():
			o.recordResult("reconnaissance", PhaseFailed, time.Since(start), ctx.Err(), profile)
			return profile, ctx.Err()
		default:
		}
		fmt.Printf("[VDS:Recon] Running plugin: %s\n", plugin.Name())
		result, err := plugin.Recon(ctx, target)
		if err != nil {
			fmt.Printf("[VDS:Recon] Plugin %s failed: %v\n", plugin.Name(), err)
			continue
		}
		profile = mergeProfile(profile, result)
	}

	fmt.Printf("[VDS:Recon] Reconnaissance completed in %v\n", time.Since(start))
	o.recordResult("reconnaissance", PhaseCompleted, time.Since(start), nil, profile)
	return profile, nil
}

func mergeProfile(base, addition *TargetProfile) *TargetProfile {
	if addition == nil {
		return base
	}
	base.Domains = appendUniqueStr(base.Domains, addition.Domains)
	base.Ports = appendUniqueInt(base.Ports, addition.Ports)
	base.Technologies = appendUniqueStr(base.Technologies, addition.Technologies)
	if addition.ServerInfo.Server != "" {
		base.ServerInfo = addition.ServerInfo
	}
	base.EntryPoints = append(base.EntryPoints, addition.EntryPoints...)
	return base
}

// =============================================
// Phase 2: 应用映射
// ATT&CK: T1595.002, T1087, T1069
// =============================================

func (o *VDSOrchestrator) runMapping(ctx context.Context, profile *TargetProfile) (*AttackSurface, error) {
	fmt.Printf("[VDS:Mapping] Starting application mapping for: %s\n", profile.URL)
	start := time.Now()

	surface := &AttackSurface{}

	for _, plugin := range o.registry.Mappers() {
		select {
		case <-ctx.Done():
			o.recordResult("mapping", PhaseFailed, time.Since(start), ctx.Err(), surface)
			return surface, ctx.Err()
		default:
		}
		fmt.Printf("[VDS:Mapping] Running plugin: %s\n", plugin.Name())
		result, err := plugin.Map(ctx, profile)
		if err != nil {
			fmt.Printf("[VDS:Mapping] Plugin %s failed: %v\n", plugin.Name(), err)
			continue
		}
		if result != nil {
			surface.Endpoints = append(surface.Endpoints, result.Endpoints...)
			surface.Parameters = append(surface.Parameters, result.Parameters...)
			surface.Cookies = append(surface.Cookies, result.Cookies...)
			surface.APIs = append(surface.APIs, result.APIs...)
		}
	}

	// 从 TargetProfile 的 EntryPoints 自动提取
	for _, ep := range profile.EntryPoints {
		surface.Endpoints = append(surface.Endpoints, Endpoint{
			URL: ep.URL, Method: ep.Method, Parameters: ep.Parameters,
		})
		surface.Parameters = append(surface.Parameters, ep.Parameters...)
	}

	fmt.Printf("[VDS:Mapping] Mapping completed in %v: %d endpoints\n",
		time.Since(start), len(surface.Endpoints))
	o.recordResult("mapping", PhaseCompleted, time.Since(start), nil, surface)
	return surface, nil
}

// =============================================
// Phase 3: 漏洞发现
// ATT&CK: T1190, T1059, T1110, T1078
// 桥接现有 SecurityAssessor + 各 Scanner
// =============================================

func (o *VDSOrchestrator) runDiscovery(ctx context.Context, surface *AttackSurface) ([]VulnerabilityFinding, error) {
	fmt.Printf("[VDS:Discovery] Starting vulnerability discovery: %d endpoints\n", len(surface.Endpoints))
	start := time.Now()

	var allFindings []VulnerabilityFinding

	for _, plugin := range o.registry.Scanners() {
		select {
		case <-ctx.Done():
			o.recordResult("discovery", PhaseFailed, time.Since(start), ctx.Err(), allFindings)
			return allFindings, ctx.Err()
		default:
		}

		fmt.Printf("[VDS:Discovery] Running scanner: %s (OWASP: %s)\n", plugin.Name(), plugin.OWASPCategory())
		findings, err := plugin.Scan(ctx, surface)
		if err != nil {
			fmt.Printf("[VDS:Discovery] Scanner %s failed: %v\n", plugin.Name(), err)
			continue
		}

		// 确保 OWASP/ATT&CK 映射字段已填充
		for i := range findings {
			if findings[i].OWASPCategory == "" {
				findings[i].OWASPCategory = plugin.OWASPCategory()
			}
			if findings[i].OWASPID == "" {
				findings[i].OWASPID = plugin.OWASPID()
			}
			if findings[i].ATTCKTechnique == "" && len(plugin.ATTCKTechniques()) > 0 {
				findings[i].ATTCKTechnique = plugin.ATTCKTechniques()[0]
			}
			if findings[i].ATTCKTactic == "" && len(plugin.ATTCKTactics()) > 0 {
				findings[i].ATTCKTactic = plugin.ATTCKTactics()[0]
			}
		}
		allFindings = append(allFindings, findings...)
	}

	fmt.Printf("[VDS:Discovery] Discovery completed in %v: %d findings\n",
		time.Since(start), len(allFindings))
	o.recordResult("discovery", PhaseCompleted, time.Since(start), nil, allFindings)
	return allFindings, nil
}

// =============================================
// Phase 4: 漏洞利用（默认禁用）
// ATT&CK: T1190, T1078, T1003, T1567
// =============================================

func (o *VDSOrchestrator) runExploitation(ctx context.Context, findings []VulnerabilityFinding) ([]ExploitResult, error) {
	fmt.Printf("[VDS:Exploitation] Starting exploitation: %d findings\n", len(findings))
	start := time.Now()

	var results []ExploitResult

	// 筛选可利用的漏洞
	var exploitable []VulnerabilityFinding
	for _, f := range findings {
		if f.Exploitable || f.Confidence >= 80 {
			exploitable = append(exploitable, f)
		}
	}

	for _, plugin := range o.registry.Exploits() {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
		for i := range exploitable {
			fmt.Printf("[VDS:Exploitation] Attempting: %s on %s\n", plugin.Name(), exploitable[i].ID)
			result, err := plugin.Exploit(ctx, &exploitable[i])
			if err != nil {
				continue
			}
			results = append(results, *result)
		}
	}

	o.recordResult("exploitation", PhaseCompleted, time.Since(start), nil, results)
	return results, nil
}

// =============================================
// Phase 6: 修复验证（默认禁用）
// =============================================

func (o *VDSOrchestrator) runRemediation(ctx context.Context, findings []VulnerabilityFinding) {
	fmt.Printf("[VDS:Remediation] Starting remediation verification: %d findings\n", len(findings))
	start := time.Now()

	var results []RemediationResult
	for _, plugin := range o.registry.Scanners() {
		matched := filterFindings(findings, plugin.OWASPCategory())
		if len(matched) == 0 {
			continue
		}
		surface := buildSurface(findings)
		retestFindings, err := plugin.Scan(ctx, surface)
		if err != nil {
			continue
		}
		retestIDs := make(map[string]bool)
		for _, rf := range retestFindings {
			retestIDs[rf.ID] = true
		}
		for _, f := range matched {
			results = append(results, RemediationResult{
				FindingID: f.ID,
				Verified:  !retestIDs[f.ID],
				Details:   fmt.Sprintf("Retested by %s", plugin.Name()),
				RetestAt:  time.Now(),
			})
		}
	}

	status := PhaseCompleted
	o.recordResult("remediation", status, time.Since(start), nil, results)
	if len(results) > 0 {
		verified := 0
		for _, r := range results {
			if r.Verified {
				verified++
			}
		}
		fmt.Printf("[VDS:Remediation] %d/%d verified\n", verified, len(results))
	}
}

// =============================================
// 辅助函数
// =============================================

func appendUniqueStr(base, items []string) []string {
	seen := make(map[string]bool, len(base))
	for _, s := range base {
		seen[s] = true
	}
	for _, s := range items {
		if !seen[s] {
			base = append(base, s)
			seen[s] = true
		}
	}
	return base
}

func appendUniqueInt(base, items []int) []int {
	seen := make(map[int]bool, len(base))
	for _, v := range base {
		seen[v] = true
	}
	for _, v := range items {
		if !seen[v] {
			base = append(base, v)
			seen[v] = true
		}
	}
	return base
}

func filterFindings(findings []VulnerabilityFinding, category string) []VulnerabilityFinding {
	var matched []VulnerabilityFinding
	for _, f := range findings {
		if f.OWASPCategory == category {
			matched = append(matched, f)
		}
	}
	return matched
}

func buildSurface(findings []VulnerabilityFinding) *AttackSurface {
	surface := &AttackSurface{}
	for _, f := range findings {
		if f.Location != "" {
			surface.Endpoints = append(surface.Endpoints, Endpoint{URL: f.Location, Method: "GET"})
		}
	}
	return surface
}
