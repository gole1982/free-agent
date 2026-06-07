package tools

import (
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/vibe-coding/free-agent/internal/vds"
)

// NmapAdapter Nmap XML 输出适配器
// 通过 nmap -sV -sC -oX 命令获取结构化网络扫描结果
type NmapAdapter struct {
	BinaryPath string // nmap 可执行文件路径
}

// NewNmapAdapter 创建 Nmap 适配器
func NewNmapAdapter(binaryPath string) *NmapAdapter {
	if binaryPath == "" {
		binaryPath = "nmap"
	}
	return &NmapAdapter{BinaryPath: binaryPath}
}

func (n *NmapAdapter) ToolName() string    { return "nmap" }
func (n *NmapAdapter) AdapterType() string  { return "cli" }
func (n *NmapAdapter) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, n.BinaryPath, "--version")
	return cmd.Run() == nil
}

// --- Nmap XML 数据模型 ---

// NmapRun Nmap XML 根节点
type NmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Args    string     `xml:"args,attr"`
	Start   string     `xml:"start,attr"`
	Hosts   []NmapHost `xml:"host"`
}

// NmapHost 扫描到的主机
type NmapHost struct {
	Addresses []NmapAddress `xml:"address"`
	Hostnames []NmapHostFn  `xml:"hostnames>hostname"`
	Ports     NmapPorts     `xml:"ports"`
	OS        NmapOS        `xml:"os"`
	Status    NmapStatus    `xml:"status"`
}

type NmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type NmapHostFn struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type NmapPorts struct {
	Ports []NmapPort `xml:"port"`
}

type NmapPort struct {
	Protocol string     `xml:"protocol,attr"`
	PortID   string     `xml:"portid,attr"`
	State    NmapState  `xml:"state"`
	Service  NmapService `xml:"service"`
	Scripts  []NmapScript `xml:"script"`
}

type NmapState struct {
	State  string `xml:"state,attr"`
	Reason string `xml:"reason,attr"`
}

type NmapService struct {
	Name      string `xml:"name,attr"`
	Product   string `xml:"product,attr"`
	Version   string `xml:"version,attr"`
	ExtraInfo string `xml:"extrainfo,attr"`
	CPE       string `xml:"cpe,attr"`
}

type NmapScript struct {
	ID     string `xml:"id,attr"`
	Output string `xml:"output,attr"`
}

type NmapOS struct {
	OSMatches []NmapOSMatch `xml:"osmatch"`
}

type NmapOSMatch struct {
	Name     string `xml:"name,attr"`
	Accuracy string `xml:"accuracy,attr"`
}

type NmapStatus struct {
	State string `xml:"state,attr"`
}

// RunScan 执行 nmap 扫描并返回 XML 结果
func (n *NmapAdapter) RunScan(ctx context.Context, target string, extraArgs ...string) ([]byte, error) {
	args := []string{
		"-sV",      // 服务版本检测
		"-sC",      // 默认脚本扫描
		"-oX", "-", // XML 输出到 stdout
	}
	args = append(args, extraArgs...)
	args = append(args, target)

	cmd := exec.CommandContext(ctx, n.BinaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nmap scan: %w", err)
	}
	return output, nil
}

// ParseXML 解析 Nmap XML 输出
func (n *NmapAdapter) ParseXML(data []byte) (*NmapRun, error) {
	var result NmapRun
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse nmap xml: %w", err)
	}
	return &result, nil
}

// ScanAndParse 执行扫描并解析结果
func (n *NmapAdapter) ScanAndParse(ctx context.Context, target string, extraArgs ...string) (*NmapRun, error) {
	data, err := n.RunScan(ctx, target, extraArgs...)
	if err != nil {
		return nil, err
	}
	return n.ParseXML(data)
}

// ToTargetProfile 将 Nmap 结果转换为 VDS TargetProfile（Phase 1 输出）
func (n *NmapAdapter) ToTargetProfile(result *NmapRun, targetURL string) *vds.TargetProfile {
	profile := &vds.TargetProfile{
		URL:          targetURL,
		Ports:        make([]int, 0),
		Technologies: make([]string, 0),
		EntryPoints:  make([]vds.EntryPoint, 0),
	}

	for _, host := range result.Hosts {
		if host.Status.State != "up" {
			continue
		}

		// 收集域名
		for _, hn := range host.Hostnames {
			profile.Domains = append(profile.Domains, hn.Name)
		}

		// 收集开放端口
		for _, port := range host.Ports.Ports {
			if port.State.State != "open" {
				continue
			}
			portNum, _ := strconv.Atoi(port.PortID)
			profile.Ports = append(profile.Ports, portNum)

			// 收集技术栈信息
			if port.Service.Product != "" {
				tech := port.Service.Product
				if port.Service.Version != "" {
					tech += " " + port.Service.Version
				}
				profile.Technologies = append(profile.Technologies, tech)
			}

			// 创建入口点
			serviceName := port.Service.Name
			if isWebService(serviceName, portNum) {
				ep := vds.EntryPoint{
					URL:    fmt.Sprintf("http://%s:%d", targetURL, portNum),
					Method: "GET",
				}
				profile.EntryPoints = append(profile.EntryPoints, ep)
			}
		}

		// OS 信息
		if len(host.OS.OSMatches) > 0 {
			profile.ServerInfo.OS = host.OS.OSMatches[0].Name
		}
	}

	return profile
}

// ToFindings 将 Nmap 脚本扫描结果转换为 VulnerabilityFinding
func (n *NmapAdapter) ToFindings(result *NmapRun) []vds.VulnerabilityFinding {
	var findings []vds.VulnerabilityFinding
	idx := 0

	for _, host := range result.Hosts {
		for _, port := range host.Ports.Ports {
			for _, script := range port.Scripts {
				// 只转换漏洞相关的脚本结果
				if isVulnScript(script.ID) {
					idx++
					finding := vds.VulnerabilityFinding{
						ID:             fmt.Sprintf("nmap-%d", idx),
						Type:           fmt.Sprintf("Nmap Script: %s", script.ID),
						Severity:       "MEDIUM",
						Description:    script.Output,
						Location:       fmt.Sprintf("Port %s/%s", port.PortID, port.Protocol),
						Confidence:     70,
						OWASPCategory:  "A05:2021-Security Misconfiguration",
						OWASPID:        "WSTG-CONF-01",
						ATTCKTechnique: "T1595",
						ATTCKTactic:    "Reconnaissance",
					}

					// 如果是 CVE 脚本，提升严重程度
					if strings.Contains(script.ID, "cve") || strings.Contains(strings.ToLower(script.Output), "vulnerable") {
						finding.Severity = "HIGH"
						finding.Exploitable = true
						finding.ATTCKTechnique = "T1190"
						finding.ATTCKTactic = "Initial Access"
					}

					findings = append(findings, finding)
				}
			}
		}
	}
	return findings
}

// --- 辅助函数 ---

func isWebService(serviceName string, port int) bool {
	webServices := map[string]bool{"http": true, "https": true, "http-proxy": true, "http-alt": true}
	if webServices[serviceName] {
		return true
	}
	webPorts := map[int]bool{80: true, 443: true, 8080: true, 8443: true, 8000: true, 3000: true}
	return webPorts[port]
}

func isVulnScript(scriptID string) bool {
	vulnPrefixes := []string{
		"http-vuln", "ssl-", "smb-vuln", "vuln",
		"http-cve", "exploit", "http-shellshock",
	}
	lower := strings.ToLower(scriptID)
	for _, prefix := range vulnPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
