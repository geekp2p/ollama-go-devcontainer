package ollama

import (
	"context"
	"net/http"
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
	BaseURL    string
	Timeout    time.Duration
	httpClient *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		BaseURL: baseURL,
		Timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	var out ChatResponse
	err := requests.
		URL(c.BaseURL + "/api/chat").
		BodyJSON(req).
		ToJSON(&out).
		CheckStatus(200).
		Client(c.httpClient).
		Fetch(ctx)
	return out, err
}
