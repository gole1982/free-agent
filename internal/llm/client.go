package llm

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL string
	apiKey  string
	byok    bool
	http    *http.Client
}

func NewClient(baseURL string, apiKey string, byok bool) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		byok:    byok,
		http:    &http.Client{},
	}
}

func (c *Client) Chat(prompt string) (string, error) {
	encodedPrompt := url.PathEscape(prompt)
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, encodedPrompt)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	if c.byok && c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response failed: %w", err)
	}

	return string(body), nil
}
