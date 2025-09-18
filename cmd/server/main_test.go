package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ollama-go-devcontainer/internal/ollama"
)

type stubChatClient struct {
	resp      ollama.ChatResponse
	err       error
	lastReq   ollama.ChatRequest
	callCount int
}

func (s *stubChatClient) Chat(ctx context.Context, req ollama.ChatRequest) (ollama.ChatResponse, error) {
	s.callCount++
	s.lastReq = req
	return s.resp, s.err
}

func TestChatHandler_Success(t *testing.T) {
	client := &stubChatClient{
		resp: ollama.ChatResponse{
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{Role: "assistant", Content: "hello there"},
		},
	}

	handler := newChatHandler(client, "test-model")
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}
	expectedBody := "{\"reply\":\"hello there\"}\n"
	if rec.Body.String() != expectedBody {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	if client.callCount != 1 {
		t.Fatalf("expected Chat to be called once, got %d", client.callCount)
	}
	if client.lastReq.Model != "test-model" {
		t.Fatalf("expected model test-model, got %s", client.lastReq.Model)
	}
	if len(client.lastReq.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(client.lastReq.Messages))
	}
	if client.lastReq.Messages[0].Role != "system" {
		t.Fatalf("expected first message role system, got %s", client.lastReq.Messages[0].Role)
	}
	if client.lastReq.Messages[1].Content != "hi" {
		t.Fatalf("expected user prompt 'hi', got %q", client.lastReq.Messages[1].Content)
	}
}

func TestChatHandler_InvalidJSON(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "test-model")

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if client.callCount != 0 {
		t.Fatalf("expected Chat not to be called, got %d", client.callCount)
	}
}

func TestChatHandler_EmptyPrompt(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "test-model")

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"   "}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if client.callCount != 0 {
		t.Fatalf("expected Chat not to be called, got %d", client.callCount)
	}
}

func TestChatHandler_InvalidJSONTrailingData(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "test-model")

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hello"}}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if client.callCount != 0 {
		t.Fatalf("expected Chat not to be called, got %d", client.callCount)
	}
}

func TestChatHandler_UpstreamError(t *testing.T) {
	client := &stubChatClient{err: errors.New("boom")}
	handler := newChatHandler(client, "test-model")

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("expected body to contain upstream error, got %q", rec.Body.String())
	}
}

func TestChatHandler_MethodNotAllowed(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "test-model")

	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
	if client.callCount != 0 {
		t.Fatalf("expected Chat not to be called, got %d", client.callCount)
	}
}
