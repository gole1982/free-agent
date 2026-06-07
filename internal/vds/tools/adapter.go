package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ToolAdapter 工具适配器基础接口 — 将外部安全工具集成到 VDS 框架
type ToolAdapter interface {
	// ToolName 返回工具名称
	ToolName() string
	// IsAvailable 检查工具是否可用（API 可达或二进制存在）
	IsAvailable(ctx context.Context) bool
	// AdapterType 返回适配器类型："api" | "cli" | "xml"
	AdapterType() string
}

// APIAdapter API 类适配器基类 — 封装 HTTP 通信
type APIAdapter struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewAPIAdapter 创建 API 适配器
func NewAPIAdapter(baseURL, apiKey string) *APIAdapter {
	return &APIAdapter{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get 发送 GET 请求并解析 JSON 响应
func (a *APIAdapter) Get(ctx context.Context, path string, result interface{}) error {
	url := a.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if a.APIKey != "" {
		req.Header.Set("X-API-Key", a.APIKey)
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Post 发送 POST 请求并解析 JSON 响应
func (a *APIAdapter) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	url := a.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = strings.NewReader(string(data))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if a.APIKey != "" {
		req.Header.Set("X-API-Key", a.APIKey)
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// PollUntilDone 轮询直到任务完成或超时
func (a *APIAdapter) PollUntilDone(ctx context.Context, path string, interval time.Duration, result interface{}, isDone func(interface{}) bool) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var current json.RawMessage
			if err := a.Get(ctx, path, &current); err != nil {
				return err
			}
			if result != nil {
				if err := json.Unmarshal(current, result); err != nil {
					return fmt.Errorf("unmarshal poll result: %w", err)
				}
			}
			if isDone(result) {
				return nil
			}
		}
	}
}

// CheckAvailability 检查 API 是否可达
func (a *APIAdapter) CheckAvailability(ctx context.Context, healthPath string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL+healthPath, nil)
	if err != nil {
		return false
	}
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
