package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
	"github.com/vibe-coding/free-agent/internal/memory"
)

// =============================================
// 调度器 - 纯硬编码的系统调度功能
// =============================================
// 调度器职责（硬编码部分）：
// 1. 计时器/超时管理
// 2. 定时任务调度
// 3. 接收返回状态
// 4. 固定串行计算（如：Intent解析、Agent选择）
// 5. Executor/Observer/Evaluator的启动和停止
// 6. 通道管理（Executor→Observer→Evaluator的信息流转）
// =============================================

// Scheduler 调度器（纯硬编码）
type Scheduler struct {
	llmClient   *llm.Client
	agentMgr    *AgentManager
	skillLoader *SkillLoader
	store       *memory.Store
	
	// 系统级配置（硬编码）
	maxIterations    int
	maxDuration      time.Duration
	executorTimeout  time.Duration
	observerInterval time.Duration
	
	// 调度状态（硬编码管理）
	mu             sync.Mutex
	currentTask    string
	originalIntent string
	startTime      time.Time
	isRunning      bool
}

// NewScheduler 创建调度器
func NewScheduler(llmClient *llm.Client, agentMgr *AgentManager, skillLoader *SkillLoader, store *memory.Store) *Scheduler {
	return &Scheduler{
		llmClient:        llmClient,
		agentMgr:         agentMgr,
		skillLoader:      skillLoader,
		store:            store,
		maxIterations:    10,
		maxDuration:      10 * time.Minute,
		executorTimeout:  5 * time.Minute,
		observerInterval: 100 * time.Millisecond,
	}
}

// ExecuteWithAgentPattern 使用Executor/Observer/Evaluator模式执行任务
// 这是调度器的核心方法，负责：
// 1. 委托 TaskCoordinator 进行意图分析和Agent选择
// 2. 启动Observer（并行）
// 3. 启动Executor（执行）
// 4. 监控超时（硬编码）
// 5. 处理Observer的停止决策
// 6. Executor退出后，启动Evaluator分析
func (s *Scheduler) ExecuteWithAgentPattern(ctx context.Context, task string) (string, error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("[MODE] Executor/Observer/Evaluator Pattern Started")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("  [Scheduler]        - System orchestration (hardcoded)")
	fmt.Println("  [TaskCoordinator]  - Intent analysis & Agent selection (Agent)")
	fmt.Println("  [Executor]         - Execution Agent (business Agent)")
	fmt.Println("  [Observer]         - Control Agent (monitoring Agent)")
	fmt.Println("  [Evaluator]        - Management Agent (analysis Agent)")
	fmt.Println(strings.Repeat("=", 80))
	
	s.mu.Lock()
	s.currentTask = task
	s.startTime = time.Now()
	s.isRunning = true
	s.mu.Unlock()
	
	// =============================================
	// 步骤 1: 硬编码 - 委托 TaskCoordinator 进行意图分析+Agent选择
	// =============================================
	fmt.Println("\n[Scheduler] Delegating intent analysis to TaskCoordinator")
	intentInfo, executorAgentName, needPlan := s.selectExecutorViaCoordinator(ctx, task)
	
	if needPlan {
		// 多Agent协作路径：TaskPlanner分解任务，依次执行子任务
		fmt.Println("[Scheduler] Multi-agent plan required, entering multi-executor mode")
		return s.executeMultiAgentWithObserver(ctx, task, intentInfo)
	}
	
	fmt.Printf("[Scheduler] Selected Executor: %s (confidence=%.2f, intent=%s)\n",
		executorAgentName, intentInfo.Confidence, intentInfo.Intent)
	
	// =============================================
	// 步骤 2-6: 单Executor执行路径（Observer+Evaluator闭环）
	// =============================================
	return s.executeSingleWithObserver(ctx, task, executorAgentName)
}

// =============================================
// 硬编码辅助方法
// =============================================

// executeAgentOnce 执行一次Agent（硬编码的执行调用）
func (s *Scheduler) executeAgentOnce(ctx context.Context, agentName string, task string) (string, string, error) {
	agent, err := s.agentMgr.GetAgent(agentName)
	if err != nil {
		return "", "error", err
	}
	
	startTime := time.Now()
	result, err := agent.Execute(ctx, task)
	duration := time.Since(startTime)
	
	// 硬编码 - 简单的标记判断
	flag := "normal"
	if err != nil {
		flag = "error"
	} else if strings.Contains(strings.ToLower(result), "error") {
		flag = "warning"
	} else if strings.Contains(strings.ToLower(result), "被抓") ||
	          strings.Contains(strings.ToLower(result), "caught") {
		flag = "honeypot"
	}
	
	fmt.Printf("[Scheduler] %s execution completed, duration: %v, flag: %s\n", agentName, duration, flag)
	
	return result, flag, err
}

// selectExecutorViaCoordinator 委托 TaskCoordinator 进行意图分析和Agent选择
// 返回：意图分析结果、目标Agent名称、是否需要多Agent协作
func (s *Scheduler) selectExecutorViaCoordinator(ctx context.Context, task string) (*IntentResult, string, bool) {
	// 尝试通过 TaskCoordinator 获取意图分析
	coordinator, err := s.agentMgr.GetAgent("TaskCoordinator")
	if err != nil {
		// TaskCoordinator 不可用，回退到 GeneralHandler
		fmt.Printf("[Scheduler] TaskCoordinator unavailable (%v), falling back to GeneralHandler\n", err)
		return &IntentResult{Intent: IntentUnknown, Confidence: 0.0, AgentName: "GeneralHandler"}, "GeneralHandler", false
	}
	
	// 调用 TaskCoordinator 的 IntentAnalyzer 获取结构化意图
	// TaskCoordinator.Execute 内部会调用 IntentAnalyzer，但会直接路由执行
	// 这里我们直接调用 IntentAnalyzer 获取结构化结果用于决策
	intentAgent, err := s.agentMgr.GetAgent("IntentAnalyzer")
	if err != nil {
		fmt.Printf("[Scheduler] IntentAnalyzer unavailable (%v), falling back to GeneralHandler\n", err)
		return &IntentResult{Intent: IntentUnknown, Confidence: 0.0, AgentName: "GeneralHandler"}, "GeneralHandler", false
	}
	
	intentOutput, err := intentAgent.Execute(ctx, task)
	if err != nil {
		fmt.Printf("[Scheduler] IntentAnalyzer failed (%v), falling back to GeneralHandler\n", err)
		return &IntentResult{Intent: IntentUnknown, Confidence: 0.0, AgentName: "GeneralHandler"}, "GeneralHandler", false
	}
	
	intentInfo, err := parseIntentResult(intentOutput)
	if err != nil {
		fmt.Printf("[Scheduler] Intent parse failed (%v), falling back to GeneralHandler\n", err)
		return &IntentResult{Intent: IntentUnknown, Confidence: 0.0, AgentName: "GeneralHandler"}, "GeneralHandler", false
	}
	
	// 低置信度回退
	if intentInfo.Confidence < 0.6 {
		fmt.Printf("[Scheduler] Low confidence (%.2f), routing to GeneralHandler\n", intentInfo.Confidence)
		intentInfo.AgentName = "GeneralHandler"
	}
	
	_ = coordinator // TaskCoordinator 已验证可用，多Agent路径可使用
	return intentInfo, intentInfo.AgentName, intentInfo.NeedPlan
}

// executeMultiAgentWithObserver 多Agent协作执行路径
// TaskPlanner分解任务为子任务，每个子任务通过Observer监控执行
func (s *Scheduler) executeMultiAgentWithObserver(ctx context.Context, task string, intentInfo *IntentResult) (string, error) {
	// 使用 TaskPlanner 分解任务
	planner, err := s.agentMgr.GetAgent("TaskPlanner")
	if err != nil {
		fmt.Printf("[Scheduler] TaskPlanner unavailable (%v), falling back to single-executor\n", err)
		return s.executeSingleWithObserver(ctx, task, intentInfo.AgentName)
	}
	
	fmt.Println("[Scheduler] TaskPlanner decomposing task into subtasks...")
	planResult, err := planner.Execute(ctx, fmt.Sprintf("Break down this task into independent subtasks. Specify which agent type should handle each subtask: %s", task))
	if err != nil {
		fmt.Printf("[Scheduler] TaskPlanner failed (%v), falling back to single-executor\n", err)
		return s.executeSingleWithObserver(ctx, task, intentInfo.AgentName)
	}
	
	subtasks := parsePlanToTasks(planResult)
	if len(subtasks) == 0 {
		fmt.Println("[Scheduler] No subtasks generated, falling back to single-executor")
		return s.executeSingleWithObserver(ctx, task, intentInfo.AgentName)
	}
	
	fmt.Printf("[Scheduler] Plan generated: %d subtasks\n", len(subtasks))
	
	// 依次执行每个子任务，每个子任务都有 Observer 闭环
	var results strings.Builder
	results.WriteString("=== Multi-Agent Execution Plan ===\n")
	results.WriteString(fmt.Sprintf("Plan: %s\n\n", truncate(planResult, 500)))
	results.WriteString("=== Subtask Results ===\n")
	
	for i, sub := range subtasks {
		// 检查总超时
		if time.Since(s.startTime) > s.maxDuration {
			fmt.Println("[Scheduler] Total execution timeout during multi-agent execution")
			break
		}
		
		fmt.Printf("\n[Scheduler] --- Subtask %d/%d: %s ---\n", i+1, len(subtasks), sub.AgentType)
		subOutput, subErr := s.executeSingleWithObserver(ctx, sub.Input, sub.AgentType)
		if subErr != nil {
			results.WriteString(fmt.Sprintf("\n[%d] %s -> error: %v\n", i+1, sub.AgentType, subErr))
			continue
		}
		results.WriteString(fmt.Sprintf("\n[%d] %s\n%s\n", i+1, sub.AgentType, subOutput))
	}
	
	return results.String(), nil
}

// executeSingleWithObserver 单Executor执行路径（Observer+Evaluator闭环）
func (s *Scheduler) executeSingleWithObserver(ctx context.Context, task string, executorAgentName string) (string, error) {
	// =============================================
	// 启动Observer（并行）
	// =============================================
	fmt.Println("\n[Scheduler] Starting Observer (parallel monitoring)")
	observer := NewObserverAgent(s.llmClient)
	observer.SetOriginalIntent(task)
	
	observerCtx, observerCancel := context.WithCancel(context.Background())
	observerDone := make(chan string, 1)
	
	go func() {
		result, _ := observer.Execute(observerCtx, "监控Executor执行")
		observerDone <- result
	}()
	
	// =============================================
	// 启动Executor（执行）
	// =============================================
	fmt.Printf("\n[Scheduler] Starting Executor: %s\n", executorAgentName)
	
	executorCtx, executorCancel := context.WithTimeout(ctx, s.executorTimeout)
	defer executorCancel()

	executorOutputs := []string{}
	skipLoop := false

	// 特殊路径：SecurityAssessor 使用 per-scanner Observer 监控
	if executorAgentName == "SecurityAssessor" {
		if sa, ok := s.getSecurityAssessor(); ok {
			fmt.Println("[Scheduler] Using per-scanner Observer monitoring for SecurityAssessor")
			output, saErr := sa.RunWithObserver(executorCtx, task, observer)
			
			flag := "normal"
			if saErr != nil {
				flag = "error"
			}
			info := ExecutorExecutionInfo{
				AgentName:     executorAgentName,
				Task:          task,
				Output:        truncate(output, 500),
				ExecutionFlag: flag,
				Timestamp:     time.Now(),
			}
			observer.ReceiveExecutorInfo(info)
			
			if output != "" {
				executorOutputs = append(executorOutputs, output)
			}
			skipLoop = true
		}
	}

	if !skipLoop {
	iteration := 0
	for iteration < s.maxIterations {
		iteration++
		fmt.Printf("\n[Iteration %d/%d]\n", iteration, s.maxIterations)
		
		// 硬编码 - 检查总超时
		if time.Since(s.startTime) > s.maxDuration {
			fmt.Println("[Scheduler] Total execution timeout")
			executorCancel()
			break
		}
		
		// 执行Executor
		output, flag, err := s.executeAgentOnce(executorCtx, executorAgentName,
			fmt.Sprintf("执行任务: %s\n迭代: %d", task, iteration))
		
		// 硬编码 - 生成Executor执行信息
		info := ExecutorExecutionInfo{
			AgentName:     executorAgentName,
			Task:          task,
			Output:        output,
			ExecutionFlag: flag,
			Timestamp:     time.Now(),
		}
		
		// 硬编码 - 发送信息给Observer
		observer.ReceiveExecutorInfo(info)
		
		// 硬编码 - 定时检查Observer的决策
		time.Sleep(s.observerInterval)
		decision := observer.GetDecision()
		
		if decision.ShouldStop {
			fmt.Printf("[Scheduler] Observer decision to stop: %s\n", decision.Reason)
			fmt.Printf("   Guidance: %s\n", decision.CorrectGuidance)
			executorCancel()
			
			if decision.CorrectGuidance != "" {
				task = task + "\n\n[Observer Guidance]: " + decision.CorrectGuidance
			}
			break
		}
		
		// 收集输出
		if output != "" {
			executorOutputs = append(executorOutputs, output)
		}
		
		// 检查是否完成
		if err == nil && flag == "normal" {
			if strings.Contains(strings.ToLower(output), "完成") ||
			   strings.Contains(strings.ToLower(output), "success") ||
			   strings.Contains(strings.ToLower(output), "done") {
				fmt.Println("[Scheduler] Executor task completed")
				break
			}
		}
	}
	} // end if !skipLoop
	
	// =============================================
	// Executor退出，停止Observer
	// =============================================
	fmt.Println("\n[Scheduler] Executor exited, stopping Observer")
	observer.StopExecutor()
	observerCancel()
	
	// 等待Observer汇总
	select {
	case observerSummary := <-observerDone:
		fmt.Println("[Scheduler] Received Observer summary")
		
		// =============================================
		// 启动Evaluator分析
		// =============================================
		fmt.Println("\n[Scheduler] Starting Evaluator analysis")
		evaluator := NewEvaluatorAgent(s.llmClient, s.skillLoader, s.store)
		
		evaluatorResult, err := evaluator.Execute(ctx, observerSummary)
		if err != nil {
			fmt.Printf("[Scheduler] Evaluator analysis failed: %v\n", err)
		} else {
			fmt.Println("[Scheduler] Evaluator analysis completed")
			fmt.Println(evaluatorResult)
		}
		
		// =============================================
		// 汇总最终结果
		// =============================================
		finalResult := s.formatFinalResult(executorOutputs, observerSummary, evaluatorResult)
		return finalResult, nil
		
	case <-time.After(5 * time.Second):
		fmt.Println("[Scheduler] Observer summary timeout")
		return strings.Join(executorOutputs, "\n---\n"), nil
	}
}

// formatFinalResult formats final result (hardcoded formatting)
func (s *Scheduler) formatFinalResult(executorOutputs []string, observerSummary string, evaluatorResult string) string {
	var result strings.Builder
	
	result.WriteString("\n" + strings.Repeat("=", 80))
	result.WriteString("FINAL RESULT\n")
	result.WriteString(strings.Repeat("=", 80))
	
	// Executor outputs
	result.WriteString("\n### Executor Execution Results:\n")
	for i, output := range executorOutputs {
		result.WriteString(fmt.Sprintf("\n[Output %d]\n%s\n", i+1, truncate(output, 500)))
	}
	
	// Observer summary
	result.WriteString("\n### Observer Monitoring Summary:\n")
	result.WriteString(observerSummary)
	
	// Evaluator analysis
	result.WriteString("\n### Evaluator Analysis Conclusion:\n")
	result.WriteString(evaluatorResult)
	
	result.WriteString("\n" + strings.Repeat("=", 80))
	
	return result.String()
}

// =============================================
// 系统级配置方法（硬编码）
// =============================================

// getSecurityAssessor 获取 SecurityAssessor Agent（类型断言）
func (s *Scheduler) getSecurityAssessor() (*SecurityAssessorAgent, bool) {
	agent, err := s.agentMgr.GetAgent("SecurityAssessor")
	if err != nil {
		return nil, false
	}
	sa, ok := agent.(*SecurityAssessorAgent)
	return sa, ok
}

func (s *Scheduler) SetMaxIterations(max int) {
	s.maxIterations = max
}

func (s *Scheduler) SetMaxDuration(duration time.Duration) {
	s.maxDuration = duration
}

func (s *Scheduler) SetExecutorTimeout(timeout time.Duration) {
	s.executorTimeout = timeout
}

func (s *Scheduler) SetObserverInterval(interval time.Duration) {
	s.observerInterval = interval
}