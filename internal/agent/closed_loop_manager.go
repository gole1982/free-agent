package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// LoopPhase 定义闭环执行的阶段
type LoopPhase string

const (
	// 执行层 - 负责业务执行 + 内容安全
	PhasePlan    LoopPhase = "Plan"    // 规划阶段
	PhaseExecute LoopPhase = "Execute" // 执行阶段
	PhaseCheck   LoopPhase = "Check"   // 执行层内部安全检查
	
	// 控制层 - 只监督执行状态和流程
	PhaseMonitor  LoopPhase = "Monitor" // 状态监控
	
	// 管理层 - 审阅数据、处理 agent 特性
	PhaseReview   LoopPhase = "Review"   // 评审阶段
	PhaseImprove  LoopPhase = "Improve"  // 改进阶段
	PhaseComplete LoopPhase = "Complete" // 完成阶段
)

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusRunning   ExecutionStatus = "running"
	StatusPaused    ExecutionStatus = "paused"
	StatusError     ExecutionStatus = "error"
	StatusCompleted ExecutionStatus = "completed"
	StatusTerminated ExecutionStatus = "terminated"
)

// AgentTraits 定义 Agent 的可量化特性（用于自学习和考核）
type AgentTraits struct {
	Name          string  // Agent 名称
	Efficiency    float64 // 效率 (0-1) - 执行速度
	Quality       float64 // 质量 (0-1) - 输出质量
	Creativity    float64 // 创造性 (0-1)
	Collaboration float64 // 协作性 (0-1)
	LearningRate  float64 // 学习率 (0-1)
	UsageCount    int     // 使用次数
	SuccessCount  int     // 成功次数
}

// ExecutionContext 执行层上下文（控制层只获取必要的监督信息）
type ExecutionContext struct {
	TaskID        string
	OriginalTask  string
	CurrentPlan   string
	CurrentStep   int
	TotalSteps    int
	CurrentAgent  string
	OutputHistory []string
	CurrentOutput string
	IssuesFound   []string // 执行过程中发现的问题列表
	StartTime     time.Time
	LastUpdate    time.Time
	IsTerminated  bool
	TermReason    string
	Status        ExecutionStatus
	CurrentPhase  LoopPhase
	ExecutionFlag string // 执行层返回的标记：normal/warning/error/honeypot/quit
}

// ExecutionMetrics 执行指标（控制层收集的考核数据）
type ExecutionMetrics struct {
	AgentName     string
	Phase         LoopPhase
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Success       bool
	Error         string
	ExecutionFlag string // 执行层返回的标记
}

// MonitorResult 监控结果（控制层只关注状态，不深入内容）
type MonitorResult struct {
	IsHealthy     bool
	NeedsAction   bool
	ActionType    string // "terminate_agent" / "terminate_all" / "security_review" / "notify_management" / "continue"
	Issues        []string
	Confidence    float64
}

// ReviewResult 评审阶段结果（管理层输出）
type ReviewResult struct {
	OverallScore     float64
	Strengths        []string
	Weaknesses       []string
	Improvements     []string
	ShouldRetry      bool
	NextIterationHint string // 下次迭代的改进建议
	AgentAdjustment  *AgentTraitsAdjustment // Agent 特性调整建议
}

// AgentTraitsAdjustment Agent 特性调整
type AgentTraitsAdjustment struct {
	AgentName         string
	EfficiencyDelta   float64
	QualityDelta      float64
	CreativityDelta   float64
}

// ExecutionLayer 执行层 - 负责完整的业务执行和内容安全
type ExecutionLayer struct {
	llmClient *llm.Client
	agentMgr  *AgentManager
}

// ControlLayer 控制层 - 只监督状态，不深入业务内容
type ControlLayer struct {
	llmClient          *llm.Client
	agentMgr           *AgentManager
	metricsHistory     []ExecutionMetrics
	executionContext   *ExecutionContext // 只获取必要的监督信息
}

// ManagementLayer 管理层 - 审阅数据、处理 agent 特性
type ManagementLayer struct {
	llmClient        *llm.Client
	agentMgr         *AgentManager
	agentTraits      map[string]*AgentTraits
	controlLayer     *ControlLayer
	skillLoader      *SkillLoader
}

// ClosedLoopManager 三层闭环管理器
type ClosedLoopManager struct {
	executionLayer  *ExecutionLayer
	controlLayer    *ControlLayer
	managementLayer *ManagementLayer
	maxIterations   int
	maxDuration     time.Duration // 最大执行时间
}

// NewClosedLoopManager 创建新的三层闭环管理器
func NewClosedLoopManager(llmClient *llm.Client, agentMgr *AgentManager, skillLoader *SkillLoader) *ClosedLoopManager {
	controlLayer := &ControlLayer{
		llmClient:      llmClient,
		agentMgr:       agentMgr,
		metricsHistory: make([]ExecutionMetrics, 0),
	}

	managementLayer := &ManagementLayer{
		llmClient:    llmClient,
		agentMgr:     agentMgr,
		agentTraits:  make(map[string]*AgentTraits),
		controlLayer: controlLayer,
		skillLoader:  skillLoader,
	}

	return &ClosedLoopManager{
		executionLayer: &ExecutionLayer{
			llmClient: llmClient,
			agentMgr:  agentMgr,
		},
		controlLayer:    controlLayer,
		managementLayer: managementLayer,
		maxIterations:   10,
		maxDuration:     10 * time.Minute, // 默认10分钟超时
	}
}

// SetMaxDuration 设置最大执行时间
func (clm *ClosedLoopManager) SetMaxDuration(duration time.Duration) {
	clm.maxDuration = duration
}

// RegisterAgentTraits 注册 Agent 特性
func (clm *ClosedLoopManager) RegisterAgentTraits(name string, traits *AgentTraits) {
	clm.managementLayer.agentTraits[name] = traits
}

// =============================================
// 执行层核心方法 - 业务执行 + 内容安全
// =============================================

// ExecuteAgent 执行层执行一个 Agent（包含内部安全检查）
func (el *ExecutionLayer) ExecuteAgent(ctx context.Context, agentName string, task string) (string, string, error) {
	// 1. 检查是否有 /quit 指令（执行层自己检查）
	if shouldExit, reason := checkExitCommand(task); shouldExit {
		return "", "quit", fmt.Errorf(reason)
	}
	
	// 2. 执行 Agent
	startTime := time.Now()
	agent, err := el.agentMgr.GetAgent(agentName)
	if err != nil {
		return "", "error", err
	}
	
	result, err := agent.Execute(ctx, task)
	
	// 3. 执行层内部安全检查（这是执行层的职责，不是控制层的）
	flag := el.performInternalSafetyCheck(result, err)
	
	duration := time.Since(startTime)
	fmt.Printf("📋 [执行层] %s 执行完成，耗时 %v，标记: %s\n", agentName, duration, flag)
	
	return result, flag, err
}

// performInternalSafetyCheck 执行层内部安全检查（执行层职责）
func (el *ExecutionLayer) performInternalSafetyCheck(result string, err error) string {
	// 执行层自己做内容安全检查
	if err != nil {
		return "error"
	}
	
	lowerResult := strings.ToLower(result)
	
	// 蜜罐检测（执行层自己检查）
	if strings.Contains(lowerResult, "你被抓了") || 
	   strings.Contains(lowerResult, "caught") || 
	   strings.Contains(lowerResult, "honeypot") {
		fmt.Println("⚠️ [执行层] 检测到蜜罐触发")
		return "honeypot"
	}
	
	// 错误检测
	if strings.Contains(lowerResult, "error") || 
	   strings.Contains(lowerResult, "failed") ||
	   strings.Contains(lowerResult, "exception") {
		return "warning"
	}
	
	// 正常
	return "normal"
}

// =============================================
// 控制层核心方法 - 只监督状态，不深入内容
// =============================================

// MonitorExecution 控制层监督执行（只关注状态，不看业务内容）
// 注意：这是一个简化的控制层实现，用于演示架构概念
// 真实生产环境的控制层需要更复杂的异常判断逻辑，包括：
// - 更复杂的蜜罐识别模式（行为分析、模式匹配等）
// - 多Agent输出的上下文关联分析
// - 历史数据学习和自适应阈值调整
// - 更精细的异常分级和处理策略
func (cl *ControlLayer) MonitorExecution(ctx context.Context, agentName string, duration time.Duration, flag string) *MonitorResult {
	issues := []string{}
	actionType := "continue"
	isHealthy := true
	
	// 1. 检查执行时间（超时检测）
	if duration > 5*time.Minute {
		issues = append(issues, "执行时间过长")
		isHealthy = false
		actionType = "terminate_agent" // 仅终止该Agent，继续其他测试
	}
	
	// 2. 检查执行层返回的标记（只看标记，不看内容）
	switch flag {
	case "error":
		issues = append(issues, "Agent返回错误标记")
		isHealthy = false
		actionType = "security_review" // 调度安全审查
	case "honeypot":
		issues = append(issues, "检测到蜜罐标记")
		isHealthy = false
		// 蜜罐只终止该类型的测试，不中断其他漏洞类型的测试
		actionType = "terminate_agent"
	case "quit":
		issues = append(issues, "用户请求退出")
		isHealthy = false
		actionType = "terminate_all" // /quit 才终止所有
	case "warning":
		issues = append(issues, "警告标记")
		// 继续执行，但记录问题
	}
	
	// 3. 检查执行状态（死循环、上下文取消等）
	if cl.executionContext != nil {
		elapsed := time.Since(cl.executionContext.StartTime)
		if elapsed > 10*time.Minute {
			issues = append(issues, "总执行时间超时")
			isHealthy = false
			actionType = "terminate_all"
		}
	}
	
	// 4. 死循环检测（检查重复输出）
	if cl.executionContext != nil && len(cl.executionContext.OutputHistory) >= 3 {
		last := cl.executionContext.OutputHistory[len(cl.executionContext.OutputHistory)-1]
		prev := cl.executionContext.OutputHistory[len(cl.executionContext.OutputHistory)-2]
		prevprev := cl.executionContext.OutputHistory[len(cl.executionContext.OutputHistory)-3]
		if last == prev && prev == prevprev && last != "" {
			issues = append(issues, "检测到重复输出，可能死循环")
			isHealthy = false
			actionType = "terminate_agent"
		}
	}
	
	needsAction := !isHealthy
	
	fmt.Printf("👁️ [控制层] 监控结果: 健康=%v, 动作=%s, 问题=%v\n", 
		isHealthy, actionType, issues)
	
	return &MonitorResult{
		IsHealthy:     isHealthy,
		NeedsAction:   needsAction,
		ActionType:    actionType,
		Issues:        issues,
		Confidence:    0.8,
	}
}

// HandleAction 控制层处理异常动作
func (cl *ControlLayer) HandleAction(ctx context.Context, actionType string, agentName string) error {
	switch actionType {
	case "terminate_all":
		fmt.Println("🛑 [控制层] 终止所有执行")
		cl.executionContext.Status = StatusTerminated
		cl.executionContext.IsTerminated = true
		cl.executionContext.TermReason = "控制层判定需要终止所有执行"
		return fmt.Errorf("执行被控制层终止")
	
	case "terminate_agent":
		fmt.Printf("🛑 [控制层] 仅终止 Agent: %s，继续其他测试\n", agentName)
		// 不设置 IsTerminated，只记录问题，继续执行其他 Agent
		// 注意：这里我们需要另外记录问题原因，而不是循环引用自身
		// 这里简化处理，只记录Agent被终止
		cl.executionContext.IssuesFound = append(cl.executionContext.IssuesFound, 
			fmt.Sprintf("Agent %s 被终止", agentName))
		return nil
		
	case "security_review":
		fmt.Println("🔍 [控制层] 调度安全审查 Agent")
		// 调度专门的安全审查 Agent
		reviewer, err := cl.agentMgr.GetAgent("Reviewer")
		if err == nil {
			_, err := reviewer.Execute(ctx, "执行安全审查，检查执行过程")
			if err != nil {
				fmt.Printf("⚠️ 安全审查执行有问题: %v\n", err)
			}
		}
		// 安全审查后继续或终止，这里简化为继续
		return nil
		
	case "notify_management":
		fmt.Println("📢 [控制层] 通知管理层进行系统级处理")
		// 管理层会在后续流程中处理
		return nil
		
	default:
		// "continue"
		return nil
	}
}

// recordMetrics 控制层记录指标（不涉及内容）
func (cl *ControlLayer) recordMetrics(agentName string, phase LoopPhase, startTime time.Time, success bool, flag string, err error) {
	endTime := time.Now()
	metrics := ExecutionMetrics{
		AgentName: agentName,
		Phase:     phase,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Success:   success,
		ExecutionFlag: flag,
	}
	
	if err != nil {
		metrics.Error = err.Error()
	}
	
	cl.metricsHistory = append(cl.metricsHistory, metrics)
}

// =============================================
// 主流程 - 职责分离的完整闭环
// =============================================

// ExecuteWithLoop 全场景闭环执行入口
func (clm *ClosedLoopManager) ExecuteWithLoop(ctx context.Context, task string) (string, error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🔄 三层闭环系统启动 - 职责分离")
	fmt.Println("=" + strings.Repeat("-", 78) + "=")
	fmt.Println("  [执行层] - 业务执行 + 内容安全检查")
	fmt.Println("  [控制层] - 仅监督执行状态，不深入业务内容")
	fmt.Println("  [管理层] - 评审 & Agent优化")
	fmt.Println(strings.Repeat("=", 80))

	taskID := fmt.Sprintf("task-%d", time.Now().Unix())
	var finalResult string

	// 初始化上下文
	clm.controlLayer.executionContext = &ExecutionContext{
		TaskID:        taskID,
		OriginalTask:  task,
		CurrentStep:   0,
		TotalSteps:    0,
		OutputHistory: []string{},
		StartTime:     time.Now(),
		LastUpdate:    time.Now(),
		IsTerminated:  false,
		Status:        StatusRunning,
		ExecutionFlag: "normal",
	}

	// 检查退出指令（这个可以在入口检查）
	if shouldExit, reason := checkExitCommand(task); shouldExit {
		fmt.Printf("⚠️ 检测到退出指令: %s\n", reason)
		return fmt.Sprintf("会话已结束: %s", reason), nil
	}

	for iteration := 1; iteration <= clm.maxIterations; iteration++ {
		fmt.Printf("\n\n🔄 [迭代 %d/%d]\n", iteration, clm.maxIterations)
		fmt.Println(strings.Repeat("-", 80))
		
		// ========== 先检查是否已终止 ==========
		if clm.controlLayer.executionContext.IsTerminated {
			fmt.Printf("⚠️ 任务已终止: %s\n", clm.controlLayer.executionContext.TermReason)
			break
		}
		
		// ========== 步骤 1: 执行层规划 ==========
		fmt.Println("\n📋 [1] 执行层规划任务")
		clm.controlLayer.executionContext.CurrentPhase = PhasePlan
		clm.controlLayer.executionContext.CurrentAgent = "Planner"
		
		planStartTime := time.Now()
		plan, flag, err := clm.executionLayer.ExecuteAgent(ctx, "Planner", 
			fmt.Sprintf("规划以下任务: %s\n第%d次迭代", task, iteration))
		
		// 控制层监控规划过程
		monitorResult := clm.controlLayer.MonitorExecution(ctx, "Planner", 
			time.Since(planStartTime), flag)
		clm.controlLayer.recordMetrics("Planner", PhasePlan, planStartTime, err == nil, flag, err)
		
		if monitorResult.NeedsAction {
			err := clm.controlLayer.HandleAction(ctx, monitorResult.ActionType, "Planner")
			if err != nil {
				return "", err
			}
			if clm.controlLayer.executionContext.IsTerminated {
				break
			}
		}
		
		clm.controlLayer.executionContext.CurrentPlan = plan
		fmt.Printf("✅ 规划完成: %s\n", truncate(plan, 200))
		
		// ========== 步骤 2: 执行层选择主 Agent ==========
		fmt.Println("\n🎯 [2] 执行层选择主 Agent")
		intentStartTime := time.Now()
		intentResult, intentFlag, intentErr := clm.executionLayer.ExecuteAgent(ctx, "Intent", task)
		
		// 控制层监控
		clm.controlLayer.recordMetrics("Intent", PhaseExecute, intentStartTime, intentErr == nil, intentFlag, intentErr)
		
		primaryAgentName := clm.executionLayer.selectAgentFromIntent(intentResult)
		clm.controlLayer.executionContext.CurrentAgent = primaryAgentName
		fmt.Printf("主 Agent 选择: %s\n", primaryAgentName)
		
		// ========== 步骤 3: 执行层执行所有任务 ==========
		fmt.Println("\n⚡ [3] 执行层执行所有子任务")
		
		// 执行主 Agent
		fmt.Printf("\n🚀 执行主 Agent: %s\n", primaryAgentName)
		primaryStartTime := time.Now()
		primaryOutput, primaryFlag, primaryErr := clm.executionLayer.ExecuteAgent(ctx, primaryAgentName, 
			fmt.Sprintf("执行主任务: %s\n规划: %s", task, plan))
		
		// 控制层监控主 Agent
		monitorPrimary := clm.controlLayer.MonitorExecution(ctx, primaryAgentName, 
			time.Since(primaryStartTime), primaryFlag)
		clm.controlLayer.recordMetrics(primaryAgentName, PhaseExecute, primaryStartTime, primaryErr == nil, primaryFlag, primaryErr)
		
		if monitorPrimary.NeedsAction {
			err := clm.controlLayer.HandleAction(ctx, monitorPrimary.ActionType, primaryAgentName)
			// 只有 terminate_all 才返回错误，terminate_agent 继续执行
			if err != nil && monitorPrimary.ActionType == "terminate_all" {
				return "", err
			}
			if clm.controlLayer.executionContext.IsTerminated {
				break
			}
		}
		
		// 执行其他安全 Agent（每个都受监控）
		// 注意：当前使用简单关键词匹配决定执行哪些Agent
		// 更理想的方式是让Planner在规划阶段明确指定需要执行的安全测试类型
		securityAgents := []string{
			"SQLiAgent",
			"XSSAgent", 
			"CommandInjectAgent",
			"PathTraversalAgent",
			"SSRFAgent",
			"FileIncludeAgent",
		}
		
		allOutputs := []string{}
		if primaryOutput != "" {
			allOutputs = append(allOutputs, fmt.Sprintf("### %s 输出:\n%s", primaryAgentName, primaryOutput))
		}
		
		for _, agentName := range securityAgents {
			if clm.controlLayer.executionContext.IsTerminated {
				break
			}
			
			if clm.executionLayer.shouldUseAgent(task, agentName) {
				fmt.Printf("\n🔍 执行安全 Agent: %s\n", agentName)
				agentStartTime := time.Now()
				output, flag, err := clm.executionLayer.ExecuteAgent(ctx, agentName, 
					fmt.Sprintf("测试目标: %s\n规划: %s", task, plan))
				
				// 控制层监控每个安全 Agent
				monitorAgent := clm.controlLayer.MonitorExecution(ctx, agentName, 
					time.Since(agentStartTime), flag)
				clm.controlLayer.recordMetrics(agentName, PhaseExecute, agentStartTime, err == nil, flag, err)
				
				if monitorAgent.NeedsAction {
					err := clm.controlLayer.HandleAction(ctx, monitorAgent.ActionType, agentName)
					// 只有 terminate_all 才返回错误，terminate_agent 继续执行其他测试
					if err != nil && monitorAgent.ActionType == "terminate_all" {
						return "", err
					}
					if clm.controlLayer.executionContext.IsTerminated {
						break
					}
				}
				
				if err == nil && output != "" {
					allOutputs = append(allOutputs, fmt.Sprintf("### %s 输出:\n%s", agentName, output))
				}
			}
		}
		
		// 合并输出
		fullOutput := strings.Join(allOutputs, "\n\n---\n\n")
		clm.controlLayer.executionContext.OutputHistory = append(
			clm.controlLayer.executionContext.OutputHistory,
			fullOutput,
		)
		clm.controlLayer.executionContext.CurrentOutput = fullOutput
		clm.controlLayer.executionContext.LastUpdate = time.Now()
		
		// ========== 步骤 4: 管理层评审 ==========
		fmt.Println("\n📊 [4] 管理层评审")
		reviewResult, err := clm.managementLayer.phaseReview(ctx, taskID, task, fullOutput)
		if err != nil {
			return "", err
		}
		
		if !reviewResult.ShouldRetry {
			fmt.Println("\n✅ 任务完成！")
			finalResult = fullOutput
			
			if reviewResult.AgentAdjustment != nil {
				clm.managementLayer.phaseImprove(ctx, taskID, reviewResult.AgentAdjustment)
			}
			break
		}
		
		// ========== 步骤 5: 继续迭代 ==========
		fmt.Println("\n🔄 继续迭代改进...")
		clm.managementLayer.phaseImprove(ctx, taskID, reviewResult.AgentAdjustment)
		task = task + "\n\n[改进建议]: " + reviewResult.NextIterationHint
	}

	// 最终处理
	if clm.controlLayer.executionContext.IsTerminated {
		finalResult = fmt.Sprintf("任务终止: %s\n\n已收集结果:\n%s", 
			clm.controlLayer.executionContext.TermReason,
			getLatestOutput(clm.controlLayer.executionContext))
	} else if finalResult == "" {
		finalResult = fmt.Sprintf("执行了 %d 次迭代，但未明确完成\n\n最新结果:\n%s", 
			clm.maxIterations,
			getLatestOutput(clm.controlLayer.executionContext))
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🏁 闭环执行结束")
	fmt.Printf("  状态: %s\n", clm.controlLayer.executionContext.Status)
	if clm.controlLayer.executionContext.IsTerminated {
		fmt.Printf("  终止原因: %s\n", clm.controlLayer.executionContext.TermReason)
	}
	fmt.Printf("  监控指标数: %d\n", len(clm.controlLayer.metricsHistory))
	fmt.Println(strings.Repeat("=", 80))

	return finalResult, nil
}

// =============================================
// 其他辅助方法
// =============================================

func checkExitCommand(input string) (bool, string) {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "/quit" || trimmed == "/exit" {
		return true, "用户请求退出"
	}
	return false, ""
}

func (el *ExecutionLayer) selectAgentFromIntent(intentResult string) string {
	lower := strings.ToLower(intentResult)
	switch {
	case strings.Contains(lower, "pentest") || strings.Contains(lower, "security") || strings.Contains(lower, "渗透"):
		return "Pentesting"
	case strings.Contains(lower, "sql"):
		return "SQLiAgent"
	case strings.Contains(lower, "xss"):
		return "XSSAgent"
	case strings.Contains(lower, "command"):
		return "CommandInjectAgent"
	case strings.Contains(lower, "path") || strings.Contains(lower, "traversal"):
		return "PathTraversalAgent"
	case strings.Contains(lower, "ssrf"):
		return "SSRFAgent"
	case strings.Contains(lower, "file") || strings.Contains(lower, "include"):
		return "FileIncludeAgent"
	case strings.Contains(lower, "ctf"):
		return "CTFExploration"
	case strings.Contains(lower, "code") || strings.Contains(lower, "create"):
		return "Coder"
	case strings.Contains(lower, "test"):
		return "Tester"
	case strings.Contains(lower, "debug"):
		return "Debugger"
	case strings.Contains(lower, "review"):
		return "Reviewer"
	case strings.Contains(lower, "git"):
		return "Git"
	default:
		return "Generic Agent"
	}
}

func (el *ExecutionLayer) shouldUseAgent(task, agentName string) bool {
	taskLower := strings.ToLower(task)
	
	// 如果是渗透测试任务，执行所有安全测试Agent
	// 这是一个简化实现，更好的方式是让Planner在规划阶段明确指定需要执行的测试类型
	isPentestTask := strings.Contains(taskLower, "pentest") || 
		strings.Contains(taskLower, "security") || 
		strings.Contains(taskLower, "渗透")
	
	if isPentestTask {
		// 渗透测试任务执行所有安全Agent
		return true
	}
	
	// 非渗透测试任务使用关键词匹配
	keywords := map[string][]string{
		"SQLiAgent":         {"sql", "injection", "sqli", "union", "select"},
		"XSSAgent":          {"xss", "cross", "script", "<script>", "javascript:"},
		"CommandInjectAgent": {"command", "inject", "cmd", "exec", "shell", ";", "&&", "||"},
		"PathTraversalAgent": {"path", "traversal", "../", "directory", "etc/passwd"},
		"SSRFAgent":          {"ssrf", "server side request forgery", "localhost", "127.0.0.1"},
		"FileIncludeAgent":   {"file", "include", "lfi", "rfi", "php://filter"},
	}
	
	if keywordList, exists := keywords[agentName]; exists {
		for _, keyword := range keywordList {
			if strings.Contains(taskLower, keyword) {
				return true
			}
		}
	}
	return false
}

func (ml *ManagementLayer) phaseReview(ctx context.Context, taskID, task, output string) (*ReviewResult, error) {
	startTime := time.Now()
	
	fmt.Printf("📊 [管理层] 执行评审...\n")
	
	metricsCount := len(ml.controlLayer.metricsHistory)
	ctxInfo := ml.controlLayer.executionContext
	
	fmt.Printf("   - 监控指标数: %d\n", metricsCount)
	fmt.Printf("   - 当前状态: %s\n", ctxInfo.Status)
	
	// 计算总体质量（基于执行层返回的标记）
	var successCount int
	var totalCount int
	for _, metric := range ml.controlLayer.metricsHistory {
		if metric.Phase == PhaseExecute {
			totalCount++
			if metric.ExecutionFlag == "normal" {
				successCount++
			}
		}
	}
	
	overallScore := 0.7
	if totalCount > 0 {
		overallScore = float64(successCount) / float64(totalCount)
	}
	
	strengths := []string{}
	weaknesses := []string{}
	improvements := []string{}
	
	if successCount == totalCount && totalCount > 0 {
		strengths = append(strengths, "所有 Agent 正常执行")
	}
	
	// 为每个执行过的Agent计算特性调整
	agentAdjustments := make([]*AgentTraitsAdjustment, 0)
	for _, metric := range ml.controlLayer.metricsHistory {
		if metric.Phase == PhaseExecute {
			if metric.ExecutionFlag == "error" || metric.ExecutionFlag == "honeypot" {
				weaknesses = append(weaknesses, fmt.Sprintf("Agent %s 返回 %s", metric.AgentName, metric.ExecutionFlag))
				// 质量下降
				agentAdjustments = append(agentAdjustments, &AgentTraitsAdjustment{
					AgentName:       metric.AgentName,
					EfficiencyDelta: -0.02,
					QualityDelta:    -0.05,
					CreativityDelta: 0,
				})
			} else if metric.ExecutionFlag == "normal" {
				// 正常执行，质量轻微提升
				agentAdjustments = append(agentAdjustments, &AgentTraitsAdjustment{
					AgentName:       metric.AgentName,
					EfficiencyDelta: 0.01,
					QualityDelta:    0.02,
					CreativityDelta: 0,
				})
			}
		}
	}
	
	if len(weaknesses) > 0 {
		improvements = append(improvements, "检查失败的 Agent")
	}
	
	shouldRetry := len(weaknesses) > 0 && overallScore < 0.7
	nextIterationHint := strings.Join(improvements, "; ")
	
	ml.controlLayer.recordMetrics("ManagementLayer", PhaseReview, startTime, true, "normal", nil)
	
	var mainAdjustment *AgentTraitsAdjustment
	if len(agentAdjustments) > 0 {
		mainAdjustment = agentAdjustments[0] // 暂时只返回第一个作为示例
	}
	
	return &ReviewResult{
		OverallScore:     overallScore,
		Strengths:        strengths,
		Weaknesses:       weaknesses,
		Improvements:     improvements,
		ShouldRetry:      shouldRetry,
		NextIterationHint: nextIterationHint,
		AgentAdjustment:  mainAdjustment,
	}, nil
}

func (ml *ManagementLayer) phaseImprove(ctx context.Context, taskID string, adjustment *AgentTraitsAdjustment) error {
	if adjustment == nil {
		return nil
	}

	fmt.Printf("📈 [管理层] 优化 Agent: %s\n", adjustment.AgentName)
	
	if traits, exists := ml.agentTraits[adjustment.AgentName]; exists {
		oldEfficiency := traits.Efficiency
		oldQuality := traits.Quality
		oldCreativity := traits.Creativity
		
		traits.Efficiency = clamp(traits.Efficiency + adjustment.EfficiencyDelta, 0, 1)
		traits.Quality = clamp(traits.Quality + adjustment.QualityDelta, 0, 1)
		traits.Creativity = clamp(traits.Creativity + adjustment.CreativityDelta, 0, 1)
		
		fmt.Printf("   - 效率: %.2f → %.2f\n", oldEfficiency, traits.Efficiency)
		fmt.Printf("   - 质量: %.2f → %.2f\n", oldQuality, traits.Quality)
		fmt.Printf("   - 创造性: %.2f → %.2f\n", oldCreativity, traits.Creativity)
		
		// 保存到 SKILL.md 文件
		if ml.skillLoader != nil {
			fmt.Printf("💾 [管理层] 保存更新到 SKILL.md\n")
			err := ml.skillLoader.SaveSkill(adjustment.AgentName, traits)
			if err != nil {
				fmt.Printf("⚠️ 保存 SKILL.md 失败: %v\n", err)
			} else {
				fmt.Printf("✅ SKILL.md 更新成功\n")
			}
		}
	}
	
	return nil
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func getLatestOutput(ctx *ExecutionContext) string {
	if len(ctx.OutputHistory) > 0 {
		return ctx.OutputHistory[len(ctx.OutputHistory)-1]
	}
	return ctx.CurrentOutput
}
