package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
)

type SecurityAssessorAgent struct {
	agentMgr  *AgentManager
	llmClient *llm.Client
}

type scannerEntry struct {
	Name     string
	OWASPCat string
}

type scannerResult struct {
	Entry    scannerEntry
	Output   string
	Err      error
	Skipped  bool
	SkipNote string
}

var defaultScannerRoster = []scannerEntry{
	{"SQLInjectionScanner", "A03"},
	{"XSSScanner", "A03"},
	{"CommandInjectionScanner", "A03"},
	{"FileIncludeScanner", "A03"},
	{"PathTraversalScanner", "A01"},
	{"SSRFScanner", "A10"},
	{"CryptographicFailuresScanner", "A02"},
	{"InsecureDesignScanner", "A04"},
	{"SecurityMisconfigurationScanner", "A05"},
	{"VulnerableComponentsScanner", "A06"},
	{"AuthenticationFailuresScanner", "A07"},
	{"SoftwareIntegrityScanner", "A08"},
	{"LoggingFailuresScanner", "A09"},
}

func NewSecurityAssessorAgent(am *AgentManager, llmClient *llm.Client) *SecurityAssessorAgent {
	return &SecurityAssessorAgent{agentMgr: am, llmClient: llmClient}
}

func (a *SecurityAssessorAgent) Name() string {
	return "SecurityAssessor"
}

func (a *SecurityAssessorAgent) Description() string {
	return "OWASP Top 10 security testing coordinator - delegates to specialized scanners and aggregates findings"
}

func (a *SecurityAssessorAgent) Execute(ctx context.Context, task string) (string, error) {
	target := extractSecurityTarget(task)
	if target == "" {
		return "", fmt.Errorf("unable to extract target URL from task: %s", task)
	}

	fmt.Printf("[SecurityAssessor] Starting security assessment for: %s\n", target)
	results := a.runScanners(ctx, target, defaultScannerRoster, nil)
	return formatAssessmentReport(target, defaultScannerRoster, results), nil
}

// RunWithObserver executes security assessment with per-scanner Observer monitoring.
// The Observer receives ExecutorExecutionInfo for each scanner and can stop execution early.
func (a *SecurityAssessorAgent) RunWithObserver(ctx context.Context, task string, observer *ObserverAgent) (string, error) {
	target := extractSecurityTarget(task)
	if target == "" {
		return "", fmt.Errorf("unable to extract target URL from task: %s", task)
	}

	fmt.Printf("[SecurityAssessor] Starting observed security assessment for: %s\n", target)
	results := a.runScanners(ctx, target, defaultScannerRoster, observer)
	return formatAssessmentReport(target, defaultScannerRoster, results), nil
}

func (a *SecurityAssessorAgent) runScanners(ctx context.Context, target string, roster []scannerEntry, observer *ObserverAgent) []scannerResult {
	results := make([]scannerResult, 0, len(roster))
	for _, entry := range roster {
		// Check if Observer wants to stop
		if observer != nil {
			select {
			case <-ctx.Done():
				fmt.Println("[SecurityAssessor] Context cancelled, stopping scanners")
				return results
			default:
			}
		}

		sc, err := a.agentMgr.GetAgent(entry.Name)
		if err != nil {
			results = append(results, scannerResult{Entry: entry, Skipped: true, SkipNote: "not registered"})
			fmt.Printf("  [skip] %s not registered\n", entry.Name)
			continue
		}
		out, runErr := sc.Execute(ctx, fmt.Sprintf("Assess %s for %s", entry.Name, target))
		results = append(results, scannerResult{Entry: entry, Output: out, Err: runErr})

		// Report per-scanner info to Observer
		if observer != nil {
			flag := "normal"
			if runErr != nil {
				flag = "error"
			}
			info := ExecutorExecutionInfo{
				AgentName:     entry.Name,
				Task:          fmt.Sprintf("Assess %s for %s", entry.Name, target),
				Output:        truncate(out, 300),
				ExecutionFlag: flag,
				Timestamp:     time.Now(),
			}
			observer.ReceiveExecutorInfo(info)

			// Brief pause to allow Observer to process
			time.Sleep(50 * time.Millisecond)
			decision := observer.GetDecision()
			if decision.ShouldStop {
				fmt.Printf("[SecurityAssessor] Observer stopped assessment: %s\n", decision.Reason)
				return results
			}
		}
	}
	return results
}

func formatAssessmentReport(target string, roster []scannerEntry, results []scannerResult) string {
	cats := summarizeCategories(roster)
	var b strings.Builder
	fmt.Fprintf(&b, "===== Security Assessment Report =====\n\n")
	fmt.Fprintf(&b, "Target: %s\n", target)
	fmt.Fprintf(&b, "OWASP Coverage: %s\n", cats)
	fmt.Fprintf(&b, "Scanners Run: %d\n\n", len(results))

	for _, r := range results {
		fmt.Fprintf(&b, "\n--- %s [%s] ---\n", r.Entry.Name, r.Entry.OWASPCat)
		if r.Skipped {
			fmt.Fprintf(&b, "[skipped] %s\n", r.SkipNote)
			continue
		}
		if r.Err != nil {
			fmt.Fprintf(&b, "Error: %v\n", r.Err)
			continue
		}
		b.WriteString(r.Output)
		if !strings.HasSuffix(r.Output, "\n") {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n===== End of Report =====\n")
	return b.String()
}

func summarizeCategories(roster []scannerEntry) string {
	seen := make(map[string]bool, 8)
	parts := make([]string, 0, 8)
	for _, r := range roster {
		if seen[r.OWASPCat] {
			continue
		}
		seen[r.OWASPCat] = true
		parts = append(parts, r.OWASPCat)
	}
	return strings.Join(parts, ", ")
}

func extractSecurityTarget(task string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if idx := strings.Index(task, prefix); idx >= 0 {
			rest := task[idx+len(prefix):]
			if end := strings.IndexAny(rest, " \t\n"); end >= 0 {
				return task[idx : idx+len(prefix)+end]
			}
			return task[idx:]
		}
	}
	return ""
}
