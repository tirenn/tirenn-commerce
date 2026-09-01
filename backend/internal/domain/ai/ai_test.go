package ai

import (
	"context"
	"testing"

	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/domain/ai/tools"
)

func TestChunkText(t *testing.T) {
	sampleText := "Tirenn Commerce adalah platform e-commerce generasi berikutnya yang dirancang untuk performa tinggi, skalabilitas tanpa batas, dan integrasi kecerdasan buatan (AI) terdepan. Kami menyediakan katalog produk lengkap, manajemen inventaris gudang real-time, sistem autentikasi aman, serta analitik bisnis eksekutif."
	chunks := chunkText(sampleText, 100, 20)

	if len(chunks) == 0 {
		t.Fatalf("Expected chunks to be generated, got 0")
	}

	for i, c := range chunks {
		if len([]rune(c)) > 100 {
			t.Errorf("Chunk %d length %d exceeds max chunk size 100", i, len([]rune(c)))
		}
	}
}

func TestToolSchemaGeneration(t *testing.T) {
	tool := tools.NewCheckProductStockTool(nil)
	schema := ToToolSchema(tool)

	if schema.Type != "function" {
		t.Errorf("Expected schema.Type to be 'function', got '%s'", schema.Type)
	}
	if schema.Function.Name != "check_product_stock" {
		t.Errorf("Expected function name 'check_product_stock', got '%s'", schema.Function.Name)
	}
	if len(schema.Function.Description) == 0 {
		t.Errorf("Expected non-empty function description")
	}
}

func TestRAGCacheKeyNormalization(t *testing.T) {
	cacheRepo := &ragCacheRepository{}
	key1 := cacheRepo.getExactKey("SOP_CUSTOMER", "Kebijakan Pengembalian Barang")
	key2 := cacheRepo.getExactKey("SOP_CUSTOMER", "kebijakan pengembalian barang  ")

	if key1 != key2 {
		t.Errorf("Expected normalized exact cache keys to match, got %s vs %s", key1, key2)
	}
}

type MockTool struct{}

func (m *MockTool) Name() string { return "mock_tool" }
func (m *MockTool) Description() string { return "Mock tool for testing" }
func (m *MockTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{"status": "success", "result": "mock_result"}, nil
}

func TestAgentHarnessInitialization(t *testing.T) {
	ollamaClient := ollama.NewClient("http://localhost:11434", "qwen2.5:3b", "nomic-embed-text")
	mockTools := []Tool{&MockTool{}}
	harness := NewAgentHarness(ollamaClient, mockTools, "Test prompt", 5)

	if len(harness.toolsSchema) != 1 {
		t.Errorf("Expected 1 tool schema, got %d", len(harness.toolsSchema))
	}
	if harness.toolsSchema[0].Function.Name != "mock_tool" {
		t.Errorf("Expected tool name 'mock_tool', got '%s'", harness.toolsSchema[0].Function.Name)
	}
}
