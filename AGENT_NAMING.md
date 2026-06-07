# Agent 命名规则

## 1. 命名原则

### 1.1 专业术语优先
- 使用行业标准术语（如 OWASP、IEEE、ACM 等）
- 避免口语化、模糊的名称

### 1.2 功能明确
- 名称应清晰表达 Agent 的职责
- 使用动词+名词或名词+动词的组合

### 1.3 分类一致性
- 同类 Agent 使用相同的命名模式
- 例如：安全测试类统一使用 `Scanner` 后缀

---

## 2. 命名对照表

### 2.1 核心 Agent（控制/管理层）

| 旧名称 | 新名称 | 说明 |
|--------|--------|------|
| **Watcher** | **Observer** | 监控执行状态（Observer 模式术语） |
| **Auditor** | **Evaluator** | 评估执行结果（评估术语） |
| **Scheduler** | **Scheduler** | 保持不变（调度器是标准术语） |

### 2.2 规划/管理层

| 旧名称 | 新名称 | 说明 |
|--------|--------|------|
| **Intent** | **IntentAnalyzer** | 意图分析（明确功能） |
| **Planner** | **TaskPlanner** | 任务规划（明确对象） |
| **Orchestrator** | **TaskCoordinator** | 任务协调（Coordinator 模式术语） |
| **Feedback** | **FeedbackCollector** | 反馈收集（明确功能） |
| **Exploration** | **SolutionExplorer** | 解决方案探索（明确目标） |

### 2.3 编码类 Agent

| 旧名称 | 新名称 | 说明 |
|--------|--------|------|
| **Coder** | **CodeGenerator** | 代码生成（明确功能） |
| **Reviewer** | **CodeReviewer** | 代码审查（明确对象） |
| **Tester** | **TestEngineer** | 测试工程师（行业术语） |
| **Debugger** | **DebugAnalyst** | 调试分析（明确功能） |
| **Git** | **GitOperator** | Git操作（明确功能） |

### 2.4 安全测试类 Agent（OWASP 对应）

| 旧名称 | 新名称 | OWASP 分类 | 说明 |
|--------|--------|-----------|------|
| **Pentesting** | **SecurityAssessor** | - | 安全评估（总体） |
| **SQLiAgent** | **SQLInjectionScanner** | A03:Injection | SQL注入扫描 |
| **XSSAgent** | **XSSScanner** | A03:Injection | XSS扫描 |
| **CommandInjectAgent** | **CommandInjectionScanner** | A03:Injection | 命令注入扫描 |
| **PathTraversalAgent** | **PathTraversalScanner** | A01:Broken Access Control | 路径遍历扫描 |
| **SSRFAgent** | **SSRFScanner** | A10:SSRF | SSRF扫描 |
| **FileIncludeAgent** | **FileIncludeScanner** | A03:Injection | 文件包含扫描 |
| **CTFExploration** | **CTFSolver** | - | CTF解决 |

### 2.5 通用类

| 旧名称 | 新名称 | 说明 |
|--------|--------|------|
| **Generic Agent** | **GeneralHandler** | 通用处理（明确功能） |

---

## 3. 文件命名规则

### 3.1 Go 文件
- 格式：`<新名称小写>_agent.go`
- 例如：`observer_agent.go`、`evaluator_agent.go`

### 3.2 SKILL.md 文件
- 格式：`<新名称大写>.SKILL.md`
- 例如：`OBSERVER.SKILL.md`、`EVALUATOR.SKILL.md`

---

## 4. OWASP Top 10 对应（2021）

| OWASP 分类 | 对应 Agent |
|-----------|-----------|
| A01: Broken Access Control | PathTraversalScanner |
| A02: Cryptographic Failures | （待实现） |
| A03: Injection | SQLInjectionScanner, XSSScanner, CommandInjectionScanner, FileIncludeScanner |
| A04: Insecure Design | （待实现） |
| A05: Security Misconfiguration | （待实现） |
| A06: Vulnerable Components | （待实现） |
| A07: Auth Failures | （待实现） |
| A08: Software Integrity | （待实现） |
| A09: Logging Failures | （待实现） |
| A10: SSRF | SSRFScanner |

---

## 5. 实施步骤

1. 重命名 Go 文件
2. 更新 Agent 结构体名称
3. 更新构造函数名称
4. 更新 SKILL.md 文件名和内容
5. 更新 main.go 中的注册
6. 更新文档（DESIGN.md、README.md）
7. 构建测试
8. 提交 Git