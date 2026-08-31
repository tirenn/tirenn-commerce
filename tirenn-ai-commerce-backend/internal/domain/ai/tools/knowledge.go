package tools

import (
	"context"
	"fmt"
	"log"
	"time"

	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/logger"
)

// KnowledgeRepository interface required for RAG tools
type KnowledgeRepository interface {
	SearchSimilarChunks(ctx context.Context, queryEmbedding []float32, docType string, topK int) ([]RAGSearchResult, error)
}

// RAGCacheRepository interface required for RAG tools
type RAGCacheRepository interface {
	GetExact(ctx context.Context, docType, query string) ([]RAGSearchResult, bool)
	SetExact(ctx context.Context, docType, query string, results []RAGSearchResult)
}

// SearchStorePoliciesAndSOPTool searches customer-facing store policies and FAQ SOPs
type SearchStorePoliciesAndSOPTool struct {
	knowledgeRepo KnowledgeRepository
	ragCacheRepo  RAGCacheRepository
	ollamaClient  *ollama.Client
}

func NewSearchStorePoliciesAndSOPTool(
	knowledgeRepo KnowledgeRepository,
	ragCacheRepo RAGCacheRepository,
	ollamaClient *ollama.Client,
) *SearchStorePoliciesAndSOPTool {
	return &SearchStorePoliciesAndSOPTool{
		knowledgeRepo: knowledgeRepo,
		ragCacheRepo:  ragCacheRepo,
		ollamaClient:  ollamaClient,
	}
}

func (t *SearchStorePoliciesAndSOPTool) Name() string {
	return "search_store_policies_and_sop"
}

func (t *SearchStorePoliciesAndSOPTool) Description() string {
	return "Search official store policies, shipping rates, returns, warranty, and payment customer guides. Strictly restricted to customer-facing policies (SOP_CUSTOMER)."
}

func (t *SearchStorePoliciesAndSOPTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The customer's question or search query regarding store policies, delivery, return, or warranty.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchStorePoliciesAndSOPTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return map[string]interface{}{"status": "error", "message": "query is required"}, nil
	}

	docType := "SOP_CUSTOMER"
	tracker := logger.NewRAGTracker(ctx, docType, query)

	if cached, ok := t.ragCacheRepo.GetExact(ctx, docType, query); ok && len(cached) > 0 {
		log.Printf("⚡ [RAG CACHE HIT: Exact] query='%s' | doc_type='%s'", query, docType)
		tracker.RecordRetrieval(0.5, len(cached), 1.0)
		tracker.RecordAugmentation(0.2)
		tracker.Finish(nil)
		return formatCustomerRAGResponse(cached), nil
	}

	retrievalStart := time.Now()
	embedding, err := t.ollamaClient.GenerateEmbedding(ctx, query)
	if err != nil {
		log.Printf("⚠️ [SearchStorePoliciesAndSOPTool] Embedding error: %v", err)
		tracker.Finish(err)
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	results, err := t.knowledgeRepo.SearchSimilarChunks(ctx, embedding, docType, 3)
	retrievalDuration := float64(time.Since(retrievalStart).Nanoseconds()) / 1e6

	if err != nil {
		tracker.Finish(err)
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	topSim := 0.0
	if len(results) > 0 {
		topSim = results[0].Similarity
	}
	tracker.RecordRetrieval(retrievalDuration, len(results), topSim)

	augmentStart := time.Now()
	t.ragCacheRepo.SetExact(ctx, docType, query, results)
	resp := formatCustomerRAGResponse(results)
	augmentDuration := float64(time.Since(augmentStart).Nanoseconds()) / 1e6
	tracker.RecordAugmentation(augmentDuration)

	tracker.Finish(nil)
	return resp, nil
}

func formatCustomerRAGResponse(results []RAGSearchResult) map[string]interface{} {
	if len(results) == 0 {
		return map[string]interface{}{
			"status":   "not_found",
			"message":  "No matching customer store policy documents found.",
			"snippets": []string{},
		}
	}

	snippets := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		snippets = append(snippets, map[string]interface{}{
			"document_title": r.DocumentTitle,
			"page_number":    r.PageNumber,
			"content":        r.Content,
			"similarity":     fmt.Sprintf("%.2f", r.Similarity),
		})
	}

	return map[string]interface{}{
		"status":      "success",
		"found_count": len(snippets),
		"snippets":    snippets,
	}
}

// SearchAdminInternalSOPTool searches internal merchant operations and warehouse picking SOPs
type SearchAdminInternalSOPTool struct {
	knowledgeRepo KnowledgeRepository
	ragCacheRepo  RAGCacheRepository
	ollamaClient  *ollama.Client
}

func NewSearchAdminInternalSOPTool(
	knowledgeRepo KnowledgeRepository,
	ragCacheRepo RAGCacheRepository,
	ollamaClient *ollama.Client,
) *SearchAdminInternalSOPTool {
	return &SearchAdminInternalSOPTool{
		knowledgeRepo: knowledgeRepo,
		ragCacheRepo:  ragCacheRepo,
		ollamaClient:  ollamaClient,
	}
}

func (t *SearchAdminInternalSOPTool) Name() string {
	return "search_admin_internal_sop"
}

func (t *SearchAdminInternalSOPTool) Description() string {
	return "Search confidential store administration guidelines, warehouse picking/packing protocols, stock audit SOPs, and merchant policies. Strictly restricted to administrative SOPs (SOP_ADMIN)."
}

func (t *SearchAdminInternalSOPTool) ParametersSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The admin's query regarding internal warehouse picking, packing, stock audits, or escalation SOPs.",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchAdminInternalSOPTool) Execute(ctx context.Context, args map[string]interface{}, contextMap map[string]interface{}) (map[string]interface{}, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return map[string]interface{}{"status": "error", "message": "query is required"}, nil
	}

	docType := "SOP_ADMIN"
	tracker := logger.NewRAGTracker(ctx, docType, query)

	if cached, ok := t.ragCacheRepo.GetExact(ctx, docType, query); ok && len(cached) > 0 {
		log.Printf("⚡ [RAG CACHE HIT: Exact Admin] query='%s' | doc_type='%s'", query, docType)
		tracker.RecordRetrieval(0.5, len(cached), 1.0)
		tracker.RecordAugmentation(0.2)
		tracker.Finish(nil)
		return formatAdminRAGResponse(cached), nil
	}

	retrievalStart := time.Now()
	embedding, err := t.ollamaClient.GenerateEmbedding(ctx, query)
	if err != nil {
		log.Printf("⚠️ [SearchAdminInternalSOPTool] Embedding error: %v", err)
		tracker.Finish(err)
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	results, err := t.knowledgeRepo.SearchSimilarChunks(ctx, embedding, docType, 4)
	retrievalDuration := float64(time.Since(retrievalStart).Nanoseconds()) / 1e6

	if err != nil {
		tracker.Finish(err)
		return map[string]interface{}{"status": "error", "message": err.Error()}, nil
	}

	topSim := 0.0
	if len(results) > 0 {
		topSim = results[0].Similarity
	}
	tracker.RecordRetrieval(retrievalDuration, len(results), topSim)

	augmentStart := time.Now()
	t.ragCacheRepo.SetExact(ctx, docType, query, results)
	resp := formatAdminRAGResponse(results)
	augmentDuration := float64(time.Since(augmentStart).Nanoseconds()) / 1e6
	tracker.RecordAugmentation(augmentDuration)

	tracker.Finish(nil)
	return resp, nil
}

func formatAdminRAGResponse(results []RAGSearchResult) map[string]interface{} {
	if len(results) == 0 {
		return map[string]interface{}{
			"status":   "not_found",
			"message":  "No matching admin SOP documents found in knowledge base.",
			"snippets": []string{},
		}
	}

	snippets := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		snippets = append(snippets, map[string]interface{}{
			"document_title": r.DocumentTitle,
			"page_number":    r.PageNumber,
			"content":        r.Content,
			"similarity":     fmt.Sprintf("%.2f", r.Similarity),
		})
	}

	return map[string]interface{}{
		"status":      "success",
		"found_count": len(snippets),
		"snippets":    snippets,
	}
}
