package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"time"
)

type ContextKey string

const (
	RequestIDKey ContextKey = "request_id"
	TraceIDKey   ContextKey = "trace_id"
)

// StructuredLog represents the canonical log payload formatted for Loki / Grafana
type StructuredLog struct {
	Timestamp  string                 `json:"timestamp"`
	Service    string                 `json:"service"`
	Level      string                 `json:"level"`
	RequestID  string                 `json:"request_id,omitempty"`
	TraceID    string                 `json:"trace_id,omitempty"`
	Layer      string                 `json:"layer,omitempty"`
	Caller     string                 `json:"caller,omitempty"`
	Func       string                 `json:"func,omitempty"`
	Event      string                 `json:"event,omitempty"`
	DurationMs float64                `json:"duration_ms,omitempty"`
	IsSlow     bool                   `json:"is_slow,omitempty"`
	Message    string                 `json:"message"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok && reqID != "" {
		return reqID
	}
	if reqID, ok := ctx.Value("request_id").(string); ok && reqID != "" {
		return reqID
	}
	if reqID, ok := ctx.Value("userID").(string); ok && reqID != "" {
		return reqID
	}
	return ""
}

// WithRequestID wraps context with request ID
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// LogEvent logs a fully structured event with metadata and latency
func LogEvent(ctx context.Context, level, layer, event, funcName, msg string, durationMs float64, metadata map[string]interface{}, err error) {
	isSlow := durationMs > 200.0 && level != "ERROR"

	entry := StructuredLog{
		Timestamp:  time.Now().UTC().Format(time.RFC3339Nano),
		Service:    "tirenn-backend",
		Level:      level,
		RequestID:  GetRequestID(ctx),
		Layer:      layer,
		Caller:     getCaller(3),
		Func:       funcName,
		Event:      event,
		DurationMs: math.Round(durationMs*100) / 100,
		IsSlow:     isSlow,
		Message:    msg,
		Metadata:   metadata,
	}

	if err != nil {
		entry.Error = err.Error()
		if level != "ERROR" {
			entry.Level = "ERROR"
		}
	}

	bytes, _ := json.Marshal(entry)
	fmt.Printf("APP_LOG: %s\n", string(bytes))
}

// Error logs error level events with caller trace
func Error(ctx context.Context, layer, msg string, err error) {
	LogEvent(ctx, "ERROR", layer, "ERROR", "", msg, 0, nil, err)
}

// Warn logs warning level events
func Warn(ctx context.Context, layer, msg string, err error) {
	LogEvent(ctx, "WARN", layer, "WARN", "", msg, 0, nil, err)
}

// Info logs standard information level events
func Info(ctx context.Context, layer, msg string) {
	LogEvent(ctx, "INFO", layer, "INFO", "", msg, 0, nil, nil)
}

// ==============================================================================
// Function Latency & Bottleneck Tracker
// ==============================================================================

// Track measures execution latency of any function via defer Track(ctx, layer, funcName)()
func Track(ctx context.Context, layer, funcName string) func(errRef *error, extraMeta ...map[string]interface{}) {
	start := time.Now()
	return func(errRef *error, extraMeta ...map[string]interface{}) {
		durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
		level := "INFO"
		var err error
		if errRef != nil && *errRef != nil {
			err = *errRef
			level = "ERROR"
		} else if durationMs > 200.0 {
			level = "WARN"
		}

		var meta map[string]interface{}
		if len(extraMeta) > 0 && extraMeta[0] != nil {
			meta = extraMeta[0]
		}

		msg := fmt.Sprintf("[%s] executed in %.2fms", funcName, durationMs)
		if durationMs > 200.0 {
			msg = fmt.Sprintf("⚠️ [SLOW EXECUTION] %s took %.2fms", funcName, durationMs)
		}

		LogEvent(ctx, level, layer, "FUNCTION_TRACE", funcName, msg, durationMs, meta, err)
	}
}

// ==============================================================================
// AI & LLM Model Inference Tracker
// ==============================================================================

// TrackLLM records the complete LLM chat request, prompt, response, latency, and tool calls
func TrackLLM(ctx context.Context, model string, messageCount int, promptPreview string, temperature float64) func(replyPreview string, toolCallsCount int, err error) {
	start := time.Now()

	// Log LLM Request Prompt Event
	LogEvent(ctx, "INFO", "ai.llm", "LLM_PROMPT", "OllamaClient.Chat", fmt.Sprintf("🚀 Sending prompt to model %s (%d msgs, temp=%.2f)", model, messageCount, temperature), 0, map[string]interface{}{
		"model":          model,
		"message_count":  messageCount,
		"prompt_preview": promptPreview,
		"temperature":    temperature,
	}, nil)

	return func(replyPreview string, toolCallsCount int, err error) {
		durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
		level := "INFO"
		if err != nil {
			level = "ERROR"
		} else if durationMs > 3000.0 {
			level = "WARN"
		}

		LogEvent(ctx, level, "ai.llm", "LLM_RESPONSE", "OllamaClient.Chat", fmt.Sprintf("🤖 Model %s responded in %.2fms (tools called: %d)", model, durationMs, toolCallsCount), durationMs, map[string]interface{}{
			"model":            model,
			"llm_duration_ms":  durationMs,
			"tool_calls_count": toolCallsCount,
			"reply_preview":    replyPreview,
		}, err)
	}
}

// ==============================================================================
// AI Tool Execution Tracker
// ==============================================================================

// TrackTool records tool name, parameters, execution latency, and return status
func TrackTool(ctx context.Context, toolName string, args map[string]interface{}) func(status string, resultSummary string, err error) {
	start := time.Now()

	return func(status string, resultSummary string, err error) {
		durationMs := float64(time.Since(start).Nanoseconds()) / 1e6
		level := "INFO"
		if err != nil || status == "error" {
			level = "ERROR"
		} else if durationMs > 500.0 {
			level = "WARN"
		}

		LogEvent(ctx, level, "ai.tool", "TOOL_EXECUTION", toolName, fmt.Sprintf("🛠️ Tool '%s' finished in %.2fms (status: %s)", toolName, durationMs, status), durationMs, map[string]interface{}{
			"tool_name":        toolName,
			"tool_args":        args,
			"tool_duration_ms": durationMs,
			"status":           status,
			"result_summary":   resultSummary,
		}, err)
	}
}

// ==============================================================================
// RAG Pipeline Stage Tracker (Retrieval -> Augmentation -> Generation)
// ==============================================================================

type RAGTracker struct {
	ctx            context.Context
	docType        string
	query          string
	startTime      time.Time
	retrievalMs    float64
	augmentationMs float64
	generationMs   float64
	chunksFound    int
	topSimilarity  float64
}

// NewRAGTracker initializes a comprehensive RAG pipeline stage tracker
func NewRAGTracker(ctx context.Context, docType, query string) *RAGTracker {
	return &RAGTracker{
		ctx:       ctx,
		docType:   docType,
		query:     query,
		startTime: time.Now(),
	}
}

// RecordRetrieval records the vector search / embedding lookup stage latency
func (r *RAGTracker) RecordRetrieval(durationMs float64, chunksFound int, topSimilarity float64) {
	r.retrievalMs = durationMs
	r.chunksFound = chunksFound
	r.topSimilarity = topSimilarity

	LogEvent(r.ctx, "INFO", "ai.rag", "RAG_RETRIEVAL", "RAGPipeline", fmt.Sprintf("🔍 [RAG Step 1: Retrieval] Found %d chunks in %.2fms (top sim: %.2f)", chunksFound, durationMs, topSimilarity), durationMs, map[string]interface{}{
		"stage":          "retrieval",
		"doc_type":       r.docType,
		"query":          r.query,
		"retrieval_ms":   durationMs,
		"chunks_found":   chunksFound,
		"top_similarity": topSimilarity,
	}, nil)
}

// RecordAugmentation records context prompt formatting stage latency
func (r *RAGTracker) RecordAugmentation(durationMs float64) {
	r.augmentationMs = durationMs

	LogEvent(r.ctx, "INFO", "ai.rag", "RAG_AUGMENTATION", "RAGPipeline", fmt.Sprintf("🧩 [RAG Step 2: Augmentation] Context prompt assembled in %.2fms", durationMs), durationMs, map[string]interface{}{
		"stage":           "augmentation",
		"augmentation_ms": durationMs,
	}, nil)
}

// RecordGeneration records LLM answer generation stage latency
func (r *RAGTracker) RecordGeneration(durationMs float64) {
	r.generationMs = durationMs

	LogEvent(r.ctx, "INFO", "ai.rag", "RAG_GENERATION", "RAGPipeline", fmt.Sprintf("✍️ [RAG Step 3: Generation] LLM response synthesized in %.2fms", durationMs), durationMs, map[string]interface{}{
		"stage":         "generation",
		"generation_ms": durationMs,
	}, nil)
}

// Finish logs the complete end-to-end RAG lifecycle breakdown
func (r *RAGTracker) Finish(err error) {
	totalMs := float64(time.Since(r.startTime).Nanoseconds()) / 1e6
	level := "INFO"
	if err != nil {
		level = "ERROR"
	}

	LogEvent(r.ctx, level, "ai.rag", "RAG_COMPLETE", "RAGPipeline", fmt.Sprintf("⚡ [RAG Complete] Total: %.2fms | Retrieval: %.2fms | Augment: %.2fms | Generation: %.2fms", totalMs, r.retrievalMs, r.augmentationMs, r.generationMs), totalMs, map[string]interface{}{
		"rag_total_ms":        totalMs,
		"rag_retrieval_ms":    r.retrievalMs,
		"rag_augmentation_ms": r.augmentationMs,
		"rag_generation_ms":   r.generationMs,
		"doc_type":            r.docType,
		"query":               r.query,
		"chunks_found":        r.chunksFound,
		"top_similarity":      r.topSimilarity,
	}, err)
}
