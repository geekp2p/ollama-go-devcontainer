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
	Prompt string   `json:"prompt"`
	Model  string   `json:"model,omitempty"`
	Images []string `json:"images,omitempty"`
}

type chatReply struct {
	Reply string `json:"reply"`
}

type chatClient interface {
	Chat(context.Context, ollama.ChatRequest) (ollama.ChatResponse, error)
}

func main() {
	ollamaURL := getenv("OLLAMA_URL", "http://ollama:11434")
	envModel := getenv("OLLAMA_MODEL", "gpt-oss:20b")
	allowedModels := parseModelList(getenv("OLLAMA_ALLOWED_MODELS", ""))
	timeout := parseTimeout(getenv("OLLAMA_TIMEOUT", ""))

	model := chooseDefaultModel(envModel, allowedModels)
	if len(allowedModels) > 0 && model != strings.TrimSpace(envModel) {
		log.Printf("default model %q not in OLLAMA_ALLOWED_MODELS; using %q instead", envModel, model)
	}

	client := ollama.New(ollamaURL, timeout)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/chat", newChatHandler(client, model, timeout, allowedModels))

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

func newChatHandler(client chatClient, model string, timeout time.Duration, allowedModels []string) http.HandlerFunc {
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

		selectedModel := model
		if trimmed := strings.TrimSpace(payload.Model); trimmed != "" {
			selectedModel = trimmed
		}

		if len(allowedModels) > 0 && !containsModel(allowedModels, selectedModel) {
			message := "model not allowed"
			if len(allowedModels) == 1 {
				message = "model not allowed: use " + allowedModels[0]
			} else {
				message = "model not allowed: use one of " + strings.Join(allowedModels, ", ")
			}
			http.Error(w, message, http.StatusBadRequest)
			return
		}

		resp, err := client.Chat(ctx, ollama.ChatRequest{
			Model:  selectedModel,
			Stream: false,
			Messages: []ollama.ChatMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: payload.Prompt, Images: payload.Images},
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

func parseModelList(value string) []string {
	parts := strings.Split(value, ",")
	var models []string
	seen := make(map[string]struct{})
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		models = append(models, name)
	}
	return models
}

func chooseDefaultModel(base string, allowed []string) string {
	trimmed := strings.TrimSpace(base)
	if len(allowed) == 0 {
		return trimmed
	}
	if trimmed != "" && containsModel(allowed, trimmed) {
		return trimmed
	}
	return allowed[0]
}

func containsModel(models []string, candidate string) bool {
	for _, m := range models {
		if m == candidate {
			return true
		}
	}
	return false
}
