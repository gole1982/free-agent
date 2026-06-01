package llm

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{},
	}
}

func (c *Client) Chat(prompt string) (string, error) {
	encodedPrompt := url.PathEscape(prompt)
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, encodedPrompt)

	resp, err := c.http.Get(reqURL)
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
