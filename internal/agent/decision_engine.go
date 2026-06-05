package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type RiskLevel int

const (
	RiskLow RiskLevel = iota
	RiskMedium
	RiskHigh
	RiskCritical
)

type StrategyPattern struct {
	ID          string
	Pattern     string
	Strategy    string
	SuccessRate float64
	Uses        int
	LastUsed    time.Time
	RiskLevel   RiskLevel
}

type DecisionEngine struct {
	mu           sync.RWMutex
	patterns     []StrategyPattern
	defaultRisk  RiskLevel
	confidenceThreshold float64
}

func NewDecisionEngine() *DecisionEngine {
	return &DecisionEngine{
		patterns:             make([]StrategyPattern, 0),
		defaultRisk:          RiskMedium,
		confidenceThreshold: 0.6,
	}
}

func (d *DecisionEngine) AddPattern(pattern StrategyPattern) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	for i, p := range d.patterns {
		if p.ID == pattern.ID {
			d.patterns[i] = pattern
			return
		}
	}
	d.patterns = append(d.patterns, pattern)
}

func (d *DecisionEngine) UpdatePatternSuccess(patternID string, success bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	for i, p := range d.patterns {
		if p.ID == patternID {
			d.patterns[i].Uses++
			if success {
				d.patterns[i].SuccessRate = 
					(p.SuccessRate*(float64(p.Uses-1)) + 1) / float64(p.Uses)
			} else {
				d.patterns[i].SuccessRate = 
					p.SuccessRate * (float64(p.Uses-1)) / float64(p.Uses)
			}
			d.patterns[i].LastUsed = time.Now()
			return
		}
	}
}

func (d *DecisionEngine) SuggestStrategies(task string, maxStrategies int) []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	var suggestions []string
	taskLower := strings.ToLower(task)
	
	for _, pattern := range d.patterns {
		if strings.Contains(taskLower, strings.ToLower(pattern.Pattern)) {
			suggestions = append(suggestions, pattern.Strategy)
		}
	}
	
	if len(suggestions) == 0 {
		suggestions = d.getDefaultStrategies()
	}
	
	if len(suggestions) > maxStrategies {
		suggestions = suggestions[:maxStrategies]
	}
	
	return suggestions
}

func (d *DecisionEngine) getDefaultStrategies() []string {
	return []string{"analysis", "scanning", "exploitation", "verification"}
}

func (d *DecisionEngine) EvaluateRisk(task string) RiskLevel {
	taskLower := strings.ToLower(task)
	
	if strings.Contains(taskLower, "critical") || 
	   strings.Contains(taskLower, "emergency") ||
	   strings.Contains(taskLower, "exploit") ||
	   strings.Contains(taskLower, "bypass") {
		return RiskCritical
	}
	
	if strings.Contains(taskLower, "vulnerability") ||
	   strings.Contains(taskLower, "security") ||
	   strings.Contains(taskLower, "attack") {
		return RiskHigh
	}
	
	if strings.Contains(taskLower, "debug") ||
	   strings.Contains(taskLower, "test") ||
	   strings.Contains(taskLower, "review") {
		return RiskMedium
	}
	
	return RiskLow
}

func (d *DecisionEngine) CalculatePriority(strategy string, risk RiskLevel) int {
	priority := 50
	
	switch risk {
	case RiskCritical:
		priority += 30
	case RiskHigh:
		priority += 20
	case RiskMedium:
		priority += 10
	}
	
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	for _, pattern := range d.patterns {
		if strings.EqualFold(pattern.Strategy, strategy) {
			if pattern.SuccessRate > 0.8 {
				priority += 15
			} else if pattern.SuccessRate < 0.3 {
				priority -= 15
			}
			
			if pattern.Uses > 10 {
				priority += 5
			}
			break
		}
	}
	
	return priority
}

func (d *DecisionEngine) SelectBestStrategy(strategies []string, task string) string {
	if len(strategies) == 0 {
		return "analysis"
	}
	
	risk := d.EvaluateRisk(task)
	bestStrategy := strategies[0]
	bestPriority := 0
	
	for _, strategy := range strategies {
		priority := d.CalculatePriority(strategy, risk)
		if priority > bestPriority {
			bestPriority = priority
			bestStrategy = strategy
		}
	}
	
	return bestStrategy
}

func (d *DecisionEngine) ShouldContinue(attempts int, successRate float64) bool {
	if attempts >= 5 {
		return false
	}
	
	if successRate < 0.2 && attempts >= 2 {
		return false
	}
	
	return true
}

func (d *DecisionEngine) GetPatternsByRisk(risk RiskLevel) []StrategyPattern {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	var result []StrategyPattern
	for _, pattern := range d.patterns {
		if pattern.RiskLevel == risk {
			result = append(result, pattern)
		}
	}
	return result
}

func (d *DecisionEngine) GetStats() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	var sb strings.Builder
	sb.WriteString("=== Decision Engine Stats ===\n")
	sb.WriteString(fmt.Sprintf("Total patterns: %d\n", len(d.patterns)))
	
	for _, pattern := range d.patterns {
		sb.WriteString(fmt.Sprintf("  - %s: %.1f%% success rate, %d uses\n", 
			pattern.Pattern, pattern.SuccessRate*100, pattern.Uses))
	}
	
	return sb.String()
}