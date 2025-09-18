package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"ollama-go-devcontainer/internal/ollama"
)

type chatPayload struct {
	Prompt string `json:"prompt"`
}

type chatReply struct {
	Reply string `json:"reply"`
}

func main() {
	ollamaURL := getenv("OLLAMA_URL", "http://ollama:11434")
	model := getenv("OLLAMA_MODEL", "gpt-oss-20b-q4_K_M")

	client := ollama.New(ollamaURL)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var p chatPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		resp, err := client.Chat(ctx, ollama.ChatRequest{
			Model:  model,
			Stream: false,
			Messages: []ollama.ChatMessage{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: p.Prompt},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(chatReply{Reply: resp.Message.Content})
	})

	log.Println("Server on :8082 → /chat POST {prompt}")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
