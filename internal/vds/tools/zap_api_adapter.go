package tools

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vibe-coding/free-agent/internal/vds"
)

// ZapAdapter OWASP ZAP REST API 适配器
// 通过 zap.sh -daemon -port 8080 启动的 API 进行 Web 安全扫描
type ZapAdapter struct {
	*APIAdapter
}

// NewZapAdapter 创建 ZAP 适配器（默认 http://127.0.0.1:8080）
func NewZapAdapter(baseURL, apiKey string) *ZapAdapter {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	return &ZapAdapter{
		APIAdapter: NewAPIAdapter(baseURL, apiKey),
	}
}

func (z *ZapAdapter) ToolName() string    { return "owasp-zap" }
func (z *ZapAdapter) AdapterType() string  { return "api" }
func (z *ZapAdapter) IsAvailable(ctx context.Context) bool {
	return z.CheckAvailability(ctx, "/JSON/core/view/version/")
}

// --- ZAP API 数据模型 ---

type zapScanResponse struct {
	Scan string `json:"scan"` // scan ID
}

type zapStatusResponse struct {
	Status string `json:"status"` // "0"-"100" (进度百分比)
}

type zapAlert struct {
	Name        string `json:"name"`
	Risk        string `json:"risk"`     // "High" | "Medium" | "Low" | "Informational"
	Confidence  string `json:"confidence"` // "High" | "Medium" | "Low"
	Description string `json:"description"`
	Solution    string `json:"solution"`
	URL         string `json:"url"`
	Param       string `json:"param"`
	Evidence    string `json:"evidence"`
	CWEID       string `json:"cweid"`
	WASCID      string `json:"wascid"`
	Reference   string `json:"reference"`
}

type zapAlertsResponse struct {
	Alerts []zapAlert `json:"alerts"`
}

// Spider 启动蜘蛛爬取
func (z *ZapAdapter) Spider(ctx context.Context, targetURL string) (string, error) {
	var resp struct {
		Spider string `json:"spider"`
	}
	path := fmt.Sprintf("/JSON/spider/action/scan/?url=%s", url.QueryEscape(targetURL))
	if err := z.Get(ctx, path, &resp); err != nil {
		return "", fmt.Errorf("zap spider: %w", err)
	}
	return resp.Spider, nil
}

// ActiveScan 启动主动扫描
func (z *ZapAdapter) ActiveScan(ctx context.Context, targetURL string) (string, error) {
	var resp zapScanResponse
	path := fmt.Sprintf("/JSON/ascan/action/scan/?url=%s", url.QueryEscape(targetURL))
	if err := z.Get(ctx, path, &resp); err != nil {
		return "", fmt.Errorf("zap active scan: %w", err)
	}
	return resp.Scan, nil
}

// GetScanStatus 获取扫描进度（0-100）
func (z *ZapAdapter) GetScanStatus(ctx context.Context, scanID string) (int, error) {
	var resp zapStatusResponse
	path := fmt.Sprintf("/JSON/ascan/view/status/?scanId=%s", scanID)
	if err := z.Get(ctx, path, &resp); err != nil {
		return 0, fmt.Errorf("zap scan status: %w", err)
	}
	status, _ := strconv.Atoi(resp.Status)
	return status, nil
}

// GetAlerts 获取扫描告警
func (z *ZapAdapter) GetAlerts(ctx context.Context, targetURL string) ([]zapAlert, error) {
	var resp zapAlertsResponse
	path := fmt.Sprintf("/JSON/alert/view/alerts/?baseurl=%s", url.QueryEscape(targetURL))
	if err := z.Get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("zap alerts: %w", err)
	}
	return resp.Alerts, nil
}

// ScanTarget 完整扫描流程：蜘蛛 → 主动扫描 → 等待 → 获取告警
func (z *ZapAdapter) ScanTarget(ctx context.Context, targetURL string) ([]vds.VulnerabilityFinding, error) {
	// 1. 蜘蛛爬取
	spiderID, err := z.Spider(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	// 等待蜘蛛完成
	spiderCtx, spiderCancel := context.WithTimeout(ctx, 10*time.Minute)
	defer spiderCancel()
	for {
		select {
		case <-spiderCtx.Done():
			return nil, fmt.Errorf("zap spider timeout")
		default:
			time.Sleep(2 * time.Second)
			var status struct {
				Status string `json:"status"`
			}
			z.Get(ctx, fmt.Sprintf("/JSON/spider/view/status/?scanId=%s", spiderID), &status)
			if status.Status == "100" {
				goto spiderDone
			}
		}
	}
spiderDone:

	// 2. 主动扫描
	scanID, err := z.ActiveScan(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	// 3. 等待扫描完成
	scanCtx, scanCancel := context.WithTimeout(ctx, 30*time.Minute)
	defer scanCancel()
	for {
		select {
		case <-scanCtx.Done():
			return nil, fmt.Errorf("zap scan timeout")
		default:
			time.Sleep(3 * time.Second)
			progress, err := z.GetScanStatus(ctx, scanID)
			if err != nil {
				return nil, err
			}
			if progress >= 100 {
				goto scanDone
			}
		}
	}
scanDone:

	// 4. 获取告警
	alerts, err := z.GetAlerts(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	// 5. 转换为 VulnerabilityFinding
	return z.toFindings(alerts), nil
}

// toFindings 将 ZAP 告警转换为 VDS VulnerabilityFinding
func (z *ZapAdapter) toFindings(alerts []zapAlert) []vds.VulnerabilityFinding {
	findings := make([]vds.VulnerabilityFinding, 0, len(alerts))

	for i, alert := range alerts {
		finding := vds.VulnerabilityFinding{
			ID:          fmt.Sprintf("zap-%d", i+1),
			Type:        alert.Name,
			Severity:    mapZapRisk(alert.Risk),
			Description: alert.Description,
			Evidence:    []byte(alert.Evidence),
			Payload:     alert.Param,
			Location:    alert.URL,
			Confidence:  mapZapConfidence(alert.Confidence),

			// OWASP/ATT&CK 映射
			OWASPCategory:  mapZapToOWASP(alert.CWEID),
			OWASPID:        fmt.Sprintf("WSTG-%s", alert.WASCID),
			ATTCKTechnique: mapZapToATTCK(alert.CWEID),
			ATTCKTactic:    "Initial Access",
			Exploitable:    alert.Risk == "High" || alert.Risk == "Medium",
		}
		findings = append(findings, finding)
	}
	return findings
}

// --- ZAP 映射辅助函数 ---

func mapZapRisk(risk string) string {
	switch strings.ToLower(risk) {
	case "high":
		return "HIGH"
	case "medium":
		return "MEDIUM"
	case "low":
		return "LOW"
	default:
		return "LOW"
	}
}

func mapZapConfidence(conf string) int {
	switch strings.ToLower(conf) {
	case "high":
		return 90
	case "medium":
		return 70
	case "low":
		return 40
	default:
		return 50
	}
}

// mapZapToOWASP 根据 CWE ID 映射 OWASP 分类
func mapZapToOWASP(cweID string) string {
	owaspMap := map[string]string{
		"89":  "A03:2021-Injection",          // SQL Injection
		"79":  "A03:2021-Injection",          // XSS
		"78":  "A03:2021-Injection",          // OS Command Injection
		"22":  "A01:2021-Broken Access Control", // Path Traversal
		"918": "A10:2021-Server-Side Request Forgery",
		"287": "A07:2021-Identification and Authentication Failures",
		"352": "A01:2021-Broken Access Control", // CSRF
		"611": "A05:2021-Security Misconfiguration", // XXE
		"200": "A05:2021-Security Misconfiguration", // Info Disclosure
		"319": "A02:2021-Cryptographic Failures",
	}
	if cat, ok := owaspMap[cweID]; ok {
		return cat
	}
	return "A05:2021-Security Misconfiguration"
}

// mapZapToATTCK 根据 CWE ID 映射 ATT&CK 技术
func mapZapToATTCK(cweID string) string {
	attckMap := map[string]string{
		"89":  "T1190", // SQL Injection
		"79":  "T1190", // XSS
		"78":  "T1059", // Command Injection
		"22":  "T1190", // Path Traversal
		"918": "T1190", // SSRF
		"287": "T1110", // Auth Failures → Brute Force
		"352": "T1190", // CSRF
		"611": "T1190", // XXE
	}
	if tech, ok := attckMap[cweID]; ok {
		return tech
	}
	return "T1190"
}
