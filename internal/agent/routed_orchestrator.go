package agent

import (
	"context"
	"fmt"
	"time"
)

// RoutedOrchestrator - 基于路由表的增强型Orchestrator
type RoutedOrchestrator struct {
	name        string
	agentMgr    *AgentManager
	routingTable *RoutingTable
}

func NewRoutedOrchestrator(agentMgr *AgentManager) *RoutedOrchestrator {
	ro := &RoutedOrchestrator{
		name:        "RoutedOrchestrator",
		agentMgr:    agentMgr,
		routingTable: NewRoutingTable(),
	}

	// 初始化默认路由表
	ro.initializeDefaultRoutes()

	return ro
}

// 初始化默认路由表（建立像IP路由一样的层次结构）
func (ro *RoutedOrchestrator) initializeDefaultRoutes() {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║            INITIALIZING AGENT ROUTING TABLE                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

	// 层次1: 默认路由 (0.0.0.0/0)
	// 用途: 所有不匹配的任务都走这个
	ro.routingTable.RegisterAgentRoute("", 0, "Generic", 100.0)
	fmt.Println("  [0/0] Default → Generic Agent")

	// 层次2: 大类路由 (类似 /8)
	// 用途: 确定是代码任务还是安全任务
	ro.routingTable.RegisterAgentRoute("code", 8, "Coder", 80.0)
	ro.routingTable.RegisterAgentRoute("create", 8, "Coder", 80.0)
	ro.routingTable.RegisterAgentRoute("安全", 8, "Pentesting", 80.0)
	ro.routingTable.RegisterAgentRoute("security", 8, "Pentesting", 80.0)
	ro.routingTable.RegisterAgentRoute("测试", 8, "Tester", 80.0)
	ro.routingTable.RegisterAgentRoute("test", 8, "Tester", 80.0)
	fmt.Println("  [/8] 类别路由已建立")

	// 层次3: 更细分的路由 (类似 /16)
	ro.routingTable.RegisterAgentRoute("sql", 16, "SQLiAgent", 60.0)
	ro.routingTable.RegisterAgentRoute("SQL", 16, "SQLiAgent", 60.0)
	ro.routingTable.RegisterAgentRoute("xss", 16, "XSSAgent", 60.0)
	ro.routingTable.RegisterAgentRoute("XSS", 16, "XSSAgent", 60.0)
	ro.routingTable.RegisterAgentRoute("命令注入", 16, "CommandInjectAgent", 60.0)
	ro.routingTable.RegisterAgentRoute("cmd inject", 16, "CommandInjectAgent", 60.0)
	ro.routingTable.RegisterAgentRoute("CTF", 16, "CTFExploration", 60.0)
	ro.routingTable.RegisterAgentRoute("ctf", 16, "CTFExploration", 60.0)
	fmt.Println("  [/16] 细分路由已建立")

	// 层次4: 最具体的路由 (类似 /24)
	ro.routingTable.RegisterAgentRoute("SQL注入", 24, "SQLiAgent", 40.0)
	ro.routingTable.RegisterAgentRoute("XSS攻击", 24, "XSSAgent", 40.0)
	ro.routingTable.RegisterAgentRoute("XSS漏洞", 24, "XSSAgent", 40.0)
	fmt.Println("  [/24] 精确路由已建立")

	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║          ROUTING TABLE INITIALIZED SUCCESSFULLY               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝\n")
}

func (ro *RoutedOrchestrator) Name() string {
	return ro.name
}

func (ro *RoutedOrchestrator) Description() string {
	return "增强型基于路由表的任务协调器，支持最长匹配优先、动态度量、回退降级机制"
}

// Execute - 执行任务（使用路由表选择）
func (ro *RoutedOrchestrator) Execute(ctx context.Context, task string) (string, error) {
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🚀 [Routed Orchestrator] 收到任务: %s\n", task)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	startTime := time.Now()
	var finalResult string
	var err error

	// 选择的路线，用于重试和回退
	var triedAgents []string
	var currentAttempt int = 0
	maxAttempts := 3

	for currentAttempt < maxAttempts {
		currentAttempt++
		
		fmt.Printf("\n🔄 [尝试 %d/%d] 正在查找最佳路由...\n", currentAttempt, maxAttempts)

		// 使用路由表选择最佳Agent
		selectedAgent, selectErr := ro.routingTable.SelectAgent(task)
		
		// 如果选择失败或者这个Agent已经尝试过了
		if selectErr != nil {
			fmt.Printf("  ⚠️ 无法选择Agent: %v\n", selectErr)
			// 降级：直接使用Generic
			selectedAgent = "Generic"
		}

		// 检查是否已经试过
		alreadyTried := false
		for _, tried := range triedAgents {
			if tried == selectedAgent {
				alreadyTried = true
				break
			}
		}

		if alreadyTried {
			fmt.Printf("  ⏭️ Agent %s 已尝试过，尝试降级...\n", selectedAgent)
			
			// 降级：选择一个更通用的Agent
			selectedAgent = ro.getFallbackAgent(triedAgents)
			if selectedAgent == "" {
				fmt.Printf("  😢 没有更多可用的Agent，放弃\n")
				break
			}
		}

		fmt.Printf("  📡 选择Agent: %s\n", selectedAgent)
		triedAgents = append(triedAgents, selectedAgent)

		// 执行
		agent, getErr := ro.agentMgr.GetAgent(selectedAgent)
		if getErr != nil {
			fmt.Printf("  ❌ 无法获取Agent: %v\n", getErr)
			
			// 记录失败
			ro.routingTable.RecordRouteResult(task, selectedAgent, false, time.Since(startTime))
			
			continue
		}

		// 执行任务
		execStartTime := time.Now()
		result, execErr := agent.Execute(ctx, task)
		execDuration := time.Since(execStartTime)

		if execErr != nil {
			fmt.Printf("  ❌ Agent执行失败: %v\n", execErr)
			
			// 记录失败
			ro.routingTable.RecordRouteResult(task, selectedAgent, false, execDuration)
			
			// 继续尝试
			continue
		}

		// 成功！
		fmt.Printf("  ✅ Agent %s 执行成功! (耗时: %v)\n", selectedAgent, execDuration)
		
		// 记录成功
		ro.routingTable.RecordRouteResult(task, selectedAgent, true, execDuration)
		
		finalResult = result
		err = nil
		break
	}

	if finalResult == "" {
		// 所有尝试都失败，使用最后的办法：用Generic Agent
		fmt.Printf("\n⚠️ 所有Agent都失败了，最后尝试默认路由...\n")
		
		agent, _ := ro.agentMgr.GetAgent("Generic")
		if agent != nil {
			finalResult, err = agent.Execute(ctx, task)
		} else {
			err = fmt.Errorf("所有路由都失败，没有可用的Agent")
		}
	}

	// 打印路由表状态
	fmt.Printf("\n" + ro.routingTable.GetRoutingTableStatus())

	return finalResult, err
}

// 获取回退Agent（降级策略）
func (ro *RoutedOrchestrator) getFallbackAgent(excluded []string) string {
	// 优先级顺序:
	// 1. Generic (默认)
	// 2. 看看是否有其他合适的但没被排除的
	allAgents := ro.agentMgr.ListAgents()
	
	for _, agent := range allAgents {
		name := agent.Name()
		
		// 排除已试过的
		tried := false
		for _, e := range excluded {
			if e == name {
				tried = true
				break
			}
		}
		if tried {
			continue
		}

		// 只返回非特定的、更通用的
		if name == "Generic" || name == "CTFExploration" || name == "Pentesting" {
			return name
		}
	}

	return "Generic"
}

// GetRoutingTable - 获取路由表（用于外部查看/调试）
func (ro *RoutedOrchestrator) GetRoutingTable() *RoutingTable {
	return ro.routingTable
}
