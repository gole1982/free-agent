package control

import (
	"fmt"
	"sync"
	"time"
)

type CoreControl struct {
	policyEngine       *PolicyEngine
	safetyGuardrails   *SafetyGuardrails
	permissionSystem   *PermissionSystem
	constraintManager  *ConstraintManager
	pentestingMode     bool
	mu                 sync.RWMutex
}

func NewCoreControl() *CoreControl {
	return &CoreControl{
		policyEngine:      NewPolicyEngine(),
		safetyGuardrails:  NewSafetyGuardrails(),
		permissionSystem:  NewPermissionSystem(),
		constraintManager: NewConstraintManager(),
		pentestingMode:    false,
	}
}

func NewCoreControlWithPentesting(pentesting bool) *CoreControl {
	return &CoreControl{
		policyEngine:      NewPolicyEngine(),
		safetyGuardrails:  NewSafetyGuardrails(),
		permissionSystem:  NewPermissionSystem(),
		constraintManager: NewConstraintManager(),
		pentestingMode:    pentesting,
	}
}

func (cc *CoreControl) Initialize() error {
	defaultUser := &User{
		ID:   "default",
		Name: "Default User",
		Role: RoleUser,
	}
	return cc.permissionSystem.AddUser(defaultUser)
}

func (cc *CoreControl) PreProcess(input string) (*ControlResult, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result := &ControlResult{
		Allowed:    true,
		Messages:   []string{},
		Warnings:   []string{},
		Errors:     []string{},
	}

	if cc.pentestingMode {
		result.Messages = append(result.Messages, "[Pentest Mode] Security restrictions are relaxed")
	}

	safetyResult := cc.safetyGuardrails.Inspect(input)
	if safetyResult.Blocked {
		if cc.pentestingMode {
			result.Warnings = append(result.Warnings, "[Pentest Mode] Blocked content allowed: "+safetyResult.Message)
		} else {
			result.Allowed = false
			result.Errors = append(result.Errors, safetyResult.Message)
			return result, nil
		}
	} else if safetyResult.Message != "" {
		result.Warnings = append(result.Warnings, safetyResult.Message)
	}

	policyResult := cc.policyEngine.Evaluate(input)
	if policyResult.Decision == ActionDeny {
		if cc.pentestingMode {
			result.Warnings = append(result.Warnings, "[Pentest Mode] Policy violation allowed")
		} else {
			result.Allowed = false
			result.Errors = append(result.Errors, "Policy violation detected")
			return result, nil
		}
	}

	if policyResult.Decision == ActionReview {
		result.Warnings = append(result.Warnings, "Requires review")
	}

	if policyResult.Decision == ActionWarn {
		result.Warnings = append(result.Warnings, "Policy warning")
	}

	inputLength := float64(len(input))
	valid, violation := cc.constraintManager.Validate(inputLength, "CT001")
	if !valid && violation != nil {
		if cc.pentestingMode {
			result.Warnings = append(result.Warnings, "[Pentest Mode] Constraint violation allowed: "+violation.Message)
		} else {
			result.Warnings = append(result.Warnings, violation.Message)
		}
	}

	return result, nil
}

func (cc *CoreControl) PostProcess(output string) (*ControlResult, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	result := &ControlResult{
		Allowed:    true,
		Messages:   []string{},
		Warnings:   []string{},
		Errors:     []string{},
	}

	safetyResult := cc.safetyGuardrails.Inspect(output)
	if safetyResult.Blocked {
		result.Allowed = false
		result.Errors = append(result.Errors, "Output blocked by safety guardrails")
		return result, nil
	}

	if safetyResult.Message != "" {
		result.Warnings = append(result.Warnings, safetyResult.Message)
	}

	return result, nil
}

func (cc *CoreControl) CheckAccess(userID, action string) (bool, string) {
	return cc.permissionSystem.CheckAccess(userID, action)
}

func (cc *CoreControl) AddPolicy(policy Policy) {
	cc.policyEngine.AddPolicy(policy)
}

func (cc *CoreControl) AddGuardrail(guardrail SafetyGuardrail) {
	cc.safetyGuardrails.AddGuardrail(guardrail)
}

func (cc *CoreControl) AddConstraint(constraint *Constraint) error {
	return cc.constraintManager.AddConstraint(constraint)
}

func (cc *CoreControl) GetViolations(limit int) []ConstraintViolation {
	return cc.constraintManager.GetViolations(limit)
}

func (cc *CoreControl) GetStats() ControlStats {
	return ControlStats{
		PolicyCount:        len(cc.policyEngine.policies),
		GuardrailCount:     len(cc.safetyGuardrails.guardrails),
		ConstraintCount:    len(cc.constraintManager.constraints),
		ViolationCount:     cc.constraintManager.GetViolationCount(),
		UserCount:          len(cc.permissionSystem.users),
		LastChecked:        time.Now(),
	}
}

type ControlResult struct {
	Allowed  bool
	Messages []string
	Warnings []string
	Errors   []string
}

type ControlStats struct {
	PolicyCount        int
	GuardrailCount     int
	ConstraintCount    int
	ViolationCount     int
	UserCount          int
	LastChecked        time.Time
}

func (cs ControlStats) String() string {
	return fmt.Sprintf(`Control Statistics:
  Policies: %d
  Guardrails: %d
  Constraints: %d
  Violations: %d
  Users: %d
  Last Checked: %s`,
		cs.PolicyCount,
		cs.GuardrailCount,
		cs.ConstraintCount,
		cs.ViolationCount,
		cs.UserCount,
		cs.LastChecked.Format(time.RFC3339),
	)
}
