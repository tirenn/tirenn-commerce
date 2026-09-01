package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := ChatResponse{
			Model: "qwen2.5:3b",
			Message: ChatMessage{
				Role:    "assistant",
				Content: "Hello! How can I help you?",
			},
			Done: true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen2.5:3b", "paraphrase-multilingual")
	msg, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hi"}}, nil, 0.2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if msg.Content != "Hello! How can I help you?" {
		t.Errorf("Unexpected reply: %s", msg.Content)
	}
}

func TestOllamaClient_GenerateEmbedding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		resp := EmbeddingResponse{
			Embedding: make([]float32, 384),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "qwen2.5:3b", "paraphrase-multilingual")
	vec, err := client.GenerateEmbedding(context.Background(), "wireless headphones")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(vec) != 384 {
		t.Errorf("Expected 384 dimensions, got %d", len(vec))
	}
}
