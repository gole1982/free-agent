package control

import (
	"fmt"
	"regexp"
	"strings"
)

type SafetyGuardrail struct {
	ID          string
	Name        string
	Description string
	Pattern     *regexp.Regexp
	Severity    SeverityLevel
	Action      PolicyAction
}

type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "critical"
	SeverityHigh     SeverityLevel = "high"
	SeverityMedium   SeverityLevel = "medium"
	SeverityLow      SeverityLevel = "low"
)

type SafetyResult struct {
	Blocked    bool
	Message    string
	Guardrails []SafetyGuardrail
}

type SafetyGuardrails struct {
	guardrails []SafetyGuardrail
}

func NewSafetyGuardrails() *SafetyGuardrails {
	return &SafetyGuardrails{
		guardrails: []SafetyGuardrail{
			{
				ID:          "SG001",
				Name:        "Malicious Code Detection",
				Description: "Blocks known malicious code patterns",
				Pattern:     regexp.MustCompile(`(rm -rf|del /f|format|curl.*|wget.*|system\(.*\)|exec\(.*\))`),
				Severity:    SeverityCritical,
				Action:      ActionDeny,
			},
			{
				ID:          "SG002",
				Name:        "Sensitive Data Protection",
				Description: "Blocks sensitive information exposure",
				Pattern:     regexp.MustCompile(`(password|secret|api[_-]?key|token|private[_-]?key)`),
				Severity:    SeverityHigh,
				Action:      ActionWarn,
			},
			{
				ID:          "SG003",
				Name:        "Network Access Restriction",
				Description: "Blocks network access attempts",
				Pattern:     regexp.MustCompile(`(http://|https://|ftp://|socket|net\.|url\.open)`),
				Severity:    SeverityMedium,
				Action:      ActionReview,
			},
			{
				ID:          "SG004",
				Name:        "Resource Consumption",
				Description: "Blocks excessive resource usage",
				Pattern:     regexp.MustCompile(`(infinite|while\s*\(|for\s*\(|loop)`),
				Severity:    SeverityMedium,
				Action:      ActionWarn,
			},
		},
	}
}

func (sg *SafetyGuardrails) AddGuardrail(guardrail SafetyGuardrail) {
	sg.guardrails = append(sg.guardrails, guardrail)
}

func (sg *SafetyGuardrails) Inspect(input string) SafetyResult {
	var matched []SafetyGuardrail
	var messages []string
	blocked := false

	for _, guardrail := range sg.guardrails {
		if guardrail.Pattern.MatchString(strings.ToLower(input)) {
			matched = append(matched, guardrail)
			messages = append(messages, fmt.Sprintf("[%s] %s: %s", 
				guardrail.Severity, guardrail.Name, guardrail.Description))

			if guardrail.Action == ActionDeny {
				blocked = true
			}
		}
	}

	return SafetyResult{
		Blocked:    blocked,
		Message:    strings.Join(messages, "\n"),
		Guardrails: matched,
	}
}

func (sg *SafetyGuardrails) ListGuardrails() []SafetyGuardrail {
	return sg.guardrails
}
