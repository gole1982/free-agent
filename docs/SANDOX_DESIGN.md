# 网络安全测试沙箱设计方案

## 1. 沙箱架构概览

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Host System (宿主机)                          │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                    VDS (Vulnerability Discovery System)             │  │
│  │  ┌─────────────────────────────────────────────────────────────┐ │  │
│  │  │                      Sandbox Manager                          │ │  │
│  │  │  - 资源配额管理    - 生命周期控制    - 审计日志              │ │  │
│  │  └─────────────────────────────────────────────────────────────┘ │  │
│  └─────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                    │                    │                    │
                    ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Sandbox Isolation Layers                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │   Network    │  │    Filesystem    │  │    Process   │  │   Syscall  │ │
│  │  Isolation   │  │    (chroot)    │  │   (cgroups)  │  │  (seccomp) │ │
│  │              │  │                │  │              │  │            │ │
│  │ • VPN隔离    │  │ • 虚拟文件系统 │  │ • CPU限制    │  │ • 系统调用 │ │
│  │ • 流量监控   │  │ • 写保护      │  │ • 内存限制   │  │   过滤    │ │
│  │ • DNS过滤    │  │ • 快照回滚    │  │ • 进程数限制 │  │            │ │
│  └──────────────┘  └──────────────┘  └──────────────┘  └──────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────────┘
                    │                    │                    │
                    ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Container / VM Layer                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                      Docker Container                           │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │  │
│  │  │ Kali Linux  │  │ Ubuntu      │  │ 隔离靶机    │          │  │
│  │  │ (攻击者)    │  │ (工具运行)  │  │ (DVWA)      │          │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. 分层隔离设计

### 2.1 网络隔离层

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Network Isolation                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                      │
│    Internet                                                         │
│        │                                                            │
│        ▼                                                            │
│   ┌─────────┐                                                       │
│   │  FW/IDS │ ← 流量监控 & 异常检测                                  │
│   └─────────┘                                                       │
│        │                                                            │
│        ▼                                                            │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │                    Isolated Network (VPN/VLAN)                │   │
│   │                                                             │   │
│   │   ┌──────────────┐         ┌──────────────┐               │   │
│   │   │ Attacker Net │◄───────►│  Target Net   │               │   │
│   │   │ (攻击者)     │         │ (隔离靶机)    │               │   │
│   │   │ 10.0.1.0/24  │         │ 10.0.2.0/24   │               │   │
│   │   └──────────────┘         └──────────────┘               │   │
│   │                                                             │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│   网络规则:                                                          │
│   • 靶机只能访问内网，禁止访问外网                                    │
│   • 攻击者可以访问靶机和有限外网（如漏洞库）                          │
│   • 所有流量经过IDS监控                                               │
│                                                                      │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 容器隔离层

```yaml
# docker-compose.yml
version: '3.8'

services:
  # 攻击者容器 - Kali Linux
  attacker:
    image: kalilinux/kali-rolling
    container_name: vds-attacker
    hostname: attacker
    networks:
      - attackernet
    cap_add:
      - NET_ADMIN  # 网络管理权限（仅沙箱内）
    volumes:
      - ./tools:/tools:ro  # 只读挂载工具
      - ./reports:/reports  # 报告输出
    environment:
      - SANDBOX_MODE=true
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 1G
    restart: unless-stopped

  # 靶机容器 - DVWA
  target-dvwa:
    image: vulnerables/web-dvwa
    container_name: vds-target-dvwa
    hostname: target-dvwa
    networks:
      - targetnet
    cap_drop:
      - ALL  # 移除所有特权
    read_only: true  # 只读文件系统
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=100m
      - /var/tmp:rw,noexec,nosuid,size=50m
    restart: unless-stopped

  # 网络隔离
networks:
  attackernet:
    driver: bridge
    ipam:
      config:
        - subnet: 10.0.1.0/24
  targetnet:
    driver: bridge
    internal: true  # 完全隔离，无外网访问
```

### 2.3 资源限制层

```go
// sandbox/resource_limits.go

package sandbox

import (
    "github.com/containerd/cgroups/v3"
)

// ResourceLimits 资源限制配置
type ResourceLimits struct {
    // CPU限制
    CPUQuota    int64  // CPU配额 (百分比)
    CPUPeriod   int64  // CPU周期
    CPUShares   int64  // CPU权重
    
    // 内存限制
    MemoryLimit int64  // 内存限制 (bytes)
    MemorySwap  int64  // swap限制
    
    // 进程限制
    PIDsLimit   int64  // 最大进程数
    
    // 磁盘限制
    DiskQuota   int64  // 磁盘配额 (bytes)
    DiskRate    int    // IO速率限制 (MB/s)
    
    // 网络限制
    NetRate     int    // 网络速率限制 (Mbps)
}

// DefaultLimits 默认安全限制
var DefaultLimits = ResourceLimits{
    CPUQuota:    200,      // 最多使用2个CPU核心
    CPUPeriod:   100000,   // 100ms周期
    CPUShares:   512,      // CPU权重
    MemoryLimit: 4 << 30,  // 4GB
    MemorySwap:  4 << 30,  // 禁用swap
    PIDsLimit:   100,      // 最多100个进程
    DiskQuota:   10 << 30, // 10GB
    DiskRate:    10,       // 10MB/s
    NetRate:     100,      // 100Mbps
}

// ApplyLimits 应用资源限制
func (l *ResourceLimits) ApplyLimits() error {
    // 使用cgroups v2
    cg, err := cgroups.New(cgroups.V2, cgroups.RootPath, cgroups.Spec{})
    if err != nil {
        return err
    }
    
    // 应用CPU限制
    cg.Add(cgroups.CPUController, cgroups.NewLimiter(cgroups.CPU, l.CPUQuota))
    
    // 应用内存限制
    cg.Add(cgroups.MemoryController, cgroups.NewLimiter(cgroups.Memory, l.MemoryLimit))
    
    // 应用进程数限制
    cg.Add(cgroups.PIDController, cgroups.NewLimiter(cgroups.PIDs, l.PIDsLimit))
    
    return nil
}
```

---

## 3. 沙箱管理器实现

```go
// sandbox/manager.go

package sandbox

import (
    "context"
    "fmt"
    "time"
    
    "github.com/docker/docker/client"
    "github.com/google/uuid"
)

// Manager 沙箱管理器
type Manager struct {
    dockerClient *client.Client
    config       *Config
    containers   map[string]*Container
}

// Config 沙箱配置
type Config struct {
    NetworkIsolation bool          // 网络隔离
    FilesystemRO    bool          // 文件系统只读
    TimeLimit       time.Duration // 时间限制
    ResourceLimits  ResourceLimits // 资源限制
    AutoCleanup     bool          // 自动清理
}

// Container 沙箱容器
type Container struct {
    ID        string
    Name      string
    Type      string  // "attacker" | "target"
    Image     string
    Status    string
    Created   time.Time
    ExpiresAt time.Time
    Limits    ResourceLimits
}

// NewManager 创建沙箱管理器
func NewManager(config *Config) (*Manager, error) {
    cli, err := client.NewClientWithOpts(client.FromEnv)
    if err != nil {
        return nil, fmt.Errorf("failed to create docker client: %v", err)
    }
    
    return &Manager{
        dockerClient: cli,
        config:       config,
        containers:   make(map[string]*Container),
    }, nil
}

// CreateSandbox 创建沙箱
func (m *Manager) CreateSandbox(ctx context.Context, sandboxType string) (*Container, error) {
    sandboxID := uuid.New().String()
    containerName := fmt.Sprintf("vds-%s-%s", sandboxType, sandboxID[:8])
    
    // 选择镜像
    image := m.selectImage(sandboxType)
    
    // 创建容器
    resp, err := m.dockerClient.ContainerCreate(ctx, &container.Config{
        Image: image,
        HostConfig: &container.HostConfig{
            NetworkMode: m.getNetworkMode(sandboxType),
            ReadonlyRootfs: m.config.FilesystemRO,
            Resources: container.Resources{
                Memory: m.config.ResourceLimits.MemoryLimit,
                NanoCPUs: m.config.ResourceLimits.CPUQuota * 10000000,
                PidsLimit: &m.config.ResourceLimits.PIDsLimit,
            },
        },
    }, nil, nil, containerName)
    if err != nil {
        return nil, fmt.Errorf("failed to create container: %v", err)
    }
    
    container := &Container{
        ID:        resp.ID,
        Name:      containerName,
        Type:      sandboxType,
        Image:     image,
        Status:    "created",
        Created:   time.Now(),
        ExpiresAt: time.Now().Add(m.config.TimeLimit),
        Limits:    m.config.ResourceLimits,
    }
    
    m.containers[sandboxID] = container
    return container, nil
}

// ExecuteInSandbox 在沙箱中执行命令
func (m *Manager) ExecuteInSandbox(ctx context.Context, sandboxID string, cmd []string) (*ExecutionResult, error) {
    container, ok := m.containers[sandboxID]
    if !ok {
        return nil, fmt.Errorf("sandbox not found: %s", sandboxID)
    }
    
    // 检查是否过期
    if time.Now().After(container.ExpiresAt) {
        return nil, fmt.Errorf("sandbox expired")
    }
    
    // 执行命令
    execResp, err := m.dockerClient.ContainerExecCreate(ctx, container.ID, container.ExecConfig{
        AttachStdout: true,
        AttachStderr: true,
        Cmd:          cmd,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create exec: %v", err)
    }
    
    // 附加到执行
    resp, err := m.dockerClient.ContainerExecAttach(ctx, execResp.ID, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to attach exec: %v", err)
    }
    defer resp.Close()
    
    // 读取输出
    result := &ExecutionResult{}
    // ... 解析输出
    
    return result, nil
}

// DestroySandbox 销毁沙箱
func (m *Manager) DestroySandbox(ctx context.Context, sandboxID string) error {
    container, ok := m.containers[sandboxID]
    if !ok {
        return fmt.Errorf("sandbox not found: %s", sandboxID)
    }
    
    // 强制停止容器
    timeout := 10 * time.Second
    if err := m.dockerClient.ContainerStop(ctx, container.ID, &timeout); err != nil {
        return fmt.Errorf("failed to stop container: %v", err)
    }
    
    // 删除容器
    if err := m.dockerClient.ContainerRemove(ctx, container.ID, container.RemoveOptions{
        Force: true,
    }); err != nil {
        return fmt.Errorf("failed to remove container: %v", err)
    }
    
    delete(m.containers, sandboxID)
    return nil
}

// CleanupExpired 清理过期沙箱
func (m *Manager) CleanupExpired(ctx context.Context) error {
    for id, container := range m.containers {
        if time.Now().After(container.ExpiresAt) {
            if err := m.DestroySandbox(ctx, id); err != nil {
                // 记录错误但继续清理其他
                continue
            }
        }
    }
    return nil
}
```

---

## 4. 审计系统

```go
// sandbox/audit.go

package sandbox

import (
    "encoding/json"
    "fmt"
    "time"
)

// AuditEvent 审计事件
type AuditEvent struct {
    EventID     string                 `json:"event_id"`
    Timestamp   time.Time             `json:"timestamp"`
    SandboxID   string                `json:"sandbox_id"`
    EventType   string                `json:"event_type"`  // create|execute|destroy|violation
    Command     []string              `json:"command,omitempty"`
    UserID      string                `json:"user_id"`
    SourceIP    string                `json:"source_ip"`
    Result      string                `json:"result"`      // success|failure|blocked
    Violations  []Violation           `json:"violations,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Violation 违规记录
type Violation struct {
    Type        string    `json:"type"`     // resource_limit|network_blocked|forbidden_action
    Description string    `json:"description"`
    Timestamp   time.Time `json:"timestamp"`
    Details     string    `json:"details"`
}

// AuditLogger 审计日志
type AuditLogger struct {
    events    []AuditEvent
    violations []Violation
}

// LogEvent 记录事件
func (l *AuditLogger) LogEvent(event AuditEvent) {
    event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
    l.events = append(l.events, event)
    
    // 持久化到文件
    l.persist(event)
}

// LogViolation 记录违规
func (l *AuditLogger) LogViolation(violation Violation) {
    l.violations = append(l.violations, violation)
    
    // 触发告警
    l.alert(violation)
}

// ExportAudit 导出审计报告
func (l *AuditLogger) ExportAudit() ([]byte, error) {
    report := map[string]interface{}{
        "generated_at": time.Now(),
        "total_events": len(l.events),
        "violations":   l.violations,
        "events":        l.events,
    }
    return json.MarshalIndent(report, "", "  ")
}

// persist 持久化事件
func (l *AuditLogger) persist(event AuditEvent) {
    // 写入审计日志文件
    // 写入审计数据库
}

// alert 发送告警
func (l *AuditLogger) alert(violation Violation) {
    // 发送邮件告警
    // 发送webhook
    // 记录日志
}
```

---

## 5. 安全策略配置

```yaml
# sandbox-policy.yaml

sandbox:
  # 通用配置
  defaults:
    timeout: 3600          # 默认1小时超时
    auto_cleanup: true      # 自动清理
    max_concurrent: 3      # 最大并发沙箱数
  
  # 网络策略
  network:
    isolation_mode: "vpn"   # vpn | vlan | none
    allow_internet: false   # 是否允许访问互联网
    blocked_ports:
      - 22                 # 禁用SSH
      - 3389               # 禁用RDP
    allowed_domains:
      - "*.exploit-db.com"
      - "*.securityfocus.com"
    rate_limit: 100        # Mbps
  
  # 文件系统策略
  filesystem:
    readonly: true
    allowed_paths:
      - "/tools"
      - "/reports"
    blocked_paths:
      - "/etc/shadow"
      - "/root/.ssh"
      - "/boot"
    max_disk_usage: 10GB
  
  # 资源限制
  resources:
    cpu_limit: 2           # CPU核心数
    memory_limit: 4GB
    process_limit: 100
    disk_io_limit: 10MB/s
  
  # 命令过滤
  commands:
    allowed:
      - "nmap"
      - "sqlmap"
      - "nikto"
      - "hydra"
    blocked:
      - "rm -rf /"
      - "dd if=/dev/zero"
      - ":(){ :|:& };:"    # Fork炸弹
      - "mkfs"
      - "fdisk"
      - "parted"
  
  # 危险操作告警
  alerts:
    enabled: true
    channels:
      - type: "email"
        recipients: ["security@example.com"]
      - type: "webhook"
        url: "https://internal.example.com/alerts"
    severity_threshold: "medium"
```

---

## 6. 快照与回滚

```go
// sandbox/snapshot.go

package sandbox

import (
    "context"
    "fmt"
    
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/api/types/mount"
)

// Snapshot 快照
type Snapshot struct {
    ID        string
    SandboxID string
    Created   time.Time
    Layers    []string  // Docker层ID
}

// CreateSnapshot 创建快照
func (m *Manager) CreateSnapshot(ctx context.Context, sandboxID string) (*Snapshot, error) {
    container, ok := m.containers[sandboxID]
    if !ok {
        return nil, fmt.Errorf("sandbox not found")
    }
    
    // 提交容器为新镜像
    resp, err := m.dockerClient.ContainerCommit(ctx, container.ID, container.CommitOptions{
        Reference: fmt.Sprintf("vds-snapshot-%s", sandboxID),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create snapshot: %v", err)
    }
    
    snapshot := &Snapshot{
        ID:        resp.ID,
        SandboxID: sandboxID,
        Created:   time.Now(),
    }
    
    return snapshot, nil
}

// Rollback 回滚到快照
func (m *Manager) Rollback(ctx context.Context, sandboxID string, snapshotID string) error {
    container, ok := m.containers[sandboxID]
    if !ok {
        return fmt.Errorf("sandbox not found")
    }
    
    // 停止当前容器
    timeout := 10 * time.Second
    if err := m.dockerClient.ContainerStop(ctx, container.ID, &timeout); err != nil {
        return fmt.Errorf("failed to stop container: %v", err)
    }
    
    // 从快照创建新容器
    newResp, err := m.dockerClient.ContainerCreate(ctx, &container.Config{
        Image: snapshotID,
    }, &container.HostConfig{
        Mounts: []mount.Mount{},  // 保持挂载
    }, nil, container.Name)
    if err != nil {
        return fmt.Errorf("failed to create container from snapshot: %v", err)
    }
    
    // 启动新容器
    if err := m.dockerClient.ContainerStart(ctx, newResp.ID, container.StartOptions{}); err != nil {
        return fmt.Errorf("failed to start container: %v", err)
    }
    
    // 更新容器记录
    container.ID = newResp.ID
    container.Status = "running"
    
    return nil
}
```

---

## 7. 实施路线图

| 阶段 | 内容 | 优先级 |
|------|------|--------|
| **Phase 1** | Docker容器隔离 + 资源限制 | P0 |
| **Phase 2** | 网络隔离 (VLAN/VPN) | P0 |
| **Phase 3** | 审计日志系统 | P1 |
| **Phase 4** | 快照与回滚 | P1 |
| **Phase 5** | 命令过滤 + 告警 | P2 |
| **Phase 6** | VM级隔离 (可选) | P3 |

---

## 8. 与VDS集成

```go
// vds/secure_orchestrator.go

// SecureOrchestrator 安全编排器
type SecureOrchestrator struct {
    sandboxManager *sandbox.Manager
    auditLogger    *sandbox.AuditLogger
    policyEngine   *PolicyEngine
}

// ExecuteSecureScan 安全扫描
func (o *SecureOrchestrator) ExecuteSecureScan(ctx context.Context, target string) (*ScanReport, error) {
    // 1. 创建沙箱
    sandbox, err := o.sandboxManager.CreateSandbox(ctx, "attacker")
    if err != nil {
        return nil, fmt.Errorf("failed to create sandbox: %v", err)
    }
    defer o.sandboxManager.DestroySandbox(ctx, sandbox.ID)
    
    // 2. 审计日志
    o.auditLogger.LogEvent(sandbox.AuditEvent{
        EventType: "scan_start",
        SandboxID: sandbox.ID,
        Metadata:  map[string]interface{}{"target": target},
    })
    
    // 3. 执行扫描
    result, err := o.executeInSandbox(ctx, sandbox.ID, target)
    
    // 4. 记录结果
    o.auditLogger.LogEvent(sandbox.AuditEvent{
        EventType: "scan_complete",
        SandboxID: sandbox.ID,
        Result:    "success",
    })
    
    return result, nil
}
```

---

## 9. 参考资料

- [Docker Security](https://docs.docker.com/engine/security/)
- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [cgroups v2](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html)
- [Seccomp](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html)
