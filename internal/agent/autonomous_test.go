package agent

import (
	"fmt"
	"strings"
	"time"
)

// ===================================================
// Autonomous Testing Environment
// ===================================================

type TestEnvironment struct {
	Target        string
	CurrentState  State
	Agent         *LearningAgent
	Actions       []Action
	EpisodeCount  int
	MaxSteps      int
}

func NewTestEnvironment(target string, agent *LearningAgent) *TestEnvironment {
	return &TestEnvironment{
		Target:       target,
		CurrentState: State{TargetURL: target},
		Agent:        agent,
		Actions:      getDefaultActions(),
		EpisodeCount: 0,
		MaxSteps:     50,
	}
}

func getDefaultActions() []Action {
	return []Action{
		{
			Name:        "test_sqli",
			Description: "Test SQL injection vulnerabilities",
			Parameters:  map[string]interface{}{"payloads": "basic"},
		},
		{
			Name:        "test_xss",
			Description: "Test cross-site scripting vulnerabilities",
			Parameters:  map[string]interface{}{"type": "reflected"},
		},
		{
			Name:        "test_lfi",
			Description: "Test local file inclusion vulnerabilities",
			Parameters:  map[string]interface{}{"depth": 4},
		},
		{
			Name:        "test_command_injection",
			Description: "Test command injection vulnerabilities",
			Parameters:  map[string]interface{}{"mode": "basic"},
		},
		{
			Name:        "test_csrf",
			Description: "Test cross-site request forgery",
			Parameters:  map[string]interface{}{"method": "post"},
		},
	}
}

func (env *TestEnvironment) RunEpisode() (string, error) {
	env.EpisodeCount++
	fmt.Printf("\n" + strings.Repeat("🎬", 20))
	fmt.Printf("\n🎬 STARTING EPISODE %d\n", env.EpisodeCount)
	fmt.Println(strings.Repeat("🎬", 20) + "\n")

	// Reset state
	env.CurrentState = State{
		TargetURL:      env.Target,
		DiscoveredVulns: 0,
		FailedAttempts:   0,
		Context:          make(map[string]string),
	}

	totalReward := 0.0
	var results []string

	for step := 0; step < env.MaxSteps; step++ {
		fmt.Printf("\n--- STEP %d ---\n", step+1)

		// Select action
		action := env.Agent.SelectAction(env.CurrentState, env.Actions)

		// Execute action
		nextState, reward, result := env.ExecuteAction(action)

		// Record experience
		experience := Experience{
			State:      env.CurrentState,
			Action:     action,
			Reward:     reward,
			NextState:  nextState,
			Timestamp:  time.Now().Format(time.RFC3339),
		}

		// Learn from experience
		env.Agent.Learn(experience)

		// Update state
		env.CurrentState = nextState
		totalReward += reward.Value

		results = append(results, fmt.Sprintf("[Step %d] %s → %s (Reward: %.2f)",
			step+1, action.Name, result, reward.Value))

		// Check if done
		if env.CurrentState.DiscoveredVulns >= 5 {
			fmt.Println("\n🎯 SUCCESS: Found 5+ vulnerabilities!")
			break
		}

		if env.CurrentState.FailedAttempts >= 10 {
			fmt.Println("\n⚠️  STOPPED: Too many failed attempts")
			break
		}
	}

	// Update stats
	env.Agent.Stats.TotalEpisodes++
	if env.Agent.Stats.TotalEpisodes > 0 {
		successCount := 0
		for _, exp := range env.Agent.ExperienceLog {
			if exp.NextState.DiscoveredVulns > exp.State.DiscoveredVulns {
				successCount++
			}
		}
		env.Agent.Stats.SuccessRate = float64(successCount) / float64(env.Agent.Stats.TotalEpisodes)
	}

	fmt.Printf("\n📊 Episode %d Complete - Total Reward: %.2f\n", env.EpisodeCount, totalReward)

	// Generate report
	var report strings.Builder
	report.WriteString("\n" + strings.Repeat("=", 80) + "\n")
	report.WriteString(fmt.Sprintf("                   AUTONOMOUS TEST REPORT - EPISODE %d\n", env.EpisodeCount))
	report.WriteString(strings.Repeat("=", 80) + "\n")
	report.WriteString(fmt.Sprintf("Target:      %s\n", env.Target))
	report.WriteString(fmt.Sprintf("Timestamp:   %s\n", time.Now().Format(time.RFC1123)))
	report.WriteString(fmt.Sprintf("Vulns found: %d\n", env.CurrentState.DiscoveredVulns))
	report.WriteString(fmt.Sprintf("Total Reward: %.2f\n", totalReward))
	report.WriteString("\nStep-by-step:\n")
	for _, r := range results {
		report.WriteString(fmt.Sprintf("  %s\n", r))
	}
	report.WriteString("\n" + strings.Repeat("=", 80) + "\n")

	return report.String(), nil
}

func (env *TestEnvironment) ExecuteAction(action Action) (State, Reward, string) {
	fmt.Printf("▶️  Executing action: %s\n", action.Name)

	// Simulate or execute real test
	var result string
	var vulnFound bool

	switch action.Name {
	case "test_sqli":
		result, vulnFound = env.testSQLi()
	case "test_xss":
		result, vulnFound = env.testXSS()
	case "test_lfi":
		result, vulnFound = env.testLFI()
	case "test_command_injection":
		result, vulnFound = env.testCommandInjection()
	case "test_csrf":
		result, vulnFound = env.testCSRF()
	default:
		result = "Unknown action"
		vulnFound = false
	}

	// Compute reward
	reward := env.ComputeReward(action.Name, result, vulnFound)

	// Create next state
	nextState := env.CurrentState
	nextState.CurrentAction = action.Name
	nextState.LastResult = result
	nextState.TimeElapsed += 1

	if vulnFound {
		nextState.DiscoveredVulns++
	} else {
		nextState.FailedAttempts++
	}

	return nextState, reward, result
}

func (env *TestEnvironment) ComputeReward(actionName string, result string, vulnFound bool) Reward {
	rewardValue := 0.0
	reason := ""

	if vulnFound {
		rewardValue = 10.0 // Big reward for finding vulnerabilities!
		reason = "Vulnerability discovered!"
		fmt.Printf("🎉 REWARD +%.1f: %s\n", rewardValue, reason)
	} else if strings.Contains(result, "error") || strings.Contains(result, "failed") {
		rewardValue = -2.0 // Penalize failures
		reason = "Action failed"
		fmt.Printf("💥 REWARD %.1f: %s\n", rewardValue, reason)
	} else {
		rewardValue = 0.5 // Small reward for trying
		reason = "Action completed, no vulnerability found"
		fmt.Printf("🔍 REWARD +%.1f: %s\n", rewardValue, reason)
	}

	return Reward{
		Value:     rewardValue,
		Reason:    reason,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (env *TestEnvironment) testSQLi() (string, bool) {
	// Use our real scanner
	testURL := env.Target + "/vulnerabilities/sqli/?id=1%27+OR+%271%27=%271&Submit=Submit"
	body, _ := simpleHTTPGet(testURL)

	if strings.Contains(body, "First name") || strings.Contains(body, "Surname") {
		return "SQL injection vulnerability found!", true
	}

	return "No SQL injection found", false
}

func (env *TestEnvironment) testXSS() (string, bool) {
	testURL := env.Target + "/vulnerabilities/xss_r/?name=%3Cscript%3Ealert(1)%3C/script%3E"
	body, _ := simpleHTTPGet(testURL)

	if strings.Contains(body, "<script>alert(1)</script>") {
		return "XSS vulnerability found!", true
	}

	return "No XSS found", false
}

func (env *TestEnvironment) testLFI() (string, bool) {
	testURL := env.Target + "/vulnerabilities/fi/?page=../../../../etc/passwd"
	body, _ := simpleHTTPGet(testURL)

	if strings.Contains(body, "root:") || strings.Contains(body, "daemon:") {
		return "LFI vulnerability found!", true
	}

	return "No LFI found", false
}

func (env *TestEnvironment) testCommandInjection() (string, bool) {
	// Command injection needs POST, let's do a simple test for now
	return "Command injection check pending (needs login)", false
}

func (env *TestEnvironment) testCSRF() (string, bool) {
	return "CSRF check pending", false
}

func simpleHTTPGet(url string) (string, error) {
	return simpleScrapeHTTP(url)
}