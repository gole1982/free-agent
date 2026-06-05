package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ===================================================
// Core Concepts: State, Action, Reward, Policy
// ===================================================

// State represents the current state of the environment
type State struct {
	TargetURL        string            `json:"target_url"`
	CurrentAction    string            `json:"current_action"`
	LastResult       string            `json:"last_result"`
	DiscoveredVulns  int               `json:"discovered_vulns"`
	FailedAttempts   int               `json:"failed_attempts"`
	TimeElapsed      int               `json:"time_elapsed"`
	Context          map[string]string `json:"context"`
}

// Action represents an action the agent can take
type Action struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Reward represents the feedback from taking an action
type Reward struct {
	Value     float64 `json:"value"`
	Reason    string  `json:"reason"`
	Timestamp string  `json:"timestamp"`
}

// Experience is a full (S, A, R, S') tuple
type Experience struct {
	State      State   `json:"state"`
	Action     Action  `json:"action"`
	Reward     Reward  `json:"reward"`
	NextState  State   `json:"next_state"`
	Timestamp  string  `json:"timestamp"`
}

// ===================================================
// Reinforcement Learning Components
// ===================================================

type QTable map[string]map[string]float64

type LearningAgent struct {
	Name          string
	QTable        QTable
	LearningRate  float64
	DiscountFactor float64
	Epsilon       float64
	ExperienceLog []Experience
	Stats         LearningStats
	mu            sync.RWMutex
}

type LearningStats struct {
	TotalEpisodes   int
	TotalSteps      int
	TotalRewards    float64
	BestReward      float64
	SuccessRate     float64
	LastUpdated     time.Time
}

func NewLearningAgent(name string) *LearningAgent {
	return &LearningAgent{
		Name:          name,
		QTable:        make(QTable),
		LearningRate:  0.1,
		DiscountFactor: 0.95,
		Epsilon:       0.3, // 30% exploration, 70% exploitation
		ExperienceLog: make([]Experience, 0),
		Stats: LearningStats{
			LastUpdated: time.Now(),
		},
	}
}

func (la *LearningAgent) GetStateKey(s State) string {
	// Simplified state representation for Q-table
	return fmt.Sprintf("%s-%d-%d", s.TargetURL, s.DiscoveredVulns, s.FailedAttempts)
}

func (la *LearningAgent) SelectAction(s State, availableActions []Action) Action {
	la.mu.RLock()
	defer la.mu.RUnlock()

	// Epsilon-greedy strategy
	if rand.Float64() < la.Epsilon {
		// Exploration: pick random action
		fmt.Printf("🧪 [Exploration] Trying random action (epsilon: %.2f)\n", la.Epsilon)
		return availableActions[rand.Intn(len(availableActions))]
	}

	// Exploitation: pick best known action
	stateKey := la.GetStateKey(s)
	actionValues := la.QTable[stateKey]

	if len(actionValues) == 0 {
		fmt.Printf("📊 No Q-values for this state, falling back to random\n")
		return availableActions[rand.Intn(len(availableActions))]
	}

	// Find action with highest Q-value
	bestAction := availableActions[0]
	bestValue := -999.9

	for _, action := range availableActions {
		if val, exists := actionValues[action.Name]; exists && val > bestValue {
			bestValue = val
			bestAction = action
		}
	}

	fmt.Printf("🎯 [Exploitation] Using best action: %s (Q-value: %.3f)\n", bestAction.Name, bestValue)
	return bestAction
}

func (la *LearningAgent) Learn(experience Experience) {
	la.mu.Lock()
	defer la.mu.Unlock()

	// Store experience
	la.ExperienceLog = append(la.ExperienceLog, experience)

	// Update Q-table using Q-learning algorithm
	stateKey := la.GetStateKey(experience.State)
	nextStateKey := la.GetStateKey(experience.NextState)

	if la.QTable[stateKey] == nil {
		la.QTable[stateKey] = make(map[string]float64)
	}

	// Current Q-value
	currentQ := la.QTable[stateKey][experience.Action.Name]

	// Best next Q-value
	maxNextQ := 0.0
	if nextActions, exists := la.QTable[nextStateKey]; exists {
		for _, v := range nextActions {
			if v > maxNextQ {
				maxNextQ = v
			}
		}
	}

	// Q-learning update rule
	newQ := currentQ + la.LearningRate * (experience.Reward.Value + la.DiscountFactor * maxNextQ - currentQ)
	la.QTable[stateKey][experience.Action.Name] = newQ

	// Update stats
	la.Stats.TotalSteps++
	la.Stats.TotalRewards += experience.Reward.Value
	if experience.Reward.Value > la.Stats.BestReward {
		la.Stats.BestReward = experience.Reward.Value
	}
	la.Stats.LastUpdated = time.Now()

	// Decay epsilon over time for less exploration, more exploitation
	if la.Epsilon > 0.05 {
		la.Epsilon *= 0.999
	}

	fmt.Printf("📚 Learning: Q(%s, %s) = %.3f -> %.3f\n",
		stateKey, experience.Action.Name, currentQ, newQ)
}

func (la *LearningAgent) SaveKnowledge(filename string) error {
	la.mu.RLock()
	defer la.mu.RUnlock()

	data := map[string]interface{}{
		"q_table":        la.QTable,
		"experience_log": la.ExperienceLog,
		"stats":          la.Stats,
		"epsilon":        la.Epsilon,
		"timestamp":      time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

func (la *LearningAgent) LoadKnowledge(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil // No knowledge yet, that's okay
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var loaded struct {
		QTable        QTable     `json:"q_table"`
		ExperienceLog []Experience `json:"experience_log"`
		Stats         LearningStats `json:"stats"`
		Epsilon       float64     `json:"epsilon"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	la.mu.Lock()
	defer la.mu.Unlock()

	la.QTable = loaded.QTable
	la.ExperienceLog = loaded.ExperienceLog
	la.Stats = loaded.Stats
	la.Epsilon = loaded.Epsilon

	return nil
}

func (la *LearningAgent) PrintStats() {
	la.mu.RLock()
	defer la.mu.RUnlock()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 LEARNING AGENT STATS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Episodes:    %d\n", la.Stats.TotalEpisodes)
	fmt.Printf("Total Steps:       %d\n", la.Stats.TotalSteps)
	fmt.Printf("Total Rewards:     %.2f\n", la.Stats.TotalRewards)
	fmt.Printf("Best Reward:       %.2f\n", la.Stats.BestReward)
	fmt.Printf("Epsilon:           %.4f\n", la.Epsilon)
	fmt.Printf("States learned:    %d\n", len(la.QTable))
	fmt.Printf("Experience count:  %d\n", len(la.ExperienceLog))
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

func simpleScrapeHTTP(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}