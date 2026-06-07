package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// Snapshot 快照
type Snapshot struct {
	ID        string    `json:"id"`
	SandboxID string    `json:"sandbox_id"`
	ImageRef  string    `json:"image_ref"`
	Created   time.Time `json:"created"`
	Label     string    `json:"label"`
}

// SnapshotManager 快照管理器
type SnapshotManager struct {
	mu        sync.Mutex
	snapshots map[string]*Snapshot // snapshotID -> Snapshot
}

// NewSnapshotManager 创建快照管理器
func NewSnapshotManager() *SnapshotManager {
	return &SnapshotManager{
		snapshots: make(map[string]*Snapshot),
	}
}

// CreateSnapshot 创建容器快照（通过 docker commit）
func (sm *SnapshotManager) CreateSnapshot(ctx context.Context, containerID string, containerName string, label string) (*Snapshot, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	snapshotID := fmt.Sprintf("snap-%d", time.Now().UnixNano())
	imageRef := fmt.Sprintf("vds-snapshot-%s:%s", containerName, snapshotID[:12])

	// 使用 docker commit 将运行中的容器提交为新镜像
	cmd := exec.CommandContext(ctx, "docker", "commit", containerName, imageRef)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker commit failed: %v, output: %s", err, string(output))
	}

	snapshot := &Snapshot{
		ID:        snapshotID,
		SandboxID: containerID,
		ImageRef:  imageRef,
		Created:   time.Now(),
		Label:     label,
	}

	sm.snapshots[snapshotID] = snapshot
	return snapshot, nil
}

// Rollback 回滚到指定快照（停止当前容器，从快照镜像启动新容器）
func (sm *SnapshotManager) Rollback(ctx context.Context, containerName string, snapshotID string) error {
	sm.mu.Lock()
	snapshot, ok := sm.snapshots[snapshotID]
	sm.mu.Unlock()

	if !ok {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	// 1. 停止当前容器
	stopCmd := exec.CommandContext(ctx, "docker", "stop", containerName)
	if output, err := stopCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("stop container failed: %v, output: %s", err, string(output))
	}

	// 2. 移除旧容器
	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", containerName)
	if output, err := rmCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove container failed: %v, output: %s", err, string(output))
	}

	// 3. 从快照镜像创建新容器
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d",
		"--name", containerName,
		snapshot.ImageRef,
	)
	if output, err := runCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create from snapshot failed: %v, output: %s", err, string(output))
	}

	return nil
}

// GetSnapshot 获取快照信息
func (sm *SnapshotManager) GetSnapshot(snapshotID string) (*Snapshot, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.snapshots[snapshotID]
	return s, ok
}

// ListSnapshots 列出所有快照
func (sm *SnapshotManager) ListSnapshots() []*Snapshot {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := make([]*Snapshot, 0, len(sm.snapshots))
	for _, s := range sm.snapshots {
		result = append(result, s)
	}
	return result
}

// DeleteSnapshot 删除快照（移除 docker 镜像）
func (sm *SnapshotManager) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	sm.mu.Lock()
	snapshot, ok := sm.snapshots[snapshotID]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	delete(sm.snapshots, snapshotID)
	sm.mu.Unlock()

	// 移除 docker 镜像
	cmd := exec.CommandContext(ctx, "docker", "rmi", "-f", snapshot.ImageRef)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("remove image failed: %v, output: %s", err, string(output))
	}

	return nil
}

// CleanupSnapshots 清理超过指定时间的快照
func (sm *SnapshotManager) CleanupSnapshots(ctx context.Context, maxAge time.Duration) int {
	sm.mu.Lock()
	var expired []string
	for id, s := range sm.snapshots {
		if time.Since(s.Created) > maxAge {
			expired = append(expired, id)
		}
	}
	sm.mu.Unlock()

	cleaned := 0
	for _, id := range expired {
		if err := sm.DeleteSnapshot(ctx, id); err == nil {
			cleaned++
		}
	}
	return cleaned
}
