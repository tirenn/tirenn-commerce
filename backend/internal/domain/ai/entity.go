package ai

import (
	"context"
	"time"

	"github.com/pgvector/pgvector-go"
)

// Tool is the standard interface for all LLM executable tools
type Tool interface {
	Name() string
	Description() string
	ParametersSchema() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error)
}

// ToToolSchema converts a Tool interface into an OpenAI / Ollama ToolSchema definition
func ToToolSchema(t Tool) ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.ParametersSchema(),
		},
	}
}

// ChatMessage represents a single message turn in the conversation
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents an LLM tool invocation request
type ToolCall struct {
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents the function name and arguments (string or object)
type FunctionCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
}

// ToolSchema represents OpenAI / Ollama standard tool schema
type ToolSchema struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef represents the parameter schema for an LLM function
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ExecutedToolRecord represents an audit record of an executed tool
type ExecutedToolRecord struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
	Status string                 `json:"status"`
	Result map[string]interface{} `json:"result,omitempty"`
}

// KnowledgeDocument represents a knowledge base document in PostgreSQL
type KnowledgeDocument struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string    `gorm:"size:255;not null" json:"title"`
	DocType     string    `gorm:"size:50;not null;index" json:"doc_type"`
	Filename    string    `gorm:"size:255;not null" json:"filename"`
	TotalPages  int       `gorm:"default:1" json:"total_pages"`
	TotalChunks int       `gorm:"default:0" json:"total_chunks"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (KnowledgeDocument) TableName() string {
	return "knowledge_documents"
}

// KnowledgeChunk represents an embedded text chunk in PostgreSQL pgvector
type KnowledgeChunk struct {
	ID         int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	DocumentID int64           `gorm:"not null;index" json:"document_id"`
	ChunkIndex int             `gorm:"not null" json:"chunk_index"`
	Content    string          `gorm:"type:text;not null" json:"content"`
	PageNumber int             `gorm:"default:1" json:"page_number"`
	Embedding  pgvector.Vector `gorm:"type:vector(384);not null" json:"-"`
	CreatedAt  time.Time       `gorm:"autoCreateTime" json:"created_at"`
}

func (KnowledgeChunk) TableName() string {
	return "knowledge_chunks"
}
