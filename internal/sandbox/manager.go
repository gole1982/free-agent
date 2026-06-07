package sandbox

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Container 沙箱容器
type Container struct {
	ID        string
	Name      string
	Type      string // "attacker" | "target" | "runner"
	Image     string
	Status    string // "created" | "running" | "stopped" | "expired"
	Created   time.Time
	ExpiresAt time.Time
	Limits    ResourceLimits
}

// Manager 沙箱管理器 — 管理多个 Container 的生命周期
type Manager struct {
	mu         sync.Mutex
	containers map[string]*Container
	config     *ManagerConfig
	audit      *AuditLogger
	policy     *PolicyEngine
	snapshots  *SnapshotManager
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	MaxConcurrent int           // 最大并发沙箱数
	DefaultTTL    time.Duration // 默认沙箱生存时间
	AutoCleanup   bool          // 自动清理过期沙箱
	CleanupInterval time.Duration // 清理检查间隔
}

// DefaultManagerConfig 默认管理器配置
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		MaxConcurrent:   3,
		DefaultTTL:      1 * time.Hour,
		AutoCleanup:     true,
		CleanupInterval: 5 * time.Minute,
	}
}

// NewManager 创建沙箱管理器
func NewManager(config *ManagerConfig, audit *AuditLogger, policy *PolicyEngine) *Manager {
	if config == nil {
		config = DefaultManagerConfig()
	}
	if audit == nil {
		audit = NewAuditLogger()
	}
	if policy == nil {
		policy = DefaultPolicyEngine()
	}
	return &Manager{
		containers: make(map[string]*Container),
		config:     config,
		audit:      audit,
		policy:     policy,
		snapshots:  NewSnapshotManager(),
	}
}

// CreateContainer 创建沙箱容器
func (m *Manager) CreateContainer(ctx context.Context, sandboxConfig SandboxConfig, containerType string) (*Container, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查并发限制
	if len(m.containers) >= m.config.MaxConcurrent {
		return nil, fmt.Errorf("max concurrent sandboxes reached (%d)", m.config.MaxConcurrent)
	}

	id := fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
	name := fmt.Sprintf("vds-%s-%s", containerType, id[:16])

	container := &Container{
		ID:        id,
		Name:      name,
		Type:      containerType,
		Image:     sandboxConfig.Image,
		Status:    "created",
		Created:   time.Now(),
		ExpiresAt: time.Now().Add(m.config.DefaultTTL),
		Limits: ResourceLimits{
			MemoryLimit: sandboxConfig.MemoryLimit,
			CPUQuota:    sandboxConfig.CPUQuota,
		},
	}

	m.containers[id] = container

	// 审计日志
	m.audit.LogEvent(AuditEvent{
		EventType: "create",
		SandboxID: id,
		Result:    "success",
		Metadata:  map[string]interface{}{"type": containerType, "image": sandboxConfig.Image},
	})

	return container, nil
}

// DestroyContainer 销毁沙箱容器
func (m *Manager) DestroyContainer(ctx context.Context, id string) error {
	m.mu.Lock()
	container, ok := m.containers[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("sandbox not found: %s", id)
	}
	delete(m.containers, id)
	m.mu.Unlock()

	container.Status = "stopped"

	// 执行 Docker 清理
	sandbox := NewSandbox(SandboxConfig{Image: container.Image})
	sandbox.cleanup(container.Name)

	m.audit.LogEvent(AuditEvent{
		EventType: "destroy",
		SandboxID: id,
		Result:    "success",
	})

	return nil
}

// GetContainer 获取容器信息
func (m *Manager) GetContainer(id string) (*Container, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.containers[id]
	return c, ok
}

// ListContainers 列出所有容器
func (m *Manager) ListContainers() []*Container {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*Container, 0, len(m.containers))
	for _, c := range m.containers {
		result = append(result, c)
	}
	return result
}

// CleanupExpired 清理过期容器
func (m *Manager) CleanupExpired(ctx context.Context) int {
	m.mu.Lock()
	var expired []string
	for id, c := range m.containers {
		if time.Now().After(c.ExpiresAt) {
			expired = append(expired, id)
		}
	}
	m.mu.Unlock()

	cleaned := 0
	for _, id := range expired {
		if err := m.DestroyContainer(ctx, id); err == nil {
			cleaned++
		}
	}

	if cleaned > 0 {
		m.audit.LogEvent(AuditEvent{
			EventType: "cleanup",
			Result:    "success",
			Metadata:  map[string]interface{}{"cleaned": cleaned},
		})
	}
	return cleaned
}

// StartAutoCleanup 启动自动清理（在独立 goroutine 中运行）
func (m *Manager) StartAutoCleanup(ctx context.Context) {
	if !m.config.AutoCleanup {
		return
	}
	go func() {
		ticker := time.NewTicker(m.config.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.CleanupExpired(ctx)
			}
		}
	}()
}

// AuditLogger 返回审计日志器
func (m *Manager) AuditLogger() *AuditLogger {
	return m.audit
}

// PolicyEngine 返回策略引擎
func (m *Manager) PolicyEngine() *PolicyEngine {
	return m.policy
}

// SnapshotManager 返回快照管理器
func (m *Manager) SnapshotManager() *SnapshotManager {
	return m.snapshots
}

// CreateSnapshot 为指定容器创建快照
func (m *Manager) CreateSnapshot(ctx context.Context, containerID string, label string) (*Snapshot, error) {
	m.mu.Lock()
	c, ok := m.containers[containerID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}
	snap, err := m.snapshots.CreateSnapshot(ctx, containerID, c.Name, label)
	if err != nil {
		return nil, err
	}
	m.audit.LogEvent(AuditEvent{
		EventType: "snapshot",
		SandboxID: containerID,
		Result:    "success",
		Metadata:  map[string]interface{}{"snapshot_id": snap.ID, "label": label},
	})
	return snap, nil
}

// RollbackContainer 将容器回滚到指定快照
func (m *Manager) RollbackContainer(ctx context.Context, containerID string, snapshotID string) error {
	m.mu.Lock()
	c, ok := m.containers[containerID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("container not found: %s", containerID)
	}
	if err := m.snapshots.Rollback(ctx, c.Name, snapshotID); err != nil {
		return err
	}
	m.audit.LogEvent(AuditEvent{
		EventType: "rollback",
		SandboxID: containerID,
		Result:    "success",
		Metadata:  map[string]interface{}{"snapshot_id": snapshotID},
	})
	return nil
}
