package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AgentTask 代理任务
type AgentTask struct {
	ID         string
	AgentType  string
	Input      string
	Context    context.Context
	Priority   int
	RetryCount int
}

// AgentResult 代理执行结果
type AgentResult struct {
	TaskID     string
	Output     string
	Error      error
	AgentID    string
	AgentType  string
	StartTime  time.Time
	EndTime    time.Time
}

// TaskState 任务状态
type TaskState int

const (
	TaskPending     TaskState = iota // 等待执行
	TaskRunning                      // 执行中
	TaskCompleted                    // 完成
	TaskFailed                       // 失败
	TaskRetrying                     // 重试中
)

// SubTask 子任务
type SubTask struct {
	ID          string
	ParentID    string
	AgentType   string
	Input       string
	Result      string
	State       TaskState
	Error       error
	StartedAt   time.Time
	CompletedAt time.Time
}

// TaskStateManager 任务状态管理器
type TaskStateManager struct {
	mu           sync.Mutex
	tasks        map[string]*SubTask
	completedCnt int
	totalCnt     int
}

func NewTaskStateManager() *TaskStateManager {
	return &TaskStateManager{
		tasks: make(map[string]*SubTask),
	}
}

func (tsm *TaskStateManager) AddSubTask(task *SubTask) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	task.ID = fmt.Sprintf("%s-%d", task.ParentID, tsm.totalCnt+1)
	task.State = TaskPending
	task.StartedAt = time.Now()
	tsm.tasks[task.ID] = task
	tsm.totalCnt++
}

func (tsm *TaskStateManager) UpdateTask(taskID string, state TaskState, result string, err error) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	if task, ok := tsm.tasks[taskID]; ok {
		task.State = state
		task.Result = result
		task.Error = err
		task.CompletedAt = time.Now()
		if state == TaskCompleted {
			tsm.completedCnt++
		}
	}
}

func (tsm *TaskStateManager) GetProgress() float64 {
	if tsm.totalCnt == 0 {
		return 0
	}
	return float64(tsm.completedCnt) / float64(tsm.totalCnt)
}

func (tsm *TaskStateManager) GetTaskByID(taskID string) (*SubTask, bool) {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	task, ok := tsm.tasks[taskID]
	return task, ok
}

func (tsm *TaskStateManager) GetAllTasks() []*SubTask {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()
	tasks := make([]*SubTask, 0, len(tsm.tasks))
	for _, task := range tsm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// AgentPool 代理池 - 支持同类型多Agent实例
type AgentPool struct {
	mu           sync.Mutex
	workers      map[string][]*worker // 按Agent类型分组
	taskQueue    chan AgentTask
	resultQueue  chan AgentResult
	agentMgr     *AgentManager
	running      bool
	stateManager *TaskStateManager
}

type worker struct {
	id        int
	agentType string
	pool      *AgentPool
	stop      chan struct{}
	running   bool
}

func NewAgentPool(agentMgr *AgentManager) *AgentPool {
	pool := &AgentPool{
		taskQueue:    make(chan AgentTask, 100),
		resultQueue:  make(chan AgentResult, 100),
		agentMgr:     agentMgr,
		running:      true,
		workers:      make(map[string][]*worker),
		stateManager: NewTaskStateManager(),
	}

	go pool.dispatch()

	return pool
}

func (p *AgentPool) AddWorkers(agentType string, count int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, err := p.agentMgr.GetAgent(agentType); err != nil {
		return fmt.Errorf("agent type %s not registered", agentType)
	}

	for i := len(p.workers[agentType]); i < len(p.workers[agentType])+count; i++ {
		w := &worker{
			id:        i,
			agentType: agentType,
			pool:      p,
			stop:      make(chan struct{}),
		}
		p.workers[agentType] = append(p.workers[agentType], w)
		go w.run()
	}

	fmt.Printf("[AgentPool] Added %d workers for %s (total: %d)\n", count, agentType, len(p.workers[agentType]))
	return nil
}

func (w *worker) run() {
	w.running = true
	for {
		select {
		case task := <-w.pool.taskQueue:
			if task.AgentType == w.agentType {
				w.execute(task)
			}
		case <-w.stop:
			w.running = false
			return
		}
	}
}

func (w *worker) execute(task AgentTask) {
	agent, err := w.pool.agentMgr.GetAgent(task.AgentType)
	if err != nil {
		w.pool.resultQueue <- AgentResult{
			TaskID:    task.ID,
			Error:     fmt.Errorf("agent %s not found", task.AgentType),
			AgentID:   fmt.Sprintf("%s-%d", w.agentType, w.id),
			AgentType: w.agentType,
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		return
	}

	startTime := time.Now()
	output, err := agent.Execute(task.Context, task.Input)
	endTime := time.Now()

	w.pool.resultQueue <- AgentResult{
		TaskID:     task.ID,
		Output:     output,
		Error:      err,
		AgentID:    fmt.Sprintf("%s-%d", w.agentType, w.id),
		AgentType:  w.agentType,
		StartTime:  startTime,
		EndTime:    endTime,
	}
}

func (p *AgentPool) dispatch() {
	for p.running {
		select {
		case task := <-p.taskQueue:
			p.stateManager.UpdateTask(task.ID, TaskRunning, "", nil)
			go func(t AgentTask) {
				p.mu.Lock()
				workers, ok := p.workers[t.AgentType]
				p.mu.Unlock()

				if !ok || len(workers) == 0 {
					err := p.AddWorkers(t.AgentType, 3)
					if err != nil {
						p.resultQueue <- AgentResult{
							TaskID:    t.ID,
							Error:     err,
							AgentType: t.AgentType,
						}
						return
					}
				}
			}(task)
		}
	}
}

func (p *AgentPool) Submit(task AgentTask) {
	p.taskQueue <- task
}

func (p *AgentPool) GetResult() AgentResult {
	result := <-p.resultQueue
	if result.Error == nil {
		p.stateManager.UpdateTask(result.TaskID, TaskCompleted, result.Output, nil)
	} else {
		p.stateManager.UpdateTask(result.TaskID, TaskFailed, "", result.Error)
	}
	return result
}

func (p *AgentPool) ExecuteParallel(tasks []AgentTask) []AgentResult {
	var wg sync.WaitGroup
	results := make([]AgentResult, len(tasks))
	resultChan := make(chan AgentResult, len(tasks))

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t AgentTask) {
			defer wg.Done()
			p.Submit(t)
			result := p.GetResult()
			resultChan <- result
		}(i, task)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	i := 0
	for result := range resultChan {
		results[i] = result
		i++
	}

	return results
}

func (p *AgentPool) GetStateManager() *TaskStateManager {
	return p.stateManager
}

func (p *AgentPool) Stop() {
	p.running = false
	for _, workers := range p.workers {
		for _, w := range workers {
			close(w.stop)
		}
	}
	close(p.taskQueue)
	close(p.resultQueue)
}