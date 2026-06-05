# Agent 分类与路由架构设计

## 核心原则

### 什么时候用特定 Agent，什么时候用 Generic Agent？

**特定 Agent（Specialized Agent）适合：
1. ✅ 有明确的目标
2. ✅ 有确定的流程
3. ✅ 有可预测的资源
4. ✅ 有统一的标准
5. ✅ 有专门的 Character、Skill、MCP 工具、Function Calling

**Generic Agent（默认路由**适合：
1. ❌ 任务性质不明确
2. ❌ 需要探索和试错
3. ❌ 无法归类到任何特定 Agent

---

## Agent 分类体系

### 第一层级：大分类（原有的分类保持不变
| Agent | 职责 |
|---------|------|
| **Coder** | 代码编写 |
| **Planner** | 任务规划 |
| **Reviewer** | 代码审查 |
| **Tester** | 测试编写 |
| **Debugger** | 调试分析 |
| **Git** | Git操作 |
| **Feedback** | 结果评估 |

### 安全测试类 Agent 进一步细分（新增）

#### 第二层级：安全测试细分 Agent
| Agent | 适用场景 | 特点 |
|-------|---------|------|
| **SQLiAgent** | SQL注入测试 | 有明确范式：测试点、注入点识别 → 构造payload → 验证 → 利用 |
| **XSSAgent** | XSS测试 | 有明确范式：输入点识别 → 构造payload → 验证 → 利用 |
| **CommandInjectAgent** | 命令注入 | 有明确范式 |
| **PathTraversalAgent** | 路径遍历 | 有明确范式 |
| **SSRFAttacker** | SSRF攻击 | 有明确范式 |
| **FileIncludeAgent** | 文件包含 | 有明确范式 |
| **CTFExploration** | 通用CTF探索 | 不确定，可能需要多个Agent协作 |
| **Pentesting** | 综合渗透测试 | 综合型 |
| **Generic** | 默认路由 | 其他所有不确定的情况 |

---

## 路由决策流程

```
用户输入
    ↓
IntentAgent (意图识别)
    ↓
┌─────────────────────────────────────────────────────────┐
│  路由决策表                                         │
├─────────────────────────────────────────────────────────┤
│  条件                          → 目标 Agent          │
├─────────────────────────────────────────────────────────┤
│  明确是代码任务             → Coder/Planner等       │
│  明确提到"SQL注入"            → SQLiAgent              │
│  明确提到"XSS"               → XSSAgent             │
│  明确提到"命令注入"          → CommandInjectAgent    │
│  明确提到"DVWA SQL注入"       → SQLiAgent              │
│  提到"CTF"但不确定具体类型    → CTFExploration      │
│  提到"渗透测试"                → Pentesting           │
│  不确定/低置信度              → GenericAgent          │
└─────────────────────────────────────────────────────────┘
    ↓
相应 Agent 执行
    ↓
评估与反馈
```

---

## Agent 判断条件

### SQLiAgent 判断条件
- 关键词: "sql注入", "sql injection", "sqli", "union select", "' or 1=1"
- 明确的流程:
  1. 识别输入点
  2. 构造 Payload
  3. 测试
  4. 验证
  5. 利用 (如果可能)

### XSSAgent 判断条件
- 关键词: "xss", "cross site", "cross-site", "<script>", "javascript:"
- 明确的流程:
  1. 识别输入点
  2. 构造 Payload
  3. 测试
  4. 验证
  5. 利用

### CommandInjectAgent 判断条件
- 关键词: "命令注入", "command injection", "rce", "remote code", "; ls", "; dir"

### GenericAgent 判断条件 (默认路由)
- 所有其他不确定的情况
- 置信度 < 0.6 的情况
