package control

import (
	"fmt"
	"sync"
	"time"
)

type ConstraintType string

const (
	ConstraintTypeHard       ConstraintType = "hard"
	ConstraintTypeSoft       ConstraintType = "soft"
	ConstraintTypeContextual ConstraintType = "contextual"
)

type Constraint struct {
	ID          string
	Name        string
	Type        ConstraintType
	Description string
	Expression  string
	MaxValue    float64
	MinValue    float64
	Units       string
	Enabled     bool
	LastUpdated time.Time
}

type ConstraintViolation struct {
	Constraint  Constraint
	Value       float64
	Message     string
	Timestamp   time.Time
	Severity    SeverityLevel
}

type ConstraintManager struct {
	constraints map[string]*Constraint
	violations  []ConstraintViolation
	mu          sync.RWMutex
}

func NewConstraintManager() *ConstraintManager {
	return &ConstraintManager{
		constraints: map[string]*Constraint{
			"CT001": {
				ID:          "CT001",
				Name:        "Max Input Length",
				Type:        ConstraintTypeHard,
				Description: "Maximum allowed input length",
				Expression:  "len(input) <= 4096",
				MaxValue:    4096,
				MinValue:    0,
				Units:       "characters",
				Enabled:     true,
				LastUpdated: time.Now(),
			},
			"CT002": {
				ID:          "CT002",
				Name:        "Max Response Time",
				Type:        ConstraintTypeSoft,
				Description: "Maximum allowed response time",
				Expression:  "response_time <= 30",
				MaxValue:    30,
				MinValue:    0,
				Units:       "seconds",
				Enabled:     true,
				LastUpdated: time.Now(),
			},
			"CT003": {
				ID:          "CT003",
				Name:        "Daily Request Limit",
				Type:        ConstraintTypeHard,
				Description: "Maximum daily API requests",
				Expression:  "daily_requests <= 1000",
				MaxValue:    1000,
				MinValue:    0,
				Units:       "requests",
				Enabled:     true,
				LastUpdated: time.Now(),
			},
		},
		violations: []ConstraintViolation{},
	}
}

func (cm *ConstraintManager) AddConstraint(constraint *Constraint) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.constraints[constraint.ID]; exists {
		return fmt.Errorf("constraint %s already exists", constraint.ID)
	}

	constraint.LastUpdated = time.Now()
	cm.constraints[constraint.ID] = constraint
	return nil
}

func (cm *ConstraintManager) UpdateConstraint(constraint *Constraint) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.constraints[constraint.ID]; !exists {
		return fmt.Errorf("constraint %s not found", constraint.ID)
	}

	constraint.LastUpdated = time.Now()
	cm.constraints[constraint.ID] = constraint
	return nil
}

func (cm *ConstraintManager) RemoveConstraint(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.constraints[id]; !exists {
		return fmt.Errorf("constraint %s not found", id)
	}

	delete(cm.constraints, id)
	return nil
}

func (cm *ConstraintManager) Validate(value float64, constraintID string) (bool, *ConstraintViolation) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	constraint, exists := cm.constraints[constraintID]
	if !exists {
		return true, nil
	}

	if !constraint.Enabled {
		return true, nil
	}

	violation := ConstraintViolation{
		Constraint: *constraint,
		Value:      value,
		Timestamp:  time.Now(),
	}

	if value > constraint.MaxValue {
		violation.Message = fmt.Sprintf("Value %.2f exceeds maximum of %.2f %s", 
			value, constraint.MaxValue, constraint.Units)
		violation.Severity = getSeverity(constraint.Type)
		cm.mu.Lock()
		cm.violations = append(cm.violations, violation)
		cm.mu.Unlock()
		return false, &violation
	}

	if value < constraint.MinValue {
		violation.Message = fmt.Sprintf("Value %.2f is below minimum of %.2f %s", 
			value, constraint.MinValue, constraint.Units)
		violation.Severity = getSeverity(constraint.Type)
		cm.mu.Lock()
		cm.violations = append(cm.violations, violation)
		cm.mu.Unlock()
		return false, &violation
	}

	return true, nil
}

func (cm *ConstraintManager) GetViolations(limit int) []ConstraintViolation {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if limit <= 0 || limit > len(cm.violations) {
		return cm.violations
	}

	return cm.violations[:limit]
}

func (cm *ConstraintManager) GetViolationCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.violations)
}

func (cm *ConstraintManager) ClearViolations() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.violations = []ConstraintViolation{}
}

func getSeverity(constraintType ConstraintType) SeverityLevel {
	switch constraintType {
	case ConstraintTypeHard:
		return SeverityCritical
	case ConstraintTypeSoft:
		return SeverityWarning
	default:
		return SeverityMedium
	}
}

const SeverityWarning SeverityLevel = "warning"
