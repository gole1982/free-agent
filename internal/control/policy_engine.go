package control

import (
	"encoding/json"
	"fmt"
	"os"
)

type PolicyType string

const (
	PolicyTypeHard       PolicyType = "hard"
	PolicyTypeSoft       PolicyType = "soft"
	PolicyTypeContextual PolicyType = "contextual"
)

type Policy struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        PolicyType  `json:"type"`
	Description string      `json:"description"`
	Rule        string      `json:"rule"`
	Action      PolicyAction `json:"action"`
	Enabled     bool        `json:"enabled"`
}

type PolicyAction string

const (
	ActionAllow    PolicyAction = "allow"
	ActionDeny     PolicyAction = "deny"
	ActionWarn     PolicyAction = "warn"
	ActionReview   PolicyAction = "review"
)

type PolicyEngine struct {
	policies []Policy
}

func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		policies: []Policy{},
	}
}

func (pe *PolicyEngine) LoadPoliciesFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var policies []Policy
	if err := json.Unmarshal(data, &policies); err != nil {
		return err
	}

	pe.policies = policies
	return nil
}

func (pe *PolicyEngine) AddPolicy(policy Policy) {
	pe.policies = append(pe.policies, policy)
}

func (pe *PolicyEngine) Evaluate(input string) PolicyResult {
	var results []PolicyMatch

	for _, policy := range pe.policies {
		if !policy.Enabled {
			continue
		}

		if matchesPolicy(input, policy.Rule) {
			results = append(results, PolicyMatch{
				Policy: policy,
				Match:  true,
			})
		}
	}

	return PolicyResult{
		Matches:  results,
		Decision: pe.makeDecision(results),
	}
}

func (pe *PolicyEngine) makeDecision(matches []PolicyMatch) PolicyAction {
	for _, match := range matches {
		if match.Policy.Type == PolicyTypeHard && match.Policy.Action == ActionDeny {
			return ActionDeny
		}
	}

	for _, match := range matches {
		if match.Policy.Action == ActionReview {
			return ActionReview
		}
	}

	for _, match := range matches {
		if match.Policy.Action == ActionWarn {
			return ActionWarn
		}
	}

	return ActionAllow
}

func matchesPolicy(input, rule string) bool {
	return containsIgnoreCase(input, rule)
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || 
		equalsIgnoreCase(s[:len(substr)], substr))
}

func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

type PolicyMatch struct {
	Policy Policy
	Match  bool
}

type PolicyResult struct {
	Matches  []PolicyMatch
	Decision PolicyAction
}

func (pr PolicyResult) String() string {
	return fmt.Sprintf("Decision: %s, Matches: %d", pr.Decision, len(pr.Matches))
}
