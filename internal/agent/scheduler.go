package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
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
func NewScheduler(llmClient *llm.Client, agentMgr *AgentManager, skillLoader *SkillLoader) *Scheduler {
	return &Scheduler{
		llmClient:        llmClient,
		agentMgr:         agentMgr,
		skillLoader:      skillLoader,
		maxIterations:    10,
		maxDuration:      10 * time.Minute,
		executorTimeout:  5 * time.Minute,
		observerInterval: 100 * time.Millisecond,
	}
}

// ExecuteWithAgentPattern 使用Executor/Observer/Evaluator模式执行任务
// 这是调度器的核心方法，负责：
// 1. 启动Observer（并行）
// 2. 启动Executor（执行）
// 3. 监控超时（硬编码）
// 4. 处理Observer的停止决策
// 5. Executor退出后，启动Evaluator分析
func (s *Scheduler) ExecuteWithAgentPattern(ctx context.Context, task string) (string, error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🔄 Executor/Observer/Evaluator 模式启动")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("  [Scheduler] - 系统调度（硬编码）")
	fmt.Println("  [Executor] - 执行Agent（业务Agent）")
	fmt.Println("  [Observer] - 控制Agent（监控Agent）")
	fmt.Println("  [Evaluator] - 管理Agent（分析Agent）")
	fmt.Println(strings.Repeat("=", 80))
	
	s.mu.Lock()
	s.currentTask = task
	s.startTime = time.Now()
	s.isRunning = true
	s.mu.Unlock()
	
	// =============================================
	// 步骤 1: 硬编码 - 解析用户意图（固定串行计算）
	// =============================================
	fmt.Println("\n📋 [Scheduler] 解析用户意图（硬编码）")
	intentResult, _, err := s.executeAgentOnce(ctx, "IntentAnalyzer", task)
	if err != nil {
		return "", fmt.Errorf("意图解析失败: %w", err)
	}
	
	// 硬编码 - 根据意图选择Executor
	executorAgentName := s.selectExecutorFromIntent(intentResult)
	fmt.Printf("✅ [Scheduler] 选择Executor: %s\n", executorAgentName)
	
	// =============================================
	// 步骤 2: 硬编码 - 启动Observer（并行）
	// =============================================
	fmt.Println("\n👁️ [Scheduler] 启动Observer（并行监控）")
	observer := NewObserverAgent(s.llmClient)
	observer.SetOriginalIntent(task) // 设置用户原始意图
	
	// 创建Observer的执行上下文（独立goroutine）
	observerCtx, observerCancel := context.WithCancel(context.Background())
	observerDone := make(chan string, 1)
	
	go func() {
		result, _ := observer.Execute(observerCtx, "监控Executor执行")
		observerDone <- result
	}()
	
	// =============================================
	// 步骤 3: 硬编码 - 启动Executor（执行）
	// =============================================
	fmt.Printf("\n🚀 [Scheduler] 启动Executor: %s\n", executorAgentName)
	
	// 创建Executor的执行上下文（带超时）
	executorCtx, executorCancel := context.WithTimeout(ctx, s.executorTimeout)
	
	// Executor执行循环
	executorOutputs := []string{}
	iteration := 0
	
	for iteration < s.maxIterations {
		iteration++
		fmt.Printf("\n🔄 [迭代 %d/%d]\n", iteration, s.maxIterations)
		
		// 硬编码 - 检查总超时
		if time.Since(s.startTime) > s.maxDuration {
			fmt.Println("⏰ [Scheduler] 总执行时间超时")
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
			fmt.Printf("🛑 [Scheduler] Observer决策停止: %s\n", decision.Reason)
			fmt.Printf("   指引: %s\n", decision.CorrectGuidance)
			executorCancel()
			
			// 如果有正确指引，可以尝试重新执行
			if decision.CorrectGuidance != "" {
				task = task + "\n\n[Observer指引]: " + decision.CorrectGuidance
			}
			break
		}
		
		// 收集输出
		if output != "" {
			executorOutputs = append(executorOutputs, output)
		}
		
		// 检查是否完成
		if err == nil && flag == "normal" {
			// 简单判断：如果输出包含"完成"或"success"，认为任务完成
			if strings.Contains(strings.ToLower(output), "完成") ||
			   strings.Contains(strings.ToLower(output), "success") ||
			   strings.Contains(strings.ToLower(output), "done") {
				fmt.Println("✅ [Scheduler] Executor任务完成")
				break
			}
		}
	}
	
	// =============================================
	// 步骤 4: 硬编码 - Executor退出，停止Observer
	// =============================================
	fmt.Println("\n🏁 [Scheduler] Executor退出，停止Observer")
	observer.StopExecutor()
	observerCancel()
	
	// 等待Observer汇总
	select {
	case observerSummary := <-observerDone:
		fmt.Println("📊 [Scheduler] 收到Observer汇总信息")
		
		// =============================================
		// 步骤 5: 硬编码 - 启动Evaluator分析
		// =============================================
		fmt.Println("\n🔍 [Scheduler] 启动Evaluator分析")
		evaluator := NewEvaluatorAgent(s.llmClient, s.skillLoader)
		
		evaluatorResult, err := evaluator.Execute(ctx, observerSummary)
		if err != nil {
			fmt.Printf("⚠️ [Scheduler] Evaluator分析失败: %v\n", err)
		} else {
			fmt.Println("✅ [Scheduler] Evaluator分析完成")
			fmt.Println(evaluatorResult)
		}
		
		// =============================================
		// 步骤 6: 硬编码 - 汇总最终结果
		// =============================================
		finalResult := s.formatFinalResult(executorOutputs, observerSummary, evaluatorResult)
		return finalResult, nil
		
	case <-time.After(5 * time.Second):
		fmt.Println("⏰ [Scheduler] Observer汇总超时")
		return strings.Join(executorOutputs, "\n---\n"), nil
	}
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
	
	fmt.Printf("📋 [Scheduler] %s 执行完成，耗时 %v，标记: %s\n", agentName, duration, flag)
	
	return result, flag, err
}

// selectExecutorFromIntent 根据意图选择Executor（硬编码的匹配规则）
func (s *Scheduler) selectExecutorFromIntent(intentResult string) string {
	lower := strings.ToLower(intentResult)
	
	// 硬编码的匹配规则 - 对应OWASP Top 10和行业术语
	switch {
	case strings.Contains(lower, "pentest") || strings.Contains(lower, "security"):
		return "SecurityAssessor"
	case strings.Contains(lower, "sql"):
		return "SQLInjectionScanner" // OWASP A03
	case strings.Contains(lower, "xss"):
		return "XSSScanner" // OWASP A03
	case strings.Contains(lower, "command") || strings.Contains(lower, "inject"):
		return "CommandInjectionScanner" // OWASP A03
	case strings.Contains(lower, "path") || strings.Contains(lower, "traversal"):
		return "PathTraversalScanner" // OWASP A01
	case strings.Contains(lower, "ssrf"):
		return "SSRFScanner" // OWASP A10
	case strings.Contains(lower, "file") || strings.Contains(lower, "include"):
		return "FileIncludeScanner" // OWASP A03
	case strings.Contains(lower, "ctf"):
		return "CTFSolver"
	case strings.Contains(lower, "code") || strings.Contains(lower, "create"):
		return "CodeGenerator"
	case strings.Contains(lower, "test"):
		return "TestEngineer"
	case strings.Contains(lower, "debug"):
		return "DebugAnalyst"
	case strings.Contains(lower, "review"):
		return "CodeReviewer"
	case strings.Contains(lower, "git"):
		return "GitOperator"
	default:
		return "GeneralHandler"
	}
}

// formatFinalResult 格式化最终结果（硬编码的格式化）
func (s *Scheduler) formatFinalResult(executorOutputs []string, observerSummary string, evaluatorResult string) string {
	var result strings.Builder
	
	result.WriteString("\n" + strings.Repeat("=", 80))
	result.WriteString("📋 最终结果汇总\n")
	result.WriteString(strings.Repeat("=", 80))
	
	// Executor输出
	result.WriteString("\n### Executor执行结果:\n")
	for i, output := range executorOutputs {
		result.WriteString(fmt.Sprintf("\n[输出 %d]\n%s\n", i+1, truncate(output, 500)))
	}
	
	// Observer汇总
	result.WriteString("\n### Observer监控汇总:\n")
	result.WriteString(observerSummary)
	
	// Evaluator分析
	result.WriteString("\n### Evaluator分析结论:\n")
	result.WriteString(evaluatorResult)
	
	result.WriteString("\n" + strings.Repeat("=", 80))
	
	return result.String()
}

// =============================================
// 系统级配置方法（硬编码）
// =============================================

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