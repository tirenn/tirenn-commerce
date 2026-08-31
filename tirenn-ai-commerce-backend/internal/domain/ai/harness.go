package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/logger"
)

// AgentHarness implements the canonical ReAct tool-calling loop
type AgentHarness struct {
	ollamaClient    *ollama.Client
	toolsList       []Tool
	toolsMap        map[string]Tool
	toolsSchema     []ToolSchema
	systemPrompt    string
	maxIterations   int
	toolTemperature float64
	chatTemperature float64
}

// NewAgentHarness initializes a new AgentHarness
func NewAgentHarness(
	ollamaClient *ollama.Client,
	toolList []Tool,
	systemPrompt string,
	maxIterations int,
	temperatures ...float64,
) *AgentHarness {
	if maxIterations <= 0 {
		maxIterations = 5
	}

	toolTemp := 0.0
	chatTemp := 0.3
	if len(temperatures) > 0 {
		toolTemp = temperatures[0]
	}
	if len(temperatures) > 1 {
		chatTemp = temperatures[1]
	}

	toolsMap := make(map[string]Tool, len(toolList))
	toolsSchema := make([]ToolSchema, 0, len(toolList))

	for _, t := range toolList {
		toolsMap[t.Name()] = t
		toolsSchema = append(toolsSchema, ToToolSchema(t))
	}

	return &AgentHarness{
		ollamaClient:    ollamaClient,
		toolsList:       toolList,
		toolsMap:        toolsMap,
		toolsSchema:     toolsSchema,
		systemPrompt:    systemPrompt,
		maxIterations:   maxIterations,
		toolTemperature: toolTemp,
		chatTemperature: chatTemp,
	}
}

func toOllamaTools(schemas []ToolSchema) []ollama.ToolSchema {
	out := make([]ollama.ToolSchema, len(schemas))
	for i, s := range schemas {
		out[i] = ollama.ToolSchema{
			Type: s.Type,
			Function: ollama.FunctionDef{
				Name:        s.Function.Name,
				Description: s.Function.Description,
				Parameters:  s.Function.Parameters,
			},
		}
	}
	return out
}

func toOllamaMessages(msgs []ChatMessage) []ollama.ChatMessage {
	out := make([]ollama.ChatMessage, len(msgs))
	for i, m := range msgs {
		toolCalls := make([]ollama.ToolCall, len(m.ToolCalls))
		for j, tc := range m.ToolCalls {
			toolCalls[j] = ollama.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: ollama.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		out[i] = ollama.ChatMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  toolCalls,
			ToolCallID: m.ToolCallID,
		}
	}
	return out
}

func fromOllamaMessage(msg *ollama.ChatMessage) *ChatMessage {
	if msg == nil {
		return nil
	}
	toolCalls := make([]ToolCall, len(msg.ToolCalls))
	for j, tc := range msg.ToolCalls {
		toolCalls[j] = ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return &ChatMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCalls:  toolCalls,
		ToolCallID: msg.ToolCallID,
	}
}

// Run executes the multi-turn ReAct reasoning loop
func (h *AgentHarness) Run(ctx context.Context, messages []ChatMessage, contextMap map[string]interface{}) (*ChatShopperResult, error) {
	startTime := time.Now()

	// 1. Build initial formatted messages starting with system prompt
	formattedMessages := make([]ChatMessage, 0, len(messages)+6)
	if h.systemPrompt != "" {
		formattedMessages = append(formattedMessages, ChatMessage{
			Role:    "system",
			Content: h.systemPrompt,
		})
	}
	formattedMessages = append(formattedMessages, messages...)

	var executedToolsData []ExecutedToolRecord
	var suggestedProducts []map[string]interface{}
	var cartAction map[string]interface{}

	ollamaToolSchemas := toOllamaTools(h.toolsSchema)

	for iteration := 0; iteration < h.maxIterations; iteration++ {
		// 2. Call Ollama Chat with Tools Schema
		ollamaMsgs := toOllamaMessages(formattedMessages)
		rawAssistantMsg, err := h.ollamaClient.Chat(ctx, ollamaMsgs, ollamaToolSchemas, h.toolTemperature)
		if err != nil {
			log.Printf("⚠️ [AgentHarness] LLM Chat iteration %d failed: %v", iteration, err)
			return nil, fmt.Errorf("llm inference error: %w", err)
		}

		if rawAssistantMsg == nil {
			break
		}

		assistantMsg := fromOllamaMessage(rawAssistantMsg)
		formattedMessages = append(formattedMessages, *assistantMsg)

		// 3. Check if LLM requested Tool Calls
		if len(assistantMsg.ToolCalls) == 0 {
			// Terminal response reached
			latency := float64(time.Since(startTime).Milliseconds())
			return &ChatShopperResult{
				Reply:             assistantMsg.Content,
				ToolCalls:         executedToolsData,
				SuggestedProducts: suggestedProducts,
				CartAction:        cartAction,
				LatencyMs:         latency,
			}, nil
		}

		// 4. Execute all requested tool calls
		for _, tc := range assistantMsg.ToolCalls {
			toolName := tc.Function.Name
			var argsMap map[string]interface{}

			switch a := tc.Function.Arguments.(type) {
			case map[string]interface{}:
				argsMap = a
			case string:
				_ = json.Unmarshal([]byte(a), &argsMap)
			}

			if argsMap == nil {
				argsMap = make(map[string]interface{})
			}

			toolTracker := logger.TrackTool(ctx, toolName, argsMap)

			tool, exists := h.toolsMap[toolName]
			var toolResult map[string]interface{}
			var toolErr error

			if !exists {
				toolErr = fmt.Errorf("tool '%s' is not registered", toolName)
				toolResult = map[string]interface{}{
					"status":  "error",
					"message": toolErr.Error(),
				}
			} else {
				toolResult, toolErr = tool.Execute(ctx, argsMap, contextMap)
				if toolErr != nil {
					toolResult = map[string]interface{}{
						"status":  "error",
						"message": toolErr.Error(),
					}
				}
			}

			// Capture domain side effects
			if prods, ok := toolResult["products"].([]map[string]interface{}); ok {
				suggestedProducts = append(suggestedProducts, prods...)
			}
			if prod, ok := toolResult["_full_product"].(map[string]interface{}); ok {
				suggestedProducts = append(suggestedProducts, prod)
			}
			if ca, ok := toolResult["cart_action"].(map[string]interface{}); ok {
				cartAction = ca
			}

			toolStatus := "success"
			if toolErr != nil {
				toolStatus = "error"
			}

			resSummary := fmt.Sprintf("status=%s", toolStatus)
			if msg, ok := toolResult["message"].(string); ok {
				resSummary = msg
			}
			toolTracker(toolStatus, resSummary, toolErr)

			executedToolsData = append(executedToolsData, ExecutedToolRecord{
				Name:   toolName,
				Params: argsMap,
				Status: toolStatus,
				Result: toolResult,
			})

			resultJSON, _ := json.Marshal(toolResult)

			formattedMessages = append(formattedMessages, ChatMessage{
				Role:       "tool",
				Content:    string(resultJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	// Fallback response if iteration limit exceeded
	latency := float64(time.Since(startTime).Milliseconds())
	lastContent := "I processed your request, but need additional clarification."
	for i := len(formattedMessages) - 1; i >= 0; i-- {
		if formattedMessages[i].Role == "assistant" && formattedMessages[i].Content != "" {
			lastContent = formattedMessages[i].Content
			break
		}
	}

	return &ChatShopperResult{
		Reply:             lastContent,
		ToolCalls:         executedToolsData,
		SuggestedProducts: suggestedProducts,
		CartAction:        cartAction,
		LatencyMs:         latency,
	}, nil
}
