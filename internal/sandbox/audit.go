package sandbox

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// AuditEvent 审计事件
type AuditEvent struct {
	EventID   string                 `json:"event_id"`
	Timestamp time.Time             `json:"timestamp"`
	SandboxID string                `json:"sandbox_id"`
	EventType string                `json:"event_type"` // create|execute|destroy|violation|cleanup
	Command   []string              `json:"command,omitempty"`
	Result    string                `json:"result"` // success|failure|blocked
	Violations []Violation          `json:"violations,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Violation 违规记录
type Violation struct {
	Type        string    `json:"type"` // resource_limit|network_blocked|forbidden_action
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Details     string    `json:"details"`
}

// AuditLogger 审计日志
type AuditLogger struct {
	mu         sync.Mutex
	events     []AuditEvent
	violations []Violation
}

// NewAuditLogger 创建审计日志器
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		events:     make([]AuditEvent, 0),
		violations: make([]Violation, 0),
	}
}

// LogEvent 记录事件
func (l *AuditLogger) LogEvent(event AuditEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	event.Timestamp = time.Now()
	l.events = append(l.events, event)
}

// LogViolation 记录违规
func (l *AuditLogger) LogViolation(violation Violation) {
	l.mu.Lock()
	defer l.mu.Unlock()

	violation.Timestamp = time.Now()
	l.violations = append(l.violations, violation)

	// 同时记录为事件
	l.events = append(l.events, AuditEvent{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		EventType: "violation",
		Result:    "blocked",
		Violations: []Violation{violation},
	})
}

// Events 获取所有事件
func (l *AuditLogger) Events() []AuditEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]AuditEvent, len(l.events))
	copy(result, l.events)
	return result
}

// Violations 获取所有违规
func (l *AuditLogger) Violations() []Violation {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]Violation, len(l.violations))
	copy(result, l.violations)
	return result
}

// ExportJSON 导出审计报告为 JSON
func (l *AuditLogger) ExportJSON() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	report := map[string]interface{}{
		"generated_at":   time.Now(),
		"total_events":   len(l.events),
		"total_violations": len(l.violations),
		"violations":     l.violations,
		"events":         l.events,
	}
	return json.MarshalIndent(report, "", "  ")
}
