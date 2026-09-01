package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"tirenn-ai-commerce/internal/logger"
)

// ChatMessage represents a chat message for Ollama API
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents an Ollama function tool call
type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function name and arguments
type FunctionCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
}

// ToolSchema represents Ollama tool definition schema
type ToolSchema struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef represents the parameter schema
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Client handles low-level HTTP communication with Ollama LLM and Embedding services
type Client struct {
	baseURL        string
	chatModel      string
	embeddingModel string
	httpClient     *http.Client
}

// NewClient initializes a new Ollama HTTP client adapter
func NewClient(baseURL, chatModel, embeddingModel string) *Client {
	if baseURL == "" {
		baseURL = "http://ollama:11434"
	}
	if chatModel == "" {
		chatModel = "qwen2.5:3b"
	}
	if embeddingModel == "" {
		embeddingModel = "bge-m3"
	}

	return &Client{
		baseURL:        baseURL,
		chatModel:      chatModel,
		embeddingModel: embeddingModel,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ChatRequest represents the payload format for Ollama /api/chat
type ChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ChatMessage          `json:"messages"`
	Tools    []ToolSchema           `json:"tools,omitempty"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ChatResponse represents the response format from Ollama /api/chat
type ChatResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
}

// EmbeddingRequest represents the payload format for Ollama /api/embeddings
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse represents the response format from Ollama /api/embeddings
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Chat executes a conversational inference turn against Ollama with tool schema support and deep logging
func (c *Client) Chat(ctx context.Context, messages []ChatMessage, tools []ToolSchema, temperature float64) (*ChatMessage, error) {
	promptPreview := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			promptPreview = messages[i].Content
			if len(promptPreview) > 150 {
				promptPreview = promptPreview[:150] + "..."
			}
			break
		}
	}

	finishTracker := logger.TrackLLM(ctx, c.chatModel, len(messages), promptPreview, temperature)

	reqBody := ChatRequest{
		Model:    c.chatModel,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
		Options: map[string]interface{}{
			"temperature": temperature,
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		err = fmt.Errorf("error marshaling ollama chat request: %w", err)
		finishTracker("", 0, err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/chat", c.baseURL), bytes.NewBuffer(jsonBytes))
	if err != nil {
		err = fmt.Errorf("error creating ollama chat request: %w", err)
		finishTracker("", 0, err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("error executing ollama chat request: %w", err)
		finishTracker("", 0, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("ollama chat returned status %d: %s", resp.StatusCode, string(bodyBytes))
		finishTracker("", 0, err)
		return nil, err
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		err = fmt.Errorf("error decoding ollama chat response: %w", err)
		finishTracker("", 0, err)
		return nil, err
	}

	replyPreview := chatResp.Message.Content
	if len(replyPreview) > 150 {
		replyPreview = replyPreview[:150] + "..."
	}

	finishTracker(replyPreview, len(chatResp.Message.ToolCalls), nil)
	return &chatResp.Message, nil
}

// GenerateEmbedding generates a vector embedding slice from text input directly via Ollama
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return make([]float32, 768), nil
	}

	start := time.Now()
	reqBody := EmbeddingRequest{
		Model:  c.embeddingModel,
		Prompt: text,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/embeddings", c.baseURL), bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing ollama embedding request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embedding returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var embedResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("error decoding embedding response: %w", err)
	}

	durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
	logger.LogEvent(ctx, "INFO", "ai.embedding", "EMBEDDING_GENERATED", "OllamaClient.GenerateEmbedding", fmt.Sprintf("Calculated %d-dim vector in %.2fms", len(embedResp.Embedding), durationMs), durationMs, map[string]interface{}{
		"model":       c.embeddingModel,
		"dimensions":  len(embedResp.Embedding),
		"text_length": len(text),
		"duration_ms": durationMs,
	}, nil)

	return embedResp.Embedding, nil
}

// Ping checks if Ollama service is reachable
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tags", c.baseURL), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama ping returned status %d", resp.StatusCode)
	}
	log.Printf("🤖 [AI Core] Ollama connectivity verified at %s (Model: %s)", c.baseURL, c.chatModel)
	return nil
}
