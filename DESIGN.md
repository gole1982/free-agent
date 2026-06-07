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
| Executor/Evaluator/Observer 启动/停止 | **硬编码** | 生命周期管理 |
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
│  委托 TaskCoordinator 进行业务级路由                                │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│              TaskCoordinator（任务协调 Agent）                       │
│  [Agent] 意图分析、任务分解、多Agent协调、结果汇总                   │
│  IntentAnalyzer 解析意图 → 选择 Executor                            │
└─────────────────────────────────────────────────────────────────────┘
                                ↓
        ┌────────────────────────┼────────────────────────┐
        ↓                        ↓                        ↓
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Executor       │    │  Observer       │    │  Evaluator      │
│  (执行Agent)    │    │  (控制Agent)    │    │  (管理Agent)    │
│                 │    │                 │    │                 │
│ - IntentAnalyzer│    │ - 意图监控      │    │ - 结果分析      │
│ - TaskPlanner   │    │ - 死循环检测    │    │ - 策略更新      │
│ - CodeGenerator │    │ - 蜜罐检测      │    │ - 特性调整      │
│ - CodeReviewer  │    │ - 恶意检测      │    │ - 结论生成      │
│ - TestEngineer  │    │ - 主动停止      │    │                 │
│ - DebugAnalyst  │    │ - 汇总信息      │    │                 │
│ - GitOperator   │    │                 │    │                 │
│ - SecurityAssess│    │                 │    │                 │
│ - SQLInjection  │    │                 │    │                 │
│ - XSSScanner    │    │                 │    │                 │
│ - ...Scanners   │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### 2.2 执行流程

```
1. 用户输入
   ↓
2. Scheduler（调度器）接收
   ↓
3. 硬编码：委托 TaskCoordinator/IntentAnalyzer 解析意图
   ↓
4. 硬编码：通过 TaskCoordinator 选择 Executor
   ↓
5. 启动 Observer（并行监控）
   ↓
6. Executor 执行任务
   ↓
7. Executor 输出 → Observer 接收
   ↓
8. Observer 判断：
   - 是否遵循意图？
   - 是否死循环？
   - 是否恶意？
   - 是否蜜罐？
   ↓
9. 如果需要：Observer 停止 Executor，给出指引
   ↓
10. Executor 完成/退出
    ↓
11. Observer 汇总执行信息
    ↓
12. Evaluator 分析：
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
- Executor/Observer/Evaluator 的启动和停止
- 通道管理（Executor→Observer→Evaluator 的信息流转）
- 固定串行计算（Intent 解析、Agent 选择）

**核心方法**：
```go
func NewScheduler(llmClient *llm.Client, am *AgentManager, skillLoader *SkillLoader) *Scheduler
func (s *Scheduler) ExecuteWithAgentPattern(ctx context.Context, task string) (string, error)
func (s *Scheduler) selectExecutorViaCoordinator(ctx context.Context, task string) (*IntentResult, string, bool)
func (s *Scheduler) executeMultiAgentWithObserver(ctx context.Context, task string, intentInfo *IntentResult) (string, error)
func (s *Scheduler) executeSingleWithObserver(ctx context.Context, task string, executorAgentName string) (string, error)
func (s *Scheduler) SetMaxIterations(max int)
func (s *Scheduler) SetMaxDuration(duration time.Duration)
```

### 3.2 Observer Agent（控制Agent）

**文件**：[observer_agent.go](file:///d:/Programing/free-agent/internal/agent/observer_agent.go)

**SKILL 配置**：[observer.SKILL.md](file:///d:/Programing/free-agent/skills/control/observer.SKILL.md)

**职责**：
- 实时接收 Executor 执行信息
- 使用 LLM 判断 Executor 是否遵循用户原始意图
- 检测死循环（重复输出、无进展）
- 检测恶意行为（危险操作）
- 检测蜜罐触发（“被抓”、“caught”等）
- 必要时停止 Executor 并提供正确指引
- Executor 退出后汇总所有执行信息

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

### 3.3 Evaluator Agent（管理Agent）

**文件**：[evaluator_agent.go](file:///d:/Programing/free-agent/internal/agent/evaluator_agent.go)

**SKILL 配置**：[evaluator.SKILL.md](file:///d:/Programing/free-agent/skills/management/evaluator.SKILL.md)

**职责**：
- 分析 Observer 的执行汇总信息
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

### 3.4 TaskCoordinator Agent（任务协调器）

**文件**：[task_coordinator_agent.go](file:///d:/Programing/free-agent/internal/agent/task_coordinator_agent.go)

**SKILL 配置**：[taskcoordinator.SKILL.md](file:///d:/Programing/free-agent/skills/management/taskcoordinator.SKILL.md)

**职责**：
- 理解复杂任务需求
- 通过 IntentAnalyzer 分析意图
- 分解任务为子任务
- 选择合适的 Executor 执行每个子任务
- 协调多个 Executor 的工作
- 合并结果为最终输出

### 3.5 Executor Agents（执行Agent）

Executor 包括所有业务执行 Agent，如：

| Agent | 职责 | 文件 |
|-------|------|------|
| IntentAnalyzer | 意图识别 | [intent_analyzer_agent.go](file:///d:/Programing/free-agent/internal/agent/intent_analyzer_agent.go) |
| TaskPlanner | 任务规划 | [task_planner_agent.go](file:///d:/Programing/free-agent/internal/agent/task_planner_agent.go) |
| CodeGenerator | 代码编写 | [code_generator_agent.go](file:///d:/Programing/free-agent/internal/agent/code_generator_agent.go) |
| CodeReviewer | 代码审查 | [code_reviewer_agent.go](file:///d:/Programing/free-agent/internal/agent/code_reviewer_agent.go) |
| TestEngineer | 测试生成 | [test_engineer_agent.go](file:///d:/Programing/free-agent/internal/agent/test_engineer_agent.go) |
| DebugAnalyst | 调试 | [debug_analyst_agent.go](file:///d:/Programing/free-agent/internal/agent/debug_analyst_agent.go) |
| GitOperator | 版本控制 | [git_operator_agent.go](file:///d:/Programing/free-agent/internal/agent/git_operator_agent.go) |
| SecurityAssessor | 安全评估 | [security_assessor_agent.go](file:///d:/Programing/free-agent/internal/agent/security_assessor_agent.go) |
| SQLInjectionScanner | SQL注入 | [sql_injection_scanner.go](file:///d:/Programing/free-agent/internal/agent/sql_injection_scanner.go) |
| XSSScanner | XSS测试 | [xss_scanner.go](file:///d:/Programing/free-agent/internal/agent/xss_scanner.go) |
| GeneralHandler | 通用任务 | [general_handler_agent.go](file:///d:/Programing/free-agent/internal/agent/general_handler_agent.go) |

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
│   ├── codegenerator.SKILL.md
│   ├── codereviewer.SKILL.md
│   ├── testengineer.SKILL.md
│   └── debuganalyst.SKILL.md
├── control/
│   └── observer.SKILL.md
├── general/
│   └── generalhandler.SKILL.md
├── management/
│   ├── evaluator.SKILL.md
│   ├── feedbackcollector.SKILL.md
│   ├── intentanalyzer.SKILL.md
│   └── taskcoordinator.SKILL.md
├── planning/
│   ├── solutionexplorer.SKILL.md
│   └── taskplanner.SKILL.md
├── security/
│   ├── securityassessor.SKILL.md
│   ├── sqlinjectionscanner.SKILL.md
│   ├── xssscanner.SKILL.md
│   ├── commandinjectionscanner.SKILL.md
│   ├── pathtraversalscanner.SKILL.md
│   ├── ssrfscanner.SKILL.md
│   ├── fileincludescanner.SKILL.md
│   ├── ctfsolver.SKILL.md
│   └── ...其他 OWASP 扫描器
└── tools/
    └── gitoperator.SKILL.md
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
2. **IntentAnalyzer** 识别：安全测试 → 选择 `SecurityAssessor` Executor
3. 启动 `Observer` 监控
4. **SecurityAssessor** 执行：
   - 调用 `SQLInjectionScanner`
   - 调用 `XSSScanner`
   - 调用 `PathTraversalScanner`
   - ...
5. **Observer** 监控每个 Executor：
   - 如果发现蜜罐，立即停止
   - 如果发现死循环，立即停止
   - 如果偏离目标，给出指引
6. 所有 Executor 完成后，`Observer` 汇总
7. **Evaluator** 分析：
   - 检查是否有安全策略需要更新
   - 调整相关 Agent 特性
8. 输出最终报告

---

## 5. 历史组件（已清理）

旧版 `ClosedLoopManager` (ExecutionLayer / ControlLayer / ManagementLayer 三层) 已被 `Scheduler + Executor/Observer/Evaluator` 完全替代并删除：

| 旧组件 | 新组件 |
|--------|--------|
| ExecutionLayer | Executor Agents |
| ControlLayer | Observer Agent |
| ManagementLayer | Evaluator Agent |
| ClosedLoopManager | Scheduler |
| Orchestrator Agent | TaskCoordinator Agent |

未挂载的子系统和已整合的组件已清理。新增 VDS（漏洞挖掘系统）框架、沙箱管理器、工具适配器层。

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
| `feat: implement Executor/Observer/Evaluator pattern` | 新架构核心实现 |
| `feat: complete migration to Scheduler pattern` | 完成架构迁移 |
| `feat: integrate TaskCoordinator routing` | Scheduler 委托 TaskCoordinator 统一路由 |
| `feat: add VDS framework` | 漏洞挖掘系统 6 阶段框架 |
| `feat: sandbox manager with policy engine` | 沙箱管理器 + 策略引擎 + 快照回滚 |
| `feat: tool adapter layer` | sqlmap/ZAP/Nmap 工具适配器 |
