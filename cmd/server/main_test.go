package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

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

	handler := newChatHandler(client, "test-model", defaultTimeout, nil)
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
	if len(client.lastReq.Messages[1].Images) != 0 {
		t.Fatalf("expected no images in default request, got %v", client.lastReq.Messages[1].Images)
	}
}

func TestChatHandler_WithImages(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "vision-model", defaultTimeout, nil)

	body := `{"prompt":"what's in the photo?","images":["aGVsbG8=","d29ybGQ="]}`
	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if client.callCount != 1 {
		t.Fatalf("expected Chat to be called once, got %d", client.callCount)
	}
	if got := client.lastReq.Messages[1].Images; len(got) != 2 || got[0] != "aGVsbG8=" || got[1] != "d29ybGQ=" {
		t.Fatalf("expected images to be forwarded, got %v", got)
	}
}

func TestChatHandler_InvalidJSON(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "test-model", defaultTimeout, nil)

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

func TestChatHandler_CustomModel(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "default-model", defaultTimeout, nil)

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","model":" openthaigpt1.5-7b-instruct "}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if client.lastReq.Model != "openthaigpt1.5-7b-instruct" {
		t.Fatalf("expected model override, got %q", client.lastReq.Model)
	}
}

func TestChatHandler_CustomModelAllowedList(t *testing.T) {
	client := &stubChatClient{}
	allowed := []string{"default-model", "openthaigpt1.5-7b-instruct"}
	handler := newChatHandler(client, "default-model", defaultTimeout, allowed)

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","model":"openthaigpt1.5-7b-instruct"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if client.lastReq.Model != "openthaigpt1.5-7b-instruct" {
		t.Fatalf("expected model override, got %q", client.lastReq.Model)
	}
}

func TestChatHandler_DisallowedModel(t *testing.T) {
	client := &stubChatClient{}
	allowed := []string{"default-model"}
	handler := newChatHandler(client, "default-model", defaultTimeout, allowed)

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi","model":"llama3"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "model not allowed") {
		t.Fatalf("expected error to mention model not allowed, got %q", rec.Body.String())
	}
	if client.callCount != 0 {
		t.Fatalf("expected Chat not to be called, got %d", client.callCount)
	}
}

func TestChatHandler_DefaultModelAllowedList(t *testing.T) {
	client := &stubChatClient{}
	allowed := []string{"openthaigpt1.5-7b-instruct", "llama3"}
	handler := newChatHandler(client, "openthaigpt1.5-7b-instruct", defaultTimeout, allowed)

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"prompt":"hi"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if client.lastReq.Model != "openthaigpt1.5-7b-instruct" {
		t.Fatalf("expected default model to be used, got %q", client.lastReq.Model)
	}
}

func TestChatHandler_EmptyPrompt(t *testing.T) {
	client := &stubChatClient{}
	handler := newChatHandler(client, "test-model", defaultTimeout, nil)

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
	handler := newChatHandler(client, "test-model", defaultTimeout, nil)

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
	handler := newChatHandler(client, "test-model", defaultTimeout, nil)

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
	handler := newChatHandler(client, "test-model", defaultTimeout, nil)

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

func TestParseTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{name: "empty", input: "", want: defaultTimeout},
		{name: "spaces", input: "   ", want: defaultTimeout},
		{name: "valid", input: "5m", want: 5 * time.Minute},
		{name: "invalid", input: "nope", want: defaultTimeout},
		{name: "negative", input: "-1m", want: defaultTimeout},
		{name: "zero", input: "0", want: defaultTimeout},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseTimeout(tt.input); got != tt.want {
				t.Fatalf("parseTimeout(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseModelList(t *testing.T) {
	t.Parallel()

	input := " gpt-oss:20b ,openthaigpt1.5-7b-instruct, gpt-oss:20b ,,llama3 "
	got := parseModelList(input)
	want := []string{"gpt-oss:20b", "openthaigpt1.5-7b-instruct", "llama3"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseModelList(%q) = %v, want %v", input, got, want)
	}
}

func TestChooseDefaultModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		base    string
		allowed []string
		want    string
	}{
		{
			name: "no allowed uses trimmed base",
			base: " gpt-oss:20b ",
			want: "gpt-oss:20b",
		},
		{
			name:    "allowed contains base",
			base:    " llama3 ",
			allowed: []string{"llama3", "openthaigpt"},
			want:    "llama3",
		},
		{
			name:    "fallback when base missing",
			base:    "missing",
			allowed: []string{"llama3", "gpt-oss:20b"},
			want:    "llama3",
		},
		{
			name:    "fallback when base empty",
			base:    "",
			allowed: []string{"llama3", "gpt-oss:20b"},
			want:    "llama3",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := chooseDefaultModel(tc.base, tc.allowed); got != tc.want {
				t.Fatalf("chooseDefaultModel(%q, %v) = %q, want %q", tc.base, tc.allowed, got, tc.want)
			}
		})
	}
}

func TestContainsModel(t *testing.T) {
	t.Parallel()

	models := []string{"gpt-oss:20b", "llama3"}

	if !containsModel(models, "llama3") {
		t.Fatalf("expected llama3 to be found")
	}

	if containsModel(models, "unknown") {
		t.Fatalf("did not expect unknown to be found")
	}
}
