# Agent 命名规则

> **状态**：✅ 已完成（2026-06）

## 1. 命名原则

- **专业术语优先**：行业标准术语（OWASP、IEEE、ACM）
- **功能明确**：动词+名词 / 名词+动词
- **分类一致**：同类用相同后缀（如安全类统一用 `Scanner`）

## 2. 命名对照表（已落地）

### 2.1 核心 Agent（控制/管理层）

| 旧名 | 新名 | 说明 |
|------|------|------|
| Watcher | **Observer** | Observer 模式术语 |
| Auditor | **Evaluator** | 评估术语 |
| Scheduler | **Scheduler** | 不变 |

### 2.2 规划/管理层

| 旧名 | 新名 |
|------|------|
| Intent | **IntentAnalyzer** |
| Planner | **TaskPlanner** |
| Orchestrator | **TaskCoordinator** |
| Feedback | **FeedbackCollector** |
| Explorer | **SolutionExplorer** |

### 2.3 编码类 Agent

| 旧名 | 新名 |
|------|------|
| Coder | **CodeGenerator** |
| Reviewer | **CodeReviewer** |
| Tester | **TestEngineer** |
| Debugger | **DebugAnalyst** |
| Git | **GitOperator** |

### 2.4 安全测试类 Agent

| 旧名 | 新名 | OWASP |
|------|------|-------|
| Pentesting | **SecurityAssessor** | - |
| SQLiAgent | **SQLInjectionScanner** | A03 |
| XSSAgent | **XSSScanner** | A03 |
| CommandInjectAgent | **CommandInjectionScanner** | A03 |
| PathTraversalAgent | **PathTraversalScanner** | A01 |
| SSRFAgent | **SSRFScanner** | A10 |
| FileIncludeAgent | **FileIncludeScanner** | A03 |
| CTFExploration | **CTFSolver** | - |

### 2.5 通用类

| 旧名 | 新名 |
|------|------|
| Generic Agent | **GeneralHandler** |

## 3. 文件命名规则

- Go 文件：`<新名小写>_agent.go`（如 `code_reviewer_agent.go`）
- SKILL.md：`<新名小写>.SKILL.md`（如 `codereviewer.SKILL.md`）
- 目录分类：保持 `coding/control/general/management/planning/security/tools`

## 4. OWASP Top 10 对应（2021）

| OWASP | Agent |
|-------|-------|
| A01: Broken Access Control | PathTraversalScanner |
| A03: Injection | SQLInjectionScanner, XSSScanner, CommandInjectionScanner, FileIncludeScanner |
| A10: SSRF | SSRFScanner |
| 其他 | 待实现 |