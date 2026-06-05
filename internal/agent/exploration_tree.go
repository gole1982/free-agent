package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type NodeStatus int

const (
	NodePending NodeStatus = iota
	NodeRunning
	NodeSuccess
	NodeFailed
	NodeSkipped
)

type ExplorationNode struct {
	ID           string
	ParentID     string
	Status       NodeStatus
	Description  string
	Strategy     string
	Payload      map[string]interface{}
	Result       string
	Error        error
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Children     []*ExplorationNode
	Metadata     map[string]interface{}
}

type ExplorationTree struct {
	mu          sync.RWMutex
	root        *ExplorationNode
	nodeMap     map[string]*ExplorationNode
	currentNode *ExplorationNode
	history     []string
}

func NewExplorationTree(rootDescription string) *ExplorationTree {
	root := &ExplorationNode{
		ID:           "root",
		Status:       NodePending,
		Description:  rootDescription,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Children:     make([]*ExplorationNode, 0),
		Payload:      make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
	}

	return &ExplorationTree{
		root:        root,
		nodeMap:     map[string]*ExplorationNode{"root": root},
		currentNode: root,
		history:     []string{"root"},
	}
}

func (t *ExplorationTree) CreateChildNode(strategy, description string, payload map[string]interface{}) *ExplorationNode {
	t.mu.Lock()
	defer t.mu.Unlock()

	nodeID := fmt.Sprintf("%s-%d", t.currentNode.ID, len(t.currentNode.Children)+1)
	
	child := &ExplorationNode{
		ID:           nodeID,
		ParentID:     t.currentNode.ID,
		Status:       NodePending,
		Strategy:     strategy,
		Description:  description,
		Payload:      payload,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Children:     make([]*ExplorationNode, 0),
		Metadata:     make(map[string]interface{}),
	}

	t.currentNode.Children = append(t.currentNode.Children, child)
	t.nodeMap[nodeID] = child

	return child
}

func (t *ExplorationTree) MoveToNode(nodeID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	node, exists := t.nodeMap[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	t.currentNode = node
	t.history = append(t.history, nodeID)
	return nil
}

func (t *ExplorationTree) Backtrack() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.currentNode.ParentID == "" {
		return fmt.Errorf("already at root node, cannot backtrack further")
	}

	parent, exists := t.nodeMap[t.currentNode.ParentID]
	if !exists {
		return fmt.Errorf("parent node %s not found", t.currentNode.ParentID)
	}

	t.currentNode = parent
	t.history = append(t.history, fmt.Sprintf("backtrack-%s", parent.ID))
	return nil
}

func (t *ExplorationTree) BacktrackToBranchPoint() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.currentNode
	for current.ParentID != "" {
		parent := t.nodeMap[current.ParentID]
		if len(parent.Children) > 1 {
			t.currentNode = parent
			t.history = append(t.history, fmt.Sprintf("branch-%s", parent.ID))
			return nil
		}
		current = parent
	}

	return fmt.Errorf("no branch point found")
}

func (t *ExplorationTree) UpdateNodeStatus(nodeID string, status NodeStatus, result string, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	node, exists := t.nodeMap[nodeID]
	if !exists {
		return
	}

	node.Status = status
	node.Result = result
	node.Error = err
	node.UpdatedAt = time.Now()
}

func (t *ExplorationTree) UpdateCurrentNodeStatus(status NodeStatus, result string, err error) {
	t.UpdateNodeStatus(t.currentNode.ID, status, result, err)
}

func (t *ExplorationTree) GetCurrentNode() *ExplorationNode {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currentNode
}

func (t *ExplorationTree) GetNode(nodeID string) (*ExplorationNode, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	node, exists := t.nodeMap[nodeID]
	return node, exists
}

func (t *ExplorationTree) GetHistory() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	history := make([]string, len(t.history))
	copy(history, t.history)
	return history
}

func (t *ExplorationTree) GetSuccessNodes() []*ExplorationNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var successNodes []*ExplorationNode
	for _, node := range t.nodeMap {
		if node.Status == NodeSuccess {
			successNodes = append(successNodes, node)
		}
	}
	return successNodes
}

func (t *ExplorationTree) GetFailedNodes() []*ExplorationNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var failedNodes []*ExplorationNode
	for _, node := range t.nodeMap {
		if node.Status == NodeFailed {
			failedNodes = append(failedNodes, node)
		}
	}
	return failedNodes
}

func (t *ExplorationTree) GetBranchPoint(nodeID string) []*ExplorationNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var siblings []*ExplorationNode
	node, exists := t.nodeMap[nodeID]
	if !exists || node.ParentID == "" {
		return siblings
	}

	parent := t.nodeMap[node.ParentID]
	for _, child := range parent.Children {
		if child.ID != nodeID {
			siblings = append(siblings, child)
		}
	}
	return siblings
}

func (t *ExplorationTree) PrintTree() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("=== Exploration Tree ===\n")
	t.printNode(t.root, &sb, 0)
	sb.WriteString("\n=== History ===\n")
	for i, step := range t.history {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
	}
	return sb.String()
}

func (t *ExplorationTree) printNode(node *ExplorationNode, sb *strings.Builder, depth int) {
	prefix := strings.Repeat("  ", depth)
	statusStr := t.getStatusString(node.Status)
	sb.WriteString(fmt.Sprintf("%s%s [%s] %s\n", prefix, node.ID, statusStr, node.Description))
	
	if node.Error != nil {
		sb.WriteString(fmt.Sprintf("%s  Error: %v\n", prefix, node.Error))
	}
	if node.Result != "" {
		sb.WriteString(fmt.Sprintf("%s  Result: %s\n", prefix, node.Result))
	}
	
	for _, child := range node.Children {
		t.printNode(child, sb, depth+1)
	}
}

func (t *ExplorationTree) getStatusString(status NodeStatus) string {
	switch status {
	case NodePending:
		return "PENDING"
	case NodeRunning:
		return "RUNNING"
	case NodeSuccess:
		return "SUCCESS"
	case NodeFailed:
		return "FAILED"
	case NodeSkipped:
		return "SKIPPED"
	default:
		return "UNKNOWN"
	}
}