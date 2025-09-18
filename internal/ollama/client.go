package ollama

import (
	"context"
	"time"

	"github.com/carlmjohnson/requests"
)

// Minimal types for /api/chat

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ChatResponse struct {
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// Client is a tiny wrapper over Ollama HTTP API.

type Client struct {
	BaseURL string
	Timeout time.Duration
}

func New(baseURL string) *Client {
	return &Client{BaseURL: baseURL, Timeout: 120 * time.Second}
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	var out ChatResponse
	err := requests.
		URL(c.BaseURL + "/api/chat").
		BodyJSON(req).
		ToJSON(&out).
		CheckStatus(200).
		Client(nil).
		Fetch(ctx)
	return out, err
}
