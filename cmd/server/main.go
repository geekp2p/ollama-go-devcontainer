package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ollama-go-devcontainer/internal/ollama"
)

const defaultTimeout = 2 * time.Minute

type chatPayload struct {
	Prompt string `json:"prompt"`
}

type chatReply struct {
	Reply string `json:"reply"`
}

type chatClient interface {
	Chat(context.Context, ollama.ChatRequest) (ollama.ChatResponse, error)
}

func main() {
	ollamaURL := getenv("OLLAMA_URL", "http://ollama:11434")
	model := getenv("OLLAMA_MODEL", "gpt-oss:20b")
	timeout := parseTimeout(getenv("OLLAMA_TIMEOUT", ""))

	client := ollama.New(ollamaURL, timeout)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/chat", newChatHandler(client, model, timeout))

	log.Println("Server on :8082 → /chat POST {prompt}")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parseTimeout(value string) time.Duration {
	if strings.TrimSpace(value) == "" {
		return defaultTimeout
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid OLLAMA_TIMEOUT %q: %v; using default %s", value, err, defaultTimeout)
		return defaultTimeout
	}
	if d <= 0 {
		log.Printf("invalid OLLAMA_TIMEOUT %q: must be >0; using default %s", value, defaultTimeout)
		return defaultTimeout
	}
	return d
}

func newChatHandler(client chatClient, model string, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)

		var payload chatPayload
		if err := dec.Decode(&payload); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(payload.Prompt) == "" {
			http.Error(w, "prompt is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		resp, err := client.Chat(ctx, ollama.ChatRequest{
			Model:  model,
			Stream: false,
			Messages: []ollama.ChatMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: payload.Prompt},
			},
		})
		if err != nil {
			log.Printf("chat request failed: %v", err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		reply := chatReply{Reply: resp.Message.Content}
		data, err := json.Marshal(reply)
		if err != nil {
			log.Printf("failed to marshal chat response: %v", err)
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(append(data, '\n')); err != nil {
			log.Printf("failed to write response: %v", err)
		}
	}
}
