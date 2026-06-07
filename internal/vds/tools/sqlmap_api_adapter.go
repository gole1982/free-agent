package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/vibe-coding/free-agent/internal/vds"
)

// SqlmapAdapter sqlmap REST API 适配器
// 通过 sqlmapapi.py -s -p 8775 启动的 REST API 进行 SQL 注入扫描
type SqlmapAdapter struct {
	*APIAdapter
}

// NewSqlmapAdapter 创建 sqlmap 适配器（默认 http://127.0.0.1:8775）
func NewSqlmapAdapter(baseURL string) *SqlmapAdapter {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8775"
	}
	return &SqlmapAdapter{
		APIAdapter: NewAPIAdapter(baseURL, ""),
	}
}

func (s *SqlmapAdapter) ToolName() string    { return "sqlmap" }
func (s *SqlmapAdapter) AdapterType() string  { return "api" }
func (s *SqlmapAdapter) IsAvailable(ctx context.Context) bool {
	return s.CheckAvailability(ctx, "/")
}

// --- sqlmap API 数据模型 ---

type sqlmapTaskResponse struct {
	Success bool   `json:"success"`
	TaskID  string `json:"taskid"`
}

type sqlmapStatusResponse struct {
	Success bool   `json:"success"`
	Status  string `json:"status"` // "running" | "terminated"
}

type sqlmapDataEntry struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Parameter string `json:"parameter"`
	Place     string `json:"place"`
	Payload   string `json:"payload"`
	DBMS      string `json:"dbms,omitempty"`
	Section   string `json:"section,omitempty"`
}

type sqlmapDataResponse struct {
	Success bool              `json:"success"`
	Data    []sqlmapDataEntry `json:"data"`
	Error   []string          `json:"error"`
}

// NewTask 创建新扫描任务
func (s *SqlmapAdapter) NewTask(ctx context.Context) (string, error) {
	var resp sqlmapTaskResponse
	if err := s.Get(ctx, "/task/new", &resp); err != nil {
		return "", fmt.Errorf("sqlmap new task: %w", err)
	}
	if !resp.Success {
		return "", fmt.Errorf("sqlmap new task failed")
	}
	return resp.TaskID, nil
}

// SetOption 设置扫描选项（如目标 URL）
func (s *SqlmapAdapter) SetOption(ctx context.Context, taskID string, options map[string]interface{}) error {
	var resp struct {
		Success bool `json:"success"`
	}
	if err := s.Post(ctx, fmt.Sprintf("/option/%s/set", taskID), options, &resp); err != nil {
		return fmt.Errorf("sqlmap set option: %w", err)
	}
	return nil
}

// StartScan 启动扫描
func (s *SqlmapAdapter) StartScan(ctx context.Context, taskID string) error {
	var resp struct {
		Success bool   `json:"success"`
		EngineID int   `json:"engineid"`
	}
	if err := s.Get(ctx, fmt.Sprintf("/scan/%s/start", taskID), &resp); err != nil {
		return fmt.Errorf("sqlmap start scan: %w", err)
	}
	return nil
}

// GetStatus 获取扫描状态
func (s *SqlmapAdapter) GetStatus(ctx context.Context, taskID string) (string, error) {
	var resp sqlmapStatusResponse
	if err := s.Get(ctx, fmt.Sprintf("/scan/%s/status", taskID), &resp); err != nil {
		return "", fmt.Errorf("sqlmap get status: %w", err)
	}
	return resp.Status, nil
}

// GetData 获取扫描结果
func (s *SqlmapAdapter) GetData(ctx context.Context, taskID string) ([]sqlmapDataEntry, error) {
	var resp sqlmapDataResponse
	if err := s.Get(ctx, fmt.Sprintf("/scan/%s/data", taskID), &resp); err != nil {
		return nil, fmt.Errorf("sqlmap get data: %w", err)
	}
	return resp.Data, nil
}

// DeleteTask 删除任务
func (s *SqlmapAdapter) DeleteTask(ctx context.Context, taskID string) error {
	var resp struct {
		Success bool `json:"success"`
	}
	return s.Get(ctx, fmt.Sprintf("/task/%s/delete", taskID), &resp)
}

// ScanTarget 完整扫描流程：创建任务 → 设置目标 → 启动 → 等待 → 获取结果
func (s *SqlmapAdapter) ScanTarget(ctx context.Context, targetURL string) ([]vds.VulnerabilityFinding, error) {
	// 1. 创建任务
	taskID, err := s.NewTask(ctx)
	if err != nil {
		return nil, err
	}
	defer s.DeleteTask(ctx, taskID)

	// 2. 设置目标
	if err := s.SetOption(ctx, taskID, map[string]interface{}{
		"url": targetURL,
	}); err != nil {
		return nil, err
	}

	// 3. 启动扫描
	if err := s.StartScan(ctx, taskID); err != nil {
		return nil, err
	}

	// 4. 轮询等待完成
	scanCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	var status string
	for {
		select {
		case <-scanCtx.Done():
			return nil, fmt.Errorf("sqlmap scan timeout: %w", scanCtx.Err())
		default:
			time.Sleep(3 * time.Second)
			status, err = s.GetStatus(ctx, taskID)
			if err != nil {
				return nil, err
			}
			if status == "terminated" {
				goto done
			}
		}
	}
done:

	// 5. 获取结果
	data, err := s.GetData(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// 6. 转换为 VulnerabilityFinding
	return s.toFindings(data, targetURL), nil
}

// toFindings 将 sqlmap 结果转换为 VDS VulnerabilityFinding
func (s *SqlmapAdapter) toFindings(data []sqlmapDataEntry, targetURL string) []vds.VulnerabilityFinding {
	findings := make([]vds.VulnerabilityFinding, 0, len(data))

	for i, entry := range data {
		severity := "HIGH"
		if entry.DBMS != "" {
			severity = "CRITICAL"
		}

		finding := vds.VulnerabilityFinding{
			ID:             fmt.Sprintf("sqlmap-%d", i+1),
			Type:           fmt.Sprintf("SQL Injection (%s)", entry.Type),
			Severity:       severity,
			Description:    entry.Title,
			Payload:        entry.Payload,
			Location:       fmt.Sprintf("Parameter: %s, Place: %s, URL: %s", entry.Parameter, entry.Place, targetURL),
			Confidence:     90,
			OWASPCategory:  "A03:2021-Injection",
			OWASPID:        "WSTG-INPV-05",
			ATTCKTechnique: "T1190",
			ATTCKTactic:    "Initial Access",
			Exploitable:    true,
		}
		findings = append(findings, finding)
	}
	return findings
}
