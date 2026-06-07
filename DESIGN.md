# Free Agent 系统需求与设计文档

## 1. 用户需求概述

### 1.1 核心需求

Free Agent 是一个多AI代理协作的软件工程系统，具有以下核心特性：

1. **多代理协作架构**：多个专业化 Agent 协作完成复杂任务
2. **三层闭环自学习**：执行层、控制层、管理层协同工作，持续优化
3. **技能驱动配置**：Agent 行为通过 SKILL.md 文件定义和配置
4. **安全测试能力**：支持渗透测试、SQL注入、XSS等安全评估
5. **迭代执行机制**：任务执行过程支持循环迭代，直到完成或无法继续
6. **意图识别与路由**：自动识别用户意图，路由到合适的 Agent

### 1.2 关键架构原则

**最重要的原则：硬编码与 Agent 边界清晰划分**

| 系统功能 | 实现方式 | 原因 |
|----------|----------|------|
| 计时器、超时管理 | **硬编码** | 系统级基础设施 |
| 定时任务调度 | **硬编码** | 固定逻辑，无需智能 |
| 接收返回状态 | **硬编码** | 系统级机制 |
| 固定串行计算（Intent解析、Agent选择） | **硬编码** | 确定性流程 |
| Worker/Watcher/Auditor 启动/停止 | **硬编码** | 生命周期管理 |
| 通道管理（信息流转） | **硬编码** | 基础设施 |
| **意图判断** | **Agent** | 需要 LLM 智能判断 |
| **死循环检测** | **Agent** | 需要理解上下文 |
| **蜜罐检测** | **Agent** | 需要智能识别 |
| **恶意行为检测** | **Agent** | 需要安全判断能力 |
| **结果评审** | **Agent** | 需要质量评估能力 |
| **安全策略更新** | **Agent** | 需要学习和适应 |
| **Agent特性调整** | **Agent** | 需要优化能力 |
| **业务执行** | **Agent** | 需要领域智能 |

---

## 2. 系统架构设计

### 2.1 最终架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Scheduler（调度器）                             │
│  [硬编码] 系统调度、计时器、超时管理、生命周期管理、通道管理         │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│                  Orchestrator（编排器 Agent）                       │
│  [Agent] 业务级任务编排、任务分解、多Agent协调、结果汇总             │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
        ┌────────────────────────┼────────────────────────┐
        ↓                        ↓                        ↓
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Worker         │    │  Watcher        │    │  Auditor        │
│  (执行Agent)    │    │  (控制Agent)    │    │  (管理Agent)    │
│                 │    │                 │    │                 │
│ - Intent        │    │ - 意图监控      │    │ - 结果分析      │
│ - Planner       │    │ - 死循环检测    │    │ - 策略更新      │
│ - Coder         │    │ - 蜜罐检测      │    │ - 特性调整      │
│ - Reviewer      │    │ - 恶意检测      │    │ - 结论生成      │
│ - Tester        │    │ - 主动停止      │    │                 │
│ - Debugger      │    │ - 汇总信息      │    │                 │
│ - Git           │    │                 │    │                 │
│ - Pentesting    │    │                 │    │                 │
│ - SQLiAgent     │    │                 │    │                 │
│ - XSSAgent      │    │                 │    │                 │
│ - ...           │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2.2 执行流程

```
1. 用户输入
   ↓
2. Scheduler（调度器）接收
   ↓
3. 硬编码：解析 Intent
   ↓
4. 硬编码：选择 Worker
   ↓
5. 启动 Watcher（并行监控）
   ↓
6. Worker 执行任务
   ↓
7. Worker 输出 → Watcher 接收
   ↓
8. Watcher 判断：
   - 是否遵循意图？
   - 是否死循环？
   - 是否恶意？
   - 是否蜜罐？
   ↓
9. 如果需要：Watcher 停止 Worker，给出指引
   ↓
10. Worker 完成/退出
    ↓
11. Watcher 汇总执行信息
    ↓
12. Auditor 分析：
    - 退出类型？
    - 任务完成？
    - 需要更新策略？
    - 需要调整 Agent 特性？
    ↓
13. 输出最终结果
```

---

## 3. 各组件详细设计

### 3.1 Scheduler（调度器）- 硬编码

**文件**：[scheduler.go](file:///d:/Programing/free-agent/internal/agent/scheduler.go)

**职责**：
- 计时器和超时管理
- Worker/Watcher/Auditor 的启动和停止
- 通道管理（Worker→Watcher→Auditor 的信息流转）
- 固定串行计算（Intent 解析、Agent 选择）

**核心方法**：
```go
func NewScheduler(llmClient *llm.Client, am *AgentManager, skillLoader *SkillLoader) *Scheduler
func (s *Scheduler) ExecuteWithAgentPattern(ctx context.Context, task string) (string, error)
func (s *Scheduler) SetMaxIterations(max int)
func (s *Scheduler) SetMaxDuration(duration time.Duration)
func (s *Scheduler) SetWorkerTimeout(timeout time.Duration)
func (s *Scheduler) SetWatcherInterval(interval time.Duration)
```

### 3.2 Watcher Agent（控制Agent）

**文件**：[watcher_agent.go](file:///d:/Programing/free-agent/internal/agent/watcher_agent.go)

**SKILL 配置**：[watcher.SKILL.md](file:///d:/Programing/free-agent/skills/control/watcher.SKILL.md)

**职责**：
- 实时接收 Worker 执行信息
- 使用 LLM 判断 Worker 是否遵循用户原始意图
- 检测死循环（重复输出、无进展）
- 检测恶意行为（危险操作）
- 检测蜜罐触发（"被抓"、"caught"等）
- 必要时停止 Worker 并提供正确指引
- Worker 退出后汇总所有执行信息

**核心数据结构**：
```go
type WorkerExecutionInfo struct {
	AgentName          string
	Task               string
	Output             string
	ExecutionFlag      string
	Timestamp          time.Time
	IntentMatch        float64
	SuspiciousIndicators []string
}

type WatcherDecision struct {
	ShouldStop       bool
	Reason           string
	CorrectGuidance  string
	IntentAlignment  float64
	IsDeadLoop       bool
	IsMalicious      bool
	IsHoneypot       bool
}
```

### 3.3 Auditor Agent（管理Agent）

**文件**：[auditor_agent.go](file:///d:/Programing/free-agent/internal/agent/auditor_agent.go)

**SKILL 配置**：[auditor.SKILL.md](file:///d:/Programing/free-agent/skills/management/auditor.SKILL.md)

**职责**：
- 分析 Watcher 的执行汇总信息
- 确定退出类型（正常/异常/恶意/死循环）
- 决定是否需要更新安全策略
- 更新输入过滤（恶意指令检测）
- 更新安全技能（蜜罐识别）
- 调整 Agent 特性
- 决定任务是否完成
- 提供重试指引

**核心数据结构**：
```go
type AuditConclusion struct {
	ExitType          string
	NeedsUpdate       bool
	UpdateType        string
	UpdateContent     string
	TaskCompleted     bool
	RetryNeeded       bool
	RetryGuidance     string
	AgentAdjustments  []*AgentTraitsAdjustment
}
```

### 3.4 Orchestrator Agent（编排器）

**文件**：[orchestrator_agent.go](file:///d:/Programing/free-agent/internal/agent/orchestrator_agent.go)

**职责**：
- 理解复杂任务需求
- 分解任务为子任务
- 选择合适的 Agent 执行每个子任务
- 协调多个 Agent 的工作
- 合并结果为最终输出

### 3.5 Worker Agents（执行Agent）

Worker 包括所有业务执行 Agent，如：

| Agent | 职责 | 文件 |
|-------|------|------|
| Intent | 意图识别 | [intent_agent.go](file:///d:/Programing/free-agent/internal/agent/intent_agent.go) |
| Planner | 任务规划 | [planner_agent.go](file:///d:/Programing/free-agent/internal/agent/planner_agent.go) |
| Coder | 代码编写 | [coder_agent.go](file:///d:/Programing/free-agent/internal/agent/coder_agent.go) |
| Reviewer | 代码审查 | [reviewer_agent.go](file:///d:/Programing/free-agent/internal/agent/reviewer_agent.go) |
| Tester | 测试生成 | [tester_agent.go](file:///d:/Programing/free-agent/internal/agent/tester_agent.go) |
| Debugger | 调试 | [debugger_agent.go](file:///d:/Programing/free-agent/internal/agent/debugger_agent.go) |
| Git | 版本控制 | [git_agent.go](file:///d:/Programing/free-agent/internal/agent/git_agent.go) |
| Pentesting | 渗透测试 | [pentesting_agent.go](file:///d:/Programing/free-agent/internal/agent/pentesting_agent.go) |
| SQLiAgent | SQL注入 | [sqli_agent.go](file:///d:/Programing/free-agent/internal/agent/sqli_agent.go) |
| XSSAgent | XSS测试 | [xss_agent.go](file:///d:/Programing/free-agent/internal/agent/xss_agent.go) |
| CommandInjectAgent | 命令注入 | [other_security_agents.go](file:///d:/Programing/free-agent/internal/agent/other_security_agents.go) |
| PathTraversalAgent | 路径遍历 | [other_security_agents.go](file:///d:/Programing/free-agent/internal/agent/other_security_agents.go) |
| SSRFAgent | SSRF测试 | [other_security_agents.go](file:///d:/Programing/free-agent/internal/agent/other_security_agents.go) |
| FileIncludeAgent | 文件包含 | [other_security_agents.go](file:///d:/Programing/free-agent/internal/agent/other_security_agents.go) |
| CTFExploration | CTF探索 | [other_security_agents.go](file:///d:/Programing/free-agent/internal/agent/other_security_agents.go) |
| Generic Agent | 通用任务 | [generic_agent.go](file:///d:/Programing/free-agent/internal/agent/generic_agent.go) |

### 3.6 SkillLoader（技能加载器）

**文件**：[skill_loader.go](file:///d:/Programing/free-agent/internal/agent/skill_loader.go)

**职责**：
- 从 `skills/` 目录加载 SKILL.md 文件
- 解析 Agent 角色、能力、工作流、质量指标
- 支持保存更新后的 Agent 特性

**目录结构**：
```
skills/
├── coding/
│   ├── coder.SKILL.md
│   ├── reviewer.SKILL.md
│   ├── tester.SKILL.md
│   └── debugger.SKILL.md
├── control/
│   └── watcher.SKILL.md
├── general/
│   └── generic.SKILL.md
├── management/
│   ├── auditor.SKILL.md
│   ├── feedback.SKILL.md
│   ├── intent.SKILL.md
│   └── orchestrator.SKILL.md
├── planning/
│   ├── exploration.SKILL.md
│   └── planner.SKILL.md
├── security/
│   ├── commandinject.SKILL.md
│   ├── ctfestoration.SKILL.md
│   ├── fileinclude.SKILL.md
│   ├── pathtraversal.SKILL.md
│   ├── pentesting.SKILL.md
│   ├── sqli.SKILL.md
│   ├── ssrf.SKILL.md
│   └── xss.SKILL.md
└── tools/
    └── git.SKILL.md
```

---

## 4. 系统使用示例

### 4.1 渗透测试场景

**用户输入**：
```
完成对 http://example.com 的渗透测试
```

**执行流程**：

1. **Scheduler** 接收任务
2. **Intent** 识别：安全测试 → 选择 `Pentesting` Agent
3. 启动 `Watcher` 监控
4. **Pentesting** Agent 执行：
   - 调用 `SQLiAgent`
   - 调用 `XSSAgent`
   - 调用 `PathTraversalAgent`
   - ...
5. **Watcher** 监控每个 Agent：
   - 如果发现蜜罐，立即停止
   - 如果发现死循环，立即停止
   - 如果偏离目标，给出指引
6. 所有 Agent 完成后，`Watcher` 汇总
7. **Auditor** 分析：
   - 检查是否有安全策略需要更新
   - 调整相关 Agent 特性
8. 输出最终报告

---

## 5. 遗留组件（待清理）

**ClosedLoopManager**（[closed_loop_manager.go](file:///d:/Programing/free-agent/internal/agent/closed_loop_manager.go)）是旧版三层闭环架构的实现，已被新架构替代：

| 旧组件 | 新组件 |
|--------|--------|
| ExecutionLayer | Worker Agents |
| ControlLayer | Watcher Agent |
| ManagementLayer | Auditor Agent |
| ClosedLoopManager | Scheduler |

在确认新架构稳定后，可以移除 ClosedLoopManager。

---

## 6. 技术栈

- **语言**：Go
- **LLM API**：可配置的 API 端点
- **技能配置**：Markdown（SKILL.md）
- **项目结构**：模块化设计

---

## 7. 提交历史

| 提交 | 说明 |
|------|------|
| `feat: implement Worker/Watcher/Auditor pattern` | 新架构核心实现 |
| `feat: complete migration to Scheduler pattern` | 完成架构迁移 |
