package agent

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/vibe-coding/free-agent/internal/llm"
)

// =============================================
// 三层闭环管理器 (PD-WC-RI)
// =============================================

// 闭环阶段
type LoopPhase string

const (
	PhasePlan   LoopPhase = "Plan"
	PhaseDo     LoopPhase = "Do"
	PhaseWatch  LoopPhase = "Watch"
	PhaseCorrect LoopPhase = "Correct"
	PhaseReview  LoopPhase = "Review"
	PhaseImprove LoopPhase = "Improve"
)

// 执行状态
type ExecutionState struct {
	TaskID        string
	CurrentPhase  LoopPhase
	Progress      float64 // 0-1
	QualityScore  float64 // 当前质量评分
	Errors        []string
	StartTime     time.Time
	LastUpdated   time.Time
}

// 闭环管理器
type ClosedLoopManager struct {
	mu           sync.RWMutex
	llmClient    *llm.Client
	agentMgr     *AgentManager
	
	// 执行状态
	states       map[string]*ExecutionState
	
	// 知识库 (可持久化)
	knowledge    *KnowledgeBase
	
	// Agent 特质
	agentTraits  map[string]*AgentTraits
}

// Agent 特质
type AgentTraits struct {
	Name          string
	Efficiency    float64 // 效率 0-1
	Quality       float64 // 质量 0-1
	Creativity    float64 // 创造性 0-1
	Collaboration float64 // 协作能力 0-1
	LearningRate  float64 // 学习速度 0-1
}

// 知识库
type KnowledgeBase struct {
	Experiences []LoopExperience
	Patterns    map[string]float64 // 任务模式→成功率
}

// 闭环经验
type LoopExperience struct {
	TaskID       string
	TaskInput    string
	AgentUsed    string
	QualityScore float64
	TimeSpent    time.Duration
	Success      bool
	Timestamp    time.Time
}

func NewClosedLoopManager(llmClient *llm.Client, agentMgr *AgentManager) *ClosedLoopManager {
	return &ClosedLoopManager{
		llmClient:   llmClient,
		agentMgr:    agentMgr,
		states:      make(map[string]*ExecutionState),
		knowledge: &KnowledgeBase{
			Patterns: make(map[string]float64),
		},
		agentTraits: make(map[string]*AgentTraits),
	}
}

// 执行带闭环的任务
func (clm *ClosedLoopManager) ExecuteWithLoop(ctx context.Context, task string) (string, error) {
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	startTime := time.Now()
	
	fmt.Println("\n" + "=" + repeat("=", 78))
	fmt.Println("🔄 [CLOSED-LOOP EXECUTION] Task:", task)
	fmt.Println("=" + repeat("=", 78))
	
	// 初始化执行状态
	clm.mu.Lock()
	clm.states[taskID] = &ExecutionState{
		TaskID:       taskID,
		CurrentPhase: PhasePlan,
		Progress:     0.0,
		QualityScore: 0.0,
		StartTime:    startTime,
		LastUpdated:  startTime,
	}
	clm.mu.Unlock()
	
	var finalResult string
	var err error
	
	// =============================================
	// 第一层：执行层 (Plan-Do)
	// =============================================
	fmt.Println("\n" + repeat("-", 80))
	fmt.Println("📍 [EXECUTION LAYER] Plan → Do")
	fmt.Println(repeat("-", 80))
	
	// 1. Plan 阶段
	planResult, err := clm.phasePlan(ctx, taskID, task)
	if err != nil {
		return "", clm.handlePhaseError(taskID, PhasePlan, err)
	}
	
	clm.updateProgress(taskID, 0.25)
	
	// 2. Do 阶段
	doResult, err := clm.phaseDo(ctx, taskID, task, planResult)
	if err != nil {
		return "", clm.handlePhaseError(taskID, PhaseDo, err)
	}
	
	clm.updateProgress(taskID, 0.5)
	finalResult = doResult
	
	// =============================================
	// 第二层：监督层 (Watch-Correct)
	// =============================================
	fmt.Println("\n" + repeat("-", 80))
	fmt.Println("👁️ [SUPERVISION LAYER] Watch → Correct")
	fmt.Println(repeat("-", 80))
	
	// 3. Watch 阶段
	watchResult, err := clm.phaseWatch(ctx, taskID, doResult)
	if err != nil {
		return "", clm.handlePhaseError(taskID, PhaseWatch, err)
	}
	
	clm.updateProgress(taskID, 0.75)
	
	// 4. Correct 阶段 (如果需要)
	if watchResult.NeedsCorrection {
		correctedResult, err := clm.phaseCorrect(ctx, taskID, watchResult)
		if err != nil {
			fmt.Printf("⚠️ Correction failed: %v, proceeding with original\n", err)
		} else {
			finalResult = correctedResult
		}
	}
	
	// =============================================
	// 第三层：管理层 (Review-Improve)
	// =============================================
	fmt.Println("\n" + repeat("-", 80))
	fmt.Println("🏛️ [MANAGEMENT LAYER] Review → Improve")
	fmt.Println(repeat("-", 80))
	
	// 5. Review 阶段
	review, err := clm.phaseReview(ctx, taskID, finalResult)
	if err != nil {
		fmt.Printf("⚠️ Review failed: %v\n", err)
	} else {
		fmt.Printf("\n📊 Review Score: %.2f\n", review.QualityScore)
		clm.states[taskID].QualityScore = review.QualityScore
	}
	
	clm.updateProgress(taskID, 1.0)
	
	// 6. Improve 阶段
	if err := clm.phaseImprove(ctx, taskID, review); err != nil {
		fmt.Printf("⚠️ Improve failed: %v\n", err)
	}
	
	// 记录经验
	clm.recordExperience(taskID, task, review)
	
	fmt.Println("\n" + "=" + repeat("=", 78))
	fmt.Println("✅ [CLOSED-LOOP COMPLETE]")
	fmt.Println("=" + repeat("=", 78))
	
	return finalResult, nil
}

// =============================================
// 执行层 (Plan-Do)
// =============================================

func (clm *ClosedLoopManager) phasePlan(ctx context.Context, taskID, task string) (string, error) {
	fmt.Println("\n📋 [Plan] Creating execution plan...")
	
	agent, err := clm.agentMgr.GetAgent("Planner")
	if err != nil {
		return "", err
	}
	
	result, err := agent.Execute(ctx, task)
	if err != nil {
		return "", err
	}
	
	fmt.Println("   └─ Plan created")
	return result, nil
}

func (clm *ClosedLoopManager) phaseDo(ctx context.Context, taskID, task, plan string) (string, error) {
	fmt.Println("\n🔨 [Do] Executing plan...")
	
	// 让 Orchestrator 选择合适的 Agent
	agent, err := clm.agentMgr.GetAgent("Orchestrator")
	if err != nil {
		return "", err
	}
	
	result, err := agent.Execute(ctx, task)
	if err != nil {
		return "", err
	}
	
	fmt.Println("   └─ Plan executed")
	return result, nil
}

// =============================================
// 监督层 (Watch-Correct)
// =============================================

type WatchResult struct {
	NeedsCorrection bool
	Issues          []string
	OriginalOutput  string
}

func (clm *ClosedLoopManager) phaseWatch(ctx context.Context, taskID, output string) (*WatchResult, error) {
	fmt.Println("\n🔍 [Watch] Monitoring execution quality...")
	
	// 使用 Reviewer 进行质量监控
	agent, err := clm.agentMgr.GetAgent("Reviewer")
	if err != nil {
		return &WatchResult{NeedsCorrection: false}, err
	}
	
	reviewResult, err := agent.Execute(ctx, output)
	if err != nil {
		return &WatchResult{NeedsCorrection: false}, err
	}
	
	// 判断是否需要纠正 (简化逻辑)
	needsCorrection := len(reviewResult) > 100 && len(reviewResult) < 500
	
	result := &WatchResult{
		NeedsCorrection: needsCorrection,
		OriginalOutput:  output,
	}
	
	fmt.Printf("   └─ Needs correction: %v\n", needsCorrection)
	return result, nil
}

func (clm *ClosedLoopManager) phaseCorrect(ctx context.Context, taskID string, watch *WatchResult) (string, error) {
	fmt.Println("\n🔧 [Correct] Applying corrections...")
	
	// 使用 Debugger 进行纠正
	agent, err := clm.agentMgr.GetAgent("Debugger")
	if err != nil {
		return "", err
	}
	
	result, err := agent.Execute(ctx, "Fix: "+watch.OriginalOutput)
	if err != nil {
		return "", err
	}
	
	fmt.Println("   └─ Corrections applied")
	return result, nil
}

// =============================================
// 管理层 (Review-Improve)
// =============================================

type ReviewResult struct {
	QualityScore float64
	Strengths    []string
	Weaknesses   []string
}

func (clm *ClosedLoopManager) phaseReview(ctx context.Context, taskID, output string) (*ReviewResult, error) {
	fmt.Println("\n📋 [Review] Evaluating overall results...")
	
	// 使用 Feedback 进行评估
	_, err := clm.agentMgr.GetAgent("Feedback")
	if err != nil {
		return &ReviewResult{QualityScore: 0.5}, err
	}
	
	// 简化评分逻辑 (实际中应该用 LLM 或更复杂的规则)
	score := 0.7
	if len(output) > 200 {
		score += 0.1
	}
	if len(output) > 500 {
		score += 0.05
	}
	
	score = math.Min(1.0, score)
	
	return &ReviewResult{
		QualityScore: score,
		Strengths:    []string{"Task completed"},
		Weaknesses:   []string{"Needs optimization"},
	}, nil
}

func (clm *ClosedLoopManager) phaseImprove(ctx context.Context, taskID string, review *ReviewResult) error {
	fmt.Println("\n✨ [Improve] Optimizing system...")
	
	// 根据质量分数调整 Agent 特质
	score := review.QualityScore
	reward := score - 0.5
	
	clm.mu.Lock()
	defer clm.mu.Unlock()
	
	for agentName, traits := range clm.agentTraits {
		adjustment := reward * traits.LearningRate
		
		traits.Quality = math.Max(0.1, math.Min(1.0, traits.Quality+adjustment*0.25))
		traits.Efficiency = math.Max(0.1, math.Min(1.0, traits.Efficiency+adjustment*0.15))
		
		fmt.Printf("   └─ %s: quality=%.3f, efficiency=%.3f\n",
			agentName, traits.Quality, traits.Efficiency)
	}
	
	return nil
}

// =============================================
// 辅助函数
// =============================================

func (clm *ClosedLoopManager) updateProgress(taskID string, progress float64) {
	clm.mu.Lock()
	defer clm.mu.Unlock()
	
	if state, ok := clm.states[taskID]; ok {
		state.Progress = progress
		state.LastUpdated = time.Now()
	}
}

func (clm *ClosedLoopManager) handlePhaseError(taskID string, phase LoopPhase, err error) error {
	clm.mu.Lock()
	if state, ok := clm.states[taskID]; ok {
		state.Errors = append(state.Errors, fmt.Sprintf("%s: %v", phase, err))
	}
	clm.mu.Unlock()
	
	return err
}

func (clm *ClosedLoopManager) recordExperience(taskID, task string, review *ReviewResult) {
	clm.mu.Lock()
	defer clm.mu.Unlock()
	
	state := clm.states[taskID]
	duration := time.Since(state.StartTime)
	
	experience := LoopExperience{
		TaskID:       taskID,
		TaskInput:    task,
		QualityScore: review.QualityScore,
		TimeSpent:    duration,
		Success:      review.QualityScore > 0.6,
		Timestamp:    time.Now(),
	}
	
	clm.knowledge.Experiences = append(clm.knowledge.Experiences, experience)
}

func (clm *ClosedLoopManager) RegisterAgentTraits(name string, traits *AgentTraits) {
	clm.mu.Lock()
	defer clm.mu.Unlock()
	clm.agentTraits[name] = traits
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
