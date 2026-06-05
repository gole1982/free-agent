package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// RouteStatus represents the status of a route/agent
type RouteStatus int

const (
	RouteUp RouteStatus = iota   // 路由正常
	RouteDown              // 路由失效
	RouteStandby           // 备用路由
)

// AgentRoute represents a single "route" in our routing table
type AgentRoute struct {
	ID            string        // 路由ID
	Pattern       string        // 匹配模式（类似IP前缀）
	PrefixLength  int           // 前缀长度（类似CIDR，越长越精确）
	AgentName     string        // 对应的Agent名称
	Metric        float64       // 路由度量（越低越好，类似路由优先级）
	SuccessCount  int           // 成功次数
	FailureCount  int           // 失败次数
	SuccessRate   float64       // 成功率
	TotalTime     time.Duration // 总执行时间
	AvgTime       time.Duration // 平均执行时间
	Status        RouteStatus   // 路由状态
	LastUsed      time.Time     // 最后使用时间
	CreatedAt     time.Time     // 创建时间
}

// RoutingTable - 路由表式的Agent选择系统
type RoutingTable struct {
	mu               sync.RWMutex
	routes           map[string]*AgentRoute        // 路由表
	defaultRoute     *AgentRoute                   // 默认路由（类似0.0.0.0/0）
	routeByPattern   map[string][]*AgentRoute      // 按模式索引
	agentHealth      map[string]*AgentHealth       // Agent健康状态
}

// AgentHealth - Agent健康状态
type AgentHealth struct {
	Name           string
	CurrentStatus  RouteStatus
	ConsecutiveFails int
	TotalFails      int
	TotalExecutions int
	LastFailure    time.Time
}

// NewRoutingTable - 创建新的路由表
func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		routes:         make(map[string]*AgentRoute),
		routeByPattern: make(map[string][]*AgentRoute),
		agentHealth:    make(map[string]*AgentHealth),
	}
}

// RegisterAgentRoute - 注册一个Agent路由
func (rt *RoutingTable) RegisterAgentRoute(pattern string, prefixLength int, agentName string, initialMetric float64) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// 创建路由ID
	routeID := fmt.Sprintf("%s/%d", pattern, prefixLength)

	// 检查是否已存在
	if _, exists := rt.routes[routeID]; exists {
		return fmt.Errorf("route already exists: %s", routeID)
	}

	// 创建新路由
	route := &AgentRoute{
		ID:           routeID,
		Pattern:      pattern,
		PrefixLength: prefixLength,
		AgentName:    agentName,
		Metric:       initialMetric,
		Status:       RouteUp,
		CreatedAt:    time.Now(),
		SuccessRate:  0.5, // 初始成功率50%
	}

	rt.routes[routeID] = route
	rt.routeByPattern[pattern] = append(rt.routeByPattern[pattern], route)

	// 初始化Agent健康状态
	if _, exists := rt.agentHealth[agentName]; !exists {
		rt.agentHealth[agentName] = &AgentHealth{
			Name:          agentName,
			CurrentStatus: RouteUp,
		}
	}

	// 设置默认路由（prefix长度为0的）
	if prefixLength == 0 {
		rt.defaultRoute = route
	}

	return nil
}

// SelectAgent - 根据任务选择最佳Agent（路由查找）
func (rt *RoutingTable) SelectAgent(task string) (string, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	fmt.Printf("\n🔍 [Routing Table] 查找路由: %s\n", task)

	// 找出所有匹配的路由
	var matchingRoutes []*AgentRoute
	taskLower := strings.ToLower(task)

	for _, route := range rt.routes {
		if route.Status != RouteUp {
			continue // 跳过失效的路由
		}

		patternLower := strings.ToLower(route.Pattern)
		
		// 前缀匹配
		if strings.HasPrefix(taskLower, patternLower) || 
		   strings.Contains(taskLower, patternLower) {
			matchingRoutes = append(matchingRoutes, route)
		}
	}

	// 按前缀长度排序（最长匹配优先）
	sortByPrefixLength(matchingRoutes)

	if len(matchingRoutes) > 0 {
		fmt.Printf("  ✓ 找到 %d 条匹配路由\n", len(matchingRoutes))
		
		// 在匹配的路由中选择度量最好的
		bestRoute := matchingRoutes[0]
		for _, route := range matchingRoutes[1:] {
			// 比较成功率和度量
			if (route.SuccessRate > bestRoute.SuccessRate) || 
			   (route.SuccessRate == bestRoute.SuccessRate && route.Metric < bestRoute.Metric) {
				bestRoute = route
			}
		}

		fmt.Printf("  ✨ 最佳路由: %s → Agent: %s (成功率: %.1f%%, 度量: %.2f)\n", 
			bestRoute.Pattern, bestRoute.AgentName, bestRoute.SuccessRate*100, bestRoute.Metric)
		
		return bestRoute.AgentName, nil
	}

	// 没有匹配的，使用默认路由
	if rt.defaultRoute != nil && rt.defaultRoute.Status == RouteUp {
		fmt.Printf("  ➡️ 使用默认路由 → Agent: %s\n", rt.defaultRoute.AgentName)
		return rt.defaultRoute.AgentName, nil
	}

	return "", fmt.Errorf("no available routes found")
}

// RecordRouteResult - 记录路由执行结果（类似路由更新）
func (rt *RoutingTable) RecordRouteResult(pattern string, agentName string, success bool, duration time.Duration) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// 找到匹配的路由
	var targetRoute *AgentRoute
	for _, route := range rt.routes {
		if route.Pattern == pattern || route.AgentName == agentName {
			targetRoute = route
			break
		}
	}

	if targetRoute == nil {
		// 尝试模糊查找
		for _, route := range rt.routes {
			if strings.Contains(pattern, route.Pattern) {
				targetRoute = route
				break
			}
		}
	}

	if targetRoute == nil {
		return fmt.Errorf("route not found for %s", pattern)
	}

	// 更新统计
	targetRoute.TotalTime += duration
	targetRoute.LastUsed = time.Now()

	if success {
		targetRoute.SuccessCount++
		if health, exists := rt.agentHealth[agentName]; exists {
			health.ConsecutiveFails = 0
		}
	} else {
		targetRoute.FailureCount++
		if health, exists := rt.agentHealth[agentName]; exists {
			health.ConsecutiveFails++
			health.TotalFails++
			health.LastFailure = time.Now()
		}
	}

	// 更新成功率
	total := targetRoute.SuccessCount + targetRoute.FailureCount
	if total > 0 {
		targetRoute.SuccessRate = float64(targetRoute.SuccessCount) / float64(total)
		targetRoute.AvgTime = targetRoute.TotalTime / time.Duration(total)
	}

	// 更新Agent健康状态
	health := rt.agentHealth[agentName]
	health.TotalExecutions++

	// 检查是否需要标记为down（连续失败）
	if health.ConsecutiveFails >= 3 {
		fmt.Printf("⚠️  Agent %s 连续失败 %d 次，标记为失效\n", agentName, health.ConsecutiveFails)
		health.CurrentStatus = RouteDown
		targetRoute.Status = RouteDown
	}

	// 更新度量（动态调整）
	// 考虑: 成功率、执行时间、最近使用
	targetRoute.Metric = calculateMetric(targetRoute)

	fmt.Printf("📊 路由更新: %s/%d → %s [成功:%d, 失败:%d, 成功率:%.1f%%, 度量:%.2f]\n",
		targetRoute.Pattern, targetRoute.PrefixLength, targetRoute.AgentName,
		targetRoute.SuccessCount, targetRoute.FailureCount, targetRoute.SuccessRate*100, targetRoute.Metric)

	return nil
}

// RecoverAgent - 恢复一个失效的Agent
func (rt *RoutingTable) RecoverAgent(agentName string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	if health, exists := rt.agentHealth[agentName]; exists {
		health.ConsecutiveFails = 0
		health.CurrentStatus = RouteUp
		
		// 恢复相关的路由
		for _, route := range rt.routes {
			if route.AgentName == agentName && route.Status == RouteDown {
				route.Status = RouteUp
			}
		}
		fmt.Printf("✅ Agent %s 已恢复\n", agentName)
	}
}

// GetRoutingTableStatus - 获取路由表状态
func (rt *RoutingTable) GetRoutingTableStatus() string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("                        AGENT ROUTING TABLE                         \n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("%-25s %-15s %-10s %-8s %-10s %-8s\n",
		"Pattern/Prefix", "Agent", "Status", "Metric", "Success", "Uses"))
	sb.WriteString(strings.Repeat("─", 80) + "\n")

	for _, route := range rt.routes {
		statusStr := "UP"
		if route.Status == RouteDown {
			statusStr = "DOWN"
		} else if route.Status == RouteStandby {
			statusStr = "STANDBY"
		}

		sb.WriteString(fmt.Sprintf("%-25s %-15s %-10s %-8.2f %-10.1f%% %-8d\n",
			fmt.Sprintf("%s/%d", route.Pattern, route.PrefixLength),
			route.AgentName,
			statusStr,
			route.Metric,
			route.SuccessRate*100,
			route.SuccessCount+route.FailureCount))
	}

	sb.WriteString("\n═══════════════════════════════════════════════════════════════════\n")
	sb.WriteString("                        AGENT HEALTH STATUS                         \n")
	sb.WriteString("═══════════════════════════════════════════════════════════════════\n")

	for _, health := range rt.agentHealth {
		statusStr := "HEALTHY"
		if health.CurrentStatus == RouteDown {
			statusStr = "DEGRADED"
		}
		sb.WriteString(fmt.Sprintf("%-20s %-12s ConsecFails: %d, Total: %d\n",
			health.Name, statusStr, health.ConsecutiveFails, health.TotalExecutions))
	}

	return sb.String()
}

// 辅助函数

// 计算度量值（越小越好）
func calculateMetric(route *AgentRoute) float64 {
	// 考虑因素:
	// 1. 成功率的倒数
	// 2. 平均执行时间
	// 3. 最近使用（有衰减）

	successFactor := (1.0 - route.SuccessRate) * 100.0

	// 归一化时间（假设10秒是基准）
	timeFactor := 0.0
	if route.AvgTime > 0 {
		timeFactor = route.AvgTime.Seconds() / 10.0
	}

	// 综合度量
	return successFactor*0.6 + timeFactor*0.3 + float64(route.PrefixLength)*0.1
}

func sortByPrefixLength(routes []*AgentRoute) {
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[i].PrefixLength < routes[j].PrefixLength {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}
}
