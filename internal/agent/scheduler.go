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
// 5. Worker/Watcher/Auditor的启动和停止
// 6. 通道管理（Worker→Watcher→Auditor的信息流转）
// =============================================

// Scheduler 调度器（纯硬编码）
type Scheduler struct {
	llmClient   *llm.Client
	agentMgr    *AgentManager
	skillLoader *SkillLoader
	
	// 系统级配置（硬编码）
	maxIterations   int
	maxDuration     time.Duration
	workerTimeout   time.Duration
	watcherInterval time.Duration
	
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
		llmClient:       llmClient,
		agentMgr:        agentMgr,
		skillLoader:     skillLoader,
		maxIterations:   10,
		maxDuration:     10 * time.Minute,
		workerTimeout:   5 * time.Minute,
		watcherInterval: 100 * time.Millisecond,
	}
}

// ExecuteWithAgentPattern 使用Worker/Watcher/Auditor模式执行任务
// 这是调度器的核心方法，负责：
// 1. 启动Watcher（并行）
// 2. 启动Worker（执行）
// 3. 监控超时（硬编码）
// 4. 处理Watcher的停止决策
// 5. Worker退出后，启动Auditor分析
func (s *Scheduler) ExecuteWithAgentPattern(ctx context.Context, task string) (string, error) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🔄 Worker/Watcher/Auditor 模式启动")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("  [调度器] - 系统调度（硬编码）")
	fmt.Println("  [Worker] - 执行Agent（业务Agent）")
	fmt.Println("  [Watcher] - 控制Agent（监控Agent）")
	fmt.Println("  [Auditor] - 管理Agent（分析Agent）")
	fmt.Println(strings.Repeat("=", 80))
	
	s.mu.Lock()
	s.currentTask = task
	s.startTime = time.Now()
	s.isRunning = true
	s.mu.Unlock()
	
	// =============================================
	// 步骤 1: 硬编码 - 解析用户意图（固定串行计算）
	// =============================================
	fmt.Println("\n📋 [调度器] 解析用户意图（硬编码）")
	intentResult, _, err := s.executeAgentOnce(ctx, "Intent", task)
	if err != nil {
		return "", fmt.Errorf("意图解析失败: %w", err)
	}
	
	// 硬编码 - 根据意图选择Worker
	workerAgentName := s.selectWorkerFromIntent(intentResult)
	fmt.Printf("✅ [调度器] 选择Worker: %s\n", workerAgentName)
	
	// =============================================
	// 步骤 2: 硬编码 - 启动Watcher（并行）
	// =============================================
	fmt.Println("\n👁️ [调度器] 启动Watcher（并行监控）")
	watcher := NewWatcherAgent(s.llmClient)
	watcher.SetOriginalIntent(task) // 设置用户原始意图
	
	// 创建Watcher的执行上下文（独立goroutine）
	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	watcherDone := make(chan string, 1)
	
	go func() {
		result, _ := watcher.Execute(watcherCtx, "监控Worker执行")
		watcherDone <- result
	}()
	
	// =============================================
	// 步骤 3: 硬编码 - 启动Worker（执行）
	// =============================================
	fmt.Printf("\n🚀 [调度器] 启动Worker: %s\n", workerAgentName)
	
	// 创建Worker的执行上下文（带超时）
	workerCtx, workerCancel := context.WithTimeout(ctx, s.workerTimeout)
	
	// Worker执行循环
	workerOutputs := []string{}
	iteration := 0
	
	for iteration < s.maxIterations {
		iteration++
		fmt.Printf("\n🔄 [迭代 %d/%d]\n", iteration, s.maxIterations)
		
		// 硬编码 - 检查总超时
		if time.Since(s.startTime) > s.maxDuration {
			fmt.Println("⏰ [调度器] 总执行时间超时")
			workerCancel()
			break
		}
		
		// 执行Worker
		output, flag, err := s.executeAgentOnce(workerCtx, workerAgentName, 
			fmt.Sprintf("执行任务: %s\n迭代: %d", task, iteration))
		
		// 硬编码 - 生成Worker执行信息
		info := WorkerExecutionInfo{
			AgentName:     workerAgentName,
			Task:          task,
			Output:        output,
			ExecutionFlag: flag,
			Timestamp:     time.Now(),
		}
		
		// 硬编码 - 发送信息给Watcher
		watcher.ReceiveWorkerInfo(info)
		
		// 硬编码 - 定时检查Watcher的决策
		time.Sleep(s.watcherInterval)
		decision := watcher.GetDecision()
		
		if decision.ShouldStop {
			fmt.Printf("🛑 [调度器] Watcher决策停止: %s\n", decision.Reason)
			fmt.Printf("   指引: %s\n", decision.CorrectGuidance)
			workerCancel()
			
			// 如果有正确指引，可以尝试重新执行
			if decision.CorrectGuidance != "" {
				task = task + "\n\n[Watcher指引]: " + decision.CorrectGuidance
			}
			break
		}
		
		// 收集输出
		if output != "" {
			workerOutputs = append(workerOutputs, output)
		}
		
		// 检查是否完成
		if err == nil && flag == "normal" {
			// 简单判断：如果输出包含"完成"或"success"，认为任务完成
			if strings.Contains(strings.ToLower(output), "完成") ||
			   strings.Contains(strings.ToLower(output), "success") ||
			   strings.Contains(strings.ToLower(output), "done") {
				fmt.Println("✅ [调度器] Worker任务完成")
				break
			}
		}
	}
	
	// =============================================
	// 步骤 4: 硬编码 - Worker退出，停止Watcher
	// =============================================
	fmt.Println("\n🏁 [调度器] Worker退出，停止Watcher")
	watcher.StopWorker()
	watcherCancel()
	
	// 等待Watcher汇总
	select {
	case watcherSummary := <-watcherDone:
		fmt.Println("📊 [调度器] 收到Watcher汇总信息")
		
		// =============================================
		// 步骤 5: 硬编码 - 启动Auditor分析
		// =============================================
		fmt.Println("\n🔍 [调度器] 启动Auditor分析")
		auditor := NewAuditorAgent(s.llmClient, s.skillLoader)
		
		auditorResult, err := auditor.Execute(ctx, watcherSummary)
		if err != nil {
			fmt.Printf("⚠️ [调度器] Auditor分析失败: %v\n", err)
		} else {
			fmt.Println("✅ [调度器] Auditor分析完成")
			fmt.Println(auditorResult)
		}
		
		// =============================================
		// 步骤 6: 硬编码 - 汇总最终结果
		// =============================================
		finalResult := s.formatFinalResult(workerOutputs, watcherSummary, auditorResult)
		return finalResult, nil
		
	case <-time.After(5 * time.Second):
		fmt.Println("⏰ [调度器] Watcher汇总超时")
		return strings.Join(workerOutputs, "\n---\n"), nil
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
	
	fmt.Printf("📋 [调度器] %s 执行完成，耗时 %v，标记: %s\n", agentName, duration, flag)
	
	return result, flag, err
}

// selectWorkerFromIntent 根据意图选择Worker（硬编码的匹配规则）
func (s *Scheduler) selectWorkerFromIntent(intentResult string) string {
	lower := strings.ToLower(intentResult)
	
	// 硬编码的匹配规则
	switch {
	case strings.Contains(lower, "pentest") || strings.Contains(lower, "security"):
		return "Pentesting"
	case strings.Contains(lower, "sql"):
		return "SQLiAgent"
	case strings.Contains(lower, "xss"):
		return "XSSAgent"
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

// formatFinalResult 格式化最终结果（硬编码的格式化）
func (s *Scheduler) formatFinalResult(workerOutputs []string, watcherSummary string, auditorResult string) string {
	var result strings.Builder
	
	result.WriteString("\n" + strings.Repeat("=", 80))
	result.WriteString("📋 最终结果汇总\n")
	result.WriteString(strings.Repeat("=", 80))
	
	// Worker输出
	result.WriteString("\n### Worker执行结果:\n")
	for i, output := range workerOutputs {
		result.WriteString(fmt.Sprintf("\n[输出 %d]\n%s\n", i+1, truncate(output, 500)))
	}
	
	// Watcher汇总
	result.WriteString("\n### Watcher监控汇总:\n")
	result.WriteString(watcherSummary)
	
	// Auditor分析
	result.WriteString("\n### Auditor分析结论:\n")
	result.WriteString(auditorResult)
	
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

func (s *Scheduler) SetWorkerTimeout(timeout time.Duration) {
	s.workerTimeout = timeout
}

func (s *Scheduler) SetWatcherInterval(interval time.Duration) {
	s.watcherInterval = interval
}