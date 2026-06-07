package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func extractJSONObject(text string) (string, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```JSON")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	start := strings.Index(text, "{")
	if start < 0 {
		return "", fmt.Errorf("no JSON object found")
	}

	depth, end := 0, -1
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return "", fmt.Errorf("unbalanced JSON braces")
	}
	return text[start : end+1], nil
}

func parseObserverDecision(response string) (ObserverDecision, error) {
	raw, err := extractJSONObject(response)
	if err != nil {
		return ObserverDecision{}, err
	}
	var wire struct {
		ShouldStop      bool    `json:"should_stop"`
		Reason          string  `json:"reason"`
		CorrectGuidance string  `json:"correct_guidance"`
		IntentAlignment float64 `json:"intent_alignment"`
		IsDeadLoop      bool    `json:"is_dead_loop"`
		IsMalicious     bool    `json:"is_malicious"`
		IsHoneypot      bool    `json:"is_honeypot"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return ObserverDecision{}, fmt.Errorf("parse observer json: %w", err)
	}
	return ObserverDecision{
		ShouldStop:      wire.ShouldStop,
		Reason:          wire.Reason,
		CorrectGuidance: wire.CorrectGuidance,
		IntentAlignment: wire.IntentAlignment,
		IsDeadLoop:      wire.IsDeadLoop,
		IsMalicious:     wire.IsMalicious,
		IsHoneypot:      wire.IsHoneypot,
	}, nil
}

func parseEvaluationConclusion(response string) (EvaluationConclusion, error) {
	raw, err := extractJSONObject(response)
	if err != nil {
		return EvaluationConclusion{}, err
	}
	var wire struct {
		ExitType      string `json:"exit_type"`
		NeedsUpdate   bool   `json:"needs_update"`
		UpdateType    string `json:"update_type"`
		UpdateContent string `json:"update_content"`
		TaskCompleted bool   `json:"task_completed"`
		RetryNeeded   bool   `json:"retry_needed"`
		RetryGuidance string `json:"retry_guidance"`
		AgentAdjustments []struct {
			AgentName       string  `json:"agent_name"`
			EfficiencyDelta float64 `json:"efficiency_delta"`
			QualityDelta    float64 `json:"quality_delta"`
			CreativityDelta float64 `json:"creativity_delta"`
		} `json:"agent_adjustments"`
	}
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		return EvaluationConclusion{}, fmt.Errorf("parse evaluation json: %w", err)
	}
	conclusion := EvaluationConclusion{
		ExitType:      wire.ExitType,
		NeedsUpdate:   wire.NeedsUpdate,
		UpdateType:    wire.UpdateType,
		UpdateContent: wire.UpdateContent,
		TaskCompleted: wire.TaskCompleted,
		RetryNeeded:   wire.RetryNeeded,
		RetryGuidance: wire.RetryGuidance,
	}
	for _, a := range wire.AgentAdjustments {
		conclusion.AgentAdjustments = append(conclusion.AgentAdjustments, &AgentTraitsAdjustment{
			AgentName:       a.AgentName,
			EfficiencyDelta: a.EfficiencyDelta,
			QualityDelta:    a.QualityDelta,
			CreativityDelta: a.CreativityDelta,
		})
	}
	return conclusion, nil
}
