package agent

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// =============================================
// 问题 1：路由优先级完整映射
// T0 - 策略路由 (Policy Based Routing)
// T1 - 掩码长度 (Prefix Length)
// T2 - 管理距离 (Administrative Distance)
// T3 - 计算距离 (Metric)
// =============================================

// RoutePriorityTier 路由优先级层级
type RoutePriorityTier int

const (
	Tier0_Policy    RoutePriorityTier = 0  // 策略路由 - 最高优先级
	Tier1_Prefix    RoutePriorityTier = 1  // 掩码长度
	Tier2_AD        RoutePriorityTier = 2  // 管理距离
	Tier3_Metric    RoutePriorityTier = 3  // 计算距离
)

// AgentPolicy 策略路由策略（T0）
type AgentPolicy struct {
	PolicyID       string
	PolicyName     string
	Condition      func(task string) bool  // 策略条件
	TargetAgent    string
	Priority       int                      // 策略内部优先级
}

// AdministrativeDistance 管理距离（T2）- 类似思科 AD 值
type AdministrativeDistance int

const (
	AD_ManualOverride AdministrativeDistance = 1   // 手动指定
	AD_LearnedPerfect AdministrativeDistance = 5   // 完美学习的路由
	AD_DefaultSpecial AdministrativeDistance = 10  // 默认专用路由
	AD_LearnedGood    AdministrativeDistance = 20  // 学习的好路由
	AD_DefaultGeneral AdministrativeDistance = 50  // 默认通用路由
	AD_LastResort     AdministrativeDistance = 255 // 最后选择
)

// AgentMetrics Agent 完整度量值（T3）
type AgentMetrics struct {
	// 成功率相关
	SuccessRate         float64       // 历史成功率 (0-1)
	SuccessCount        int           // 成功次数
	FailureCount        int           // 失败次数
	RecentSuccessRate   float64       // 最近成功率（窗口最近10次）

	// 时间相关
	AvgExecutionTime    time.Duration // 平均执行时间
	P95ExecutionTime    time.Duration // 95分位执行时间
	TotalExecutionTime  time.Duration // 总执行时间

	// 质量相关
	AvgQualityScore     float64       // 平均质量评分 (0-1)
	QualityVariance     float64       // 质量方差（稳定性）

	// 效率相关
	ResourceEfficiency  float64       // 资源效率 (0-1)

	// 综合度量
	CompositeMetric     float64       // 综合度量值（越小越好）
}

// EnhancedAgentRoute 增强版的 Agent 路由
type EnhancedAgentRoute struct {
	ID               string
	Pattern          string
	PrefixLength     int
	AgentName        string
	
	// T0 - 策略路由
	PolicyID         string
	PolicyPriority   int
	
	// T2 - 管理距离
	AdminDistance    AdministrativeDistance
	
	// T3 - 度量值
	Metrics          *AgentMetrics
	
	// 路由状态
	Status           RouteStatus
	LastUsed         time.Time
	CreatedAt        time.Time
	HitCount         int64
}

// EnhancedRoutingTable 增强版路由表
type EnhancedRoutingTable struct {
	mu              sync.RWMutex
	routes          map[string]*EnhancedAgentRoute
	policies        []*AgentPolicy
	
	// Agent 特性库
	agentProfiles   map[string]*AgentProfile
	metricHistory   []MetricSnapshot
}

// AgentProfile Agent 完整特性档案
type AgentProfile struct {
	Name            string
	Description     string
	
	// 技能特性
	Skills          map[string]float64 // 技能名称 -> 熟练度 (0-1)
	
	// 性能特性
	DefaultAD       AdministrativeDistance
	BaseMetric      float64
	
	// 行为特性
	Reliability     float64  // 可靠性 (0-1)
	Creativity      float64  // 创造性 (0-1)
	Conservatism    float64  // 保守程度 (0-1)
	
	// 统计
	TotalInvocations int64
}

// MetricSnapshot 度量快照（用于历史分析）
type MetricSnapshot struct {
	Timestamp   time.Time
	RouteID     string
	AgentName   string
	Success     bool
	ExecutionTime time.Duration
}

// =============================================
// 创建增强路由表
// =============================================
func NewEnhancedRoutingTable() *EnhancedRoutingTable {
	ert := &EnhancedRoutingTable{
		routes:        make(map[string]*EnhancedAgentRoute),
		agentProfiles: make(map[string]*AgentProfile),
		policies:      make([]*AgentPolicy, 0),
	}
	
	// 初始化默认 Agent 特性档案
	ert.initDefaultProfiles()
	
	return ert
}

// 初始化默认 Agent 特性
func (ert *EnhancedRoutingTable) initDefaultProfiles() {
	// 代码生成
	ert.agentProfiles["Coder"] = &AgentProfile{
		Name:        "Coder",
		Description: "代码生成 Agent",
		Skills: map[string]float64{
			"coding": 1.0, "golang": 0.9, "web": 0.8,
		},
		DefaultAD:       AD_DefaultSpecial,
		BaseMetric:      50,
		Reliability:     0.85,
		Creativity:      0.7,
		Conservatism:    0.5,
	}
	
	// SQL 注入
	ert.agentProfiles["SQLiAgent"] = &AgentProfile{
		Name:        "SQLiAgent",
		Description: "SQL注入测试 Agent",
		Skills: map[string]float64{
			"security": 0.9, "sqli": 1.0, "pentesting": 0.9,
		},
		DefaultAD:       AD_DefaultSpecial,
		BaseMetric:      40,
		Reliability:     0.8,
		Creativity:      0.8,
		Conservatism:    0.3,
	}
	
	// XSS
	ert.agentProfiles["XSSAgent"] = &AgentProfile{
		Name:        "XSSAgent",
		Description: "XSS测试 Agent",
		Skills: map[string]float64{
			"security": 0.9, "xss": 1.0, "web": 0.8,
		},
		DefaultAD:       AD_DefaultSpecial,
		BaseMetric:      40,
		Reliability:     0.8,
		Creativity:      0.9,
		Conservatism:    0.3,
	}
	
	// Pentesting
	ert.agentProfiles["Pentesting"] = &AgentProfile{
		Name:        "Pentesting",
		Description: "综合渗透测试",
		Skills: map[string]float64{
			"security": 0.95, "pentesting": 0.9, "web": 0.7,
		},
		DefaultAD:       AD_DefaultGeneral,
		BaseMetric:      70,
		Reliability:     0.75,
		Creativity:      0.8,
		Conservatism:    0.4,
	}
	
	// Generic
	ert.agentProfiles["Generic"] = &AgentProfile{
		Name:        "Generic",
		Description: "通用 Agent",
		Skills: map[string]float64{
			"general": 0.8,
		},
		DefaultAD:       AD_LastResort,
		BaseMetric:      100,
		Reliability:     0.7,
		Creativity:      0.6,
		Conservatism:    0.5,
	}
}

// =============================================
// 注册策略路由（T0）
// =============================================
func (ert *EnhancedRoutingTable) RegisterPolicy(policy *AgentPolicy) {
	ert.mu.Lock()
	defer ert.mu.Unlock()
	ert.policies = append(ert.policies, policy)
	
	// 按优先级排序
	sort.Slice(ert.policies, func(i, j int) bool {
		return ert.policies[i].Priority < ert.policies[j].Priority
	})
}

// =============================================
// 注册路由
// =============================================
func (ert *EnhancedRoutingTable) RegisterRoute(
	pattern string,
	prefixLen int,
	agentName string,
	ad AdministrativeDistance,
) error {
	ert.mu.Lock()
	defer ert.mu.Unlock()
	
	// 验证 Agent 存在
	if _, exists := ert.agentProfiles[agentName]; !exists {
		return fmt.Errorf("agent profile not found: %s", agentName)
	}
	
	routeID := fmt.Sprintf("%s/%d", pattern, prefixLen)
	if _, exists := ert.routes[routeID]; exists {
		return fmt.Errorf("route already exists: %s", routeID)
	}
	
	profile := ert.agentProfiles[agentName]
	
	route := &EnhancedAgentRoute{
		ID:              routeID,
		Pattern:         pattern,
		PrefixLength:    prefixLen,
		AgentName:       agentName,
		AdminDistance:   ad,
		Metrics: &AgentMetrics{
			SuccessRate:      0.5,
			RecentSuccessRate: 0.5,
			CompositeMetric:  profile.BaseMetric,
		},
		Status:          RouteUp,
		CreatedAt:       time.Now(),
		LastUsed:        time.Now(),
	}
	
	ert.routes[routeID] = route
	fmt.Printf("[ROUTE TABLE] Registered: %s/%d → %s (AD:%d, BaseMetric:%.0f)\n",
		pattern, prefixLen, agentName, ad, profile.BaseMetric)
	
	return nil
}

// =============================================
// 问题 2：选择最佳 Agent - 完整的优先级算法
// T0 > T1 > T2 > T3
// =============================================
func (ert *EnhancedRoutingTable) SelectBestAgent(task string) (string, *EnhancedAgentRoute, error) {
	ert.mu.RLock()
	defer ert.mu.RUnlock()
	
	fmt.Printf("\n🔍 [ENHANCED ROUTING] Selecting best agent for: %s\n", task)
	
	// -----------------------------------------
	// T0: 策略路由 - 最高优先级
	// -----------------------------------------
	for _, policy := range ert.policies {
		if policy.Condition(task) {
			fmt.Printf("  ✅ [T0-POLICY] Match policy: %s → %s\n",
				policy.PolicyName, policy.TargetAgent)
			
			// 查找对应的路由
			for _, route := range ert.routes {
				if route.AgentName == policy.TargetAgent && route.Status == RouteUp {
					return route.AgentName, route, nil
				}
			}
		}
	}
	
	// -----------------------------------------
	// 收集所有匹配的路由
	// -----------------------------------------
	var candidates []*EnhancedAgentRoute
	taskLower := strings.ToLower(task)
	
	for _, route := range ert.routes {
		if route.Status != RouteUp {
			continue
		}
		
		patternLower := strings.ToLower(route.Pattern)
		
		// 匹配检查
		if route.Pattern == "" || // 默认路由
			strings.Contains(taskLower, patternLower) ||
			strings.HasPrefix(taskLower, patternLower) {
			candidates = append(candidates, route)
		}
	}
	
	if len(candidates) == 0 {
		return "", nil, fmt.Errorf("no available routes")
	}
	
	fmt.Printf("  📊 Found %d candidate routes\n", len(candidates))
	
	// -----------------------------------------
	// T1 > T2 > T3: 多层级排序
	// -----------------------------------------
	
	// 1. 按 T1: 掩码长度（降序）
	// 2. 按 T2: 管理距离（升序）
	// 3. 按 T3: 计算距离（升序）
	sort.Slice(candidates, func(i, j int) bool {
		ri, rj := candidates[i], candidates[j]
		
		// T1: 掩码长度优先
		if ri.PrefixLength != rj.PrefixLength {
			return ri.PrefixLength > rj.PrefixLength
		}
		
		// T2: 管理距离
		if ri.AdminDistance != rj.AdminDistance {
			return ri.AdminDistance < rj.AdminDistance
		}
		
		// T3: 综合度量
		return ert.calculateCompositeMetric(ri) < ert.calculateCompositeMetric(rj)
	})
	
	// 选择最佳的
	best := candidates[0]
	
	fmt.Printf("  🎯 Best Route: %s/%d → %s (AD:%d, Metric:%.2f)\n",
		best.Pattern, best.PrefixLength, best.AgentName,
		best.AdminDistance, ert.calculateCompositeMetric(best))
	
	// 打印前 3 个候选
	for i, r := range candidates[:ertMin(3, len(candidates))] {
		fmt.Printf("     #%d: %s/%d → %s (AD:%d, Metric:%.2f)\n",
			i+1, r.Pattern, r.PrefixLength, r.AgentName,
			r.AdminDistance, ert.calculateCompositeMetric(r))
	}
	
	return best.AgentName, best, nil
}

// =============================================
// T3: 计算综合度量值
// 考虑多个因素：成功率、执行时间、稳定性等
// =============================================
func (ert *EnhancedRoutingTable) calculateCompositeMetric(route *EnhancedAgentRoute) float64 {
	m := route.Metrics
	profile := ert.agentProfiles[route.AgentName]
	
	// 因子 1: 历史成功率 (权重: 40%)
	// 成功率越低，度量值越高（越差）
	successFactor := (1.0 - m.SuccessRate) * 100 * 0.4
	
	// 因子 2: 最近成功率 (权重: 30%)
	// 更看重近期表现
	recentSuccessFactor := (1.0 - m.RecentSuccessRate) * 100 * 0.3
	
	// 因子 3: 执行时间 (权重: 20%)
	timeFactor := 0.0
	if m.AvgExecutionTime > 0 {
		// 归一化：假设 10 秒 = 50 分
		normTime := math.Min(m.AvgExecutionTime.Seconds() / 10.0, 5.0)
		timeFactor = normTime * 20 * 0.2
	}
	
	// 因子 4: 稳定性/方差 (权重: 10%)
	stabilityFactor := m.QualityVariance * 100 * 0.1
	
	// 基础度量
	baseFactor := profile.BaseMetric * 0.1
	
	totalMetric := successFactor + recentSuccessFactor + timeFactor + stabilityFactor + baseFactor
	
	return math.Max(1, totalMetric) // 最小值为 1
}

// =============================================
// 问题 2 + 3: 记录执行结果 - 更新度量 + 学习触发
// =============================================
func (ert *EnhancedRoutingTable) RecordExecution(
	routeID string,
	agentName string,
	success bool,
	execTime time.Duration,
) {
	ert.mu.Lock()
	defer ert.mu.Unlock()
	
	// 记录历史快照
	ert.metricHistory = append(ert.metricHistory, MetricSnapshot{
		Timestamp:     time.Now(),
		RouteID:       routeID,
		AgentName:     agentName,
		Success:       success,
		ExecutionTime: execTime,
	})
	
	// 更新 Agent Profile 统计
	if profile, exists := ert.agentProfiles[agentName]; exists {
		profile.TotalInvocations++
	}
	
	// 更新路由统计
	if route, exists := ert.routes[routeID]; exists {
		m := route.Metrics
		
		// 更新基本统计
		if success {
			m.SuccessCount++
		} else {
			m.FailureCount++
		}
		
		// 更新总执行时间
		m.TotalExecutionTime += execTime
		total := m.SuccessCount + m.FailureCount
		if total > 0 {
			m.SuccessRate = float64(m.SuccessCount) / float64(total)
			m.AvgExecutionTime = m.TotalExecutionTime / time.Duration(total)
		}
		
		// 更新最近成功率（最近 10 次）
		m.RecentSuccessRate = ert.calculateRecentSuccess(agentName, 10)
		
		// 更新最后使用时间
		route.LastUsed = time.Now()
		route.HitCount++
		
		// 更新管理距离（动态学习）
		ert.updateAdministrativeDistance(route, total)
		
		fmt.Printf("  📊 Route update: %s/%d (success:%t, time:%v) | success:%.1f%%, recent:%.1f%%\n",
			route.Pattern, route.PrefixLength, success, execTime,
			m.SuccessRate*100, m.RecentSuccessRate*100)
			
		// 检查是否触发 问题 3: 特性修改或路由增删
		ert.evaluateRouteChange(route, total)
	}
}

// 计算最近成功率（滑动窗口）
func (ert *EnhancedRoutingTable) calculateRecentSuccess(agentName string, windowSize int) float64 {
	if len(ert.metricHistory) == 0 {
		return 0.5
	}
	
	startIdx := ertMax(0, len(ert.metricHistory)-windowSize)
	recent := ert.metricHistory[startIdx:]
	
	successCount := 0
	count := 0
	
	for _, snap := range recent {
		if snap.AgentName == agentName {
			count++
			if snap.Success {
				successCount++
			}
		}
	}
	
	if count == 0 {
		return 0.5
	}
	
	return float64(successCount) / float64(count)
}

// 更新管理距离（基于长期表现）
func (ert *EnhancedRoutingTable) updateAdministrativeDistance(route *EnhancedAgentRoute, totalExecutions int) {
	m := route.Metrics
	
	// 只有足够的执行次数才调整
	if totalExecutions < 5 {
		return
	}
	
	// 完美 -> 提升 AD
	if m.SuccessRate >= 0.95 {
		if route.AdminDistance > AD_LearnedPerfect {
			route.AdminDistance = AD_LearnedPerfect
			fmt.Printf("  ⬆️ AD improved: %d → %d (PERFECT)\n", 
				route.AdminDistance, AD_LearnedPerfect)
		}
	} else if m.SuccessRate >= 0.85 {
		if route.AdminDistance > AD_LearnedGood {
			route.AdminDistance = AD_LearnedGood
			fmt.Printf("  ⬆️ AD improved: %d → %d (GOOD)\n",
				route.AdminDistance, AD_LearnedGood)
		}
	}
	
	// 太差 -> 降低优先级
	if m.SuccessRate < 0.3 && totalExecutions > 10 {
		route.AdminDistance = AD_DefaultGeneral
		fmt.Printf("  ⬇️ AD degraded: %d → %d (POOR)\n", 
			route.AdminDistance, AD_DefaultGeneral)
	}
}

// =============================================
// 问题 3: 何时增删路由、修改特性
// =============================================
func (ert *EnhancedRoutingTable) evaluateRouteChange(route *EnhancedAgentRoute, totalExecutions int) {
	m := route.Metrics
	
	// -----------------------------------------
	// 删除条件：长期表现极差（且有替代）
	// -----------------------------------------
	if m.SuccessRate < 0.3 && totalExecutions > 20 {
		fmt.Printf("  ⚠️ ROUTE CONSIDERING DELETION: %s/%d (Success:%.1f%%)\n",
			route.Pattern, route.PrefixLength, m.SuccessRate*100)
	}
	
	// -----------------------------------------
	// 修改特性：根据成功率调整 Agent Profile
	// -----------------------------------------
	if totalExecutions % 5 == 0 && totalExecutions > 0 {
		ert.adjustAgentProfile(route.AgentName, m.SuccessRate)
	}
	
	// -----------------------------------------
	// 新增路由：当发现新的成功模式
	// -----------------------------------------
	// (在实际系统中，这部分可能由 Intent 或 Planner Agent 触发)
}

// 调整 Agent 特性
func (ert *EnhancedRoutingTable) adjustAgentProfile(agentName string, recentSuccess float64) {
	profile, exists := ert.agentProfiles[agentName]
	if !exists {
		return
	}
	
	// 学习率
	learnRate := 0.05
	
	if recentSuccess > 0.8 {
		// 成功 → 提升可靠性
		profile.Reliability = math.Min(1.0, profile.Reliability + learnRate)
		fmt.Printf("  📈 Profile adjust: %s Reliability → %.2f\n",
			agentName, profile.Reliability)
	} else if recentSuccess < 0.4 {
		// 失败 → 降低可靠性
		profile.Reliability = math.Max(0.1, profile.Reliability - learnRate)
		fmt.Printf("  📉 Profile adjust: %s Reliability → %.2f\n",
			agentName, profile.Reliability)
	}
}

// =============================================
// 获取完整路由表状态
// =============================================
func (ert *EnhancedRoutingTable) GetStatusString() string {
	ert.mu.RLock()
	defer ert.mu.RUnlock()
	
	var sb strings.Builder
	
	sb.WriteString(strings.Repeat("═", 110) + "\n")
	sb.WriteString(fmt.Sprintf(
		"%-20s %-15s %-4s %-4s %-8s %-8s %-8s %-10s\n",
		"PATTERN", "AGENT", "T1-P", "T2-AD", "T3-METRIC", "SUCCESS", "RECENT", "STATUS"))
	sb.WriteString(strings.Repeat("─", 110) + "\n")
	
	for _, route := range ert.routes {
		statusStr := "UP"
		if route.Status == RouteDown {
			statusStr = "DOWN"
		}
		
		sb.WriteString(fmt.Sprintf(
			"%-20s %-15s %-4d %-4d %-8.1f %-8.1f%% %-8.1f%% %-10s\n",
			fmt.Sprintf("%s/%d", route.Pattern, route.PrefixLength),
			route.AgentName,
			route.PrefixLength,
			route.AdminDistance,
			ert.calculateCompositeMetric(route),
			route.Metrics.SuccessRate*100,
			route.Metrics.RecentSuccessRate*100,
			statusStr,
		))
	}
	
	return sb.String()
}

func ertMin(a, b int) int {
	if a < b { return a }
	return b
}

func ertMax(a, b int) int {
	if a > b { return a }
	return b
}
