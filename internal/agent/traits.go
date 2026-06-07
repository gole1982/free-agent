package agent

import "time"

type AgentTraits struct {
	Name          string
	Efficiency    float64
	Quality       float64
	Creativity    float64
	Collaboration float64
	LearningRate  float64
	UsageCount    int
	SuccessCount  int
}

type AgentTraitsAdjustment struct {
	AgentName       string
	EfficiencyDelta float64
	QualityDelta    float64
	CreativityDelta float64
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

var _ = time.Now
