package ai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/pgvector/pgvector-go"
	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/config"
	"tirenn-ai-commerce/internal/domain/ai/tools"
	"gorm.io/gorm"
)

const SHOPPER_SYSTEM_PROMPT = `You are 'Tirenn AI Shopper', a friendly, knowledgeable, and reliable e-commerce AI shopping assistant for Tirenn Commerce.

CORE CAPABILITIES & TOOLS:
1. PRODUCT DISCOVERY & CATALOG SEARCH:
   - Use 'search_products' to discover items based on intent, category, price range, and stock availability.
   - When suggesting products, present exact names, SKUs, prices, and highlight why they match the user's needs.

2. PRODUCT DETAILS & SPECS:
   - Use 'get_product_detail' to view comprehensive specs, descriptions, currencies, and features.

3. INVENTORY CHECK:
   - Use 'check_product_stock' to confirm real-time availability before recommending or adding to cart.

4. CART ACTIONS:
   - Use 'add_to_cart' when a user wants to purchase or add an item to their cart.
   - Use 'view_cart' when a user asks to see what is currently in their cart.

5. STORE POLICIES & CUSTOMER SOP (RAG):
   - Use 'search_store_policies_and_sop' for questions about shipping, delivery times, return policies, warranty, and customer guides.
   - Strictly answer based on factual retrieved policy documents.

6. LANGUAGE POLICY (BILINGUAL MIRRORING):
   - Automatically detect the user's language from context.
   - If the user chats in BAHASA INDONESIA, respond 100% in natural, polite Bahasa Indonesia.
   - If the user chats in ENGLISH, respond 100% in natural, professional English.

7. TONE & BEHAVIOR:
   - Be helpful, concise, and proactive.
   - Never invent non-existent products or prices. Rely solely on tool facts.
`

const ADMIN_SYSTEM_PROMPT = `You are 'Tirenn Admin AI Copilot', an intelligent, secure, and executive operations assistant for Tirenn Commerce Merchant & Store Administration.

CORE RESPONSIBILITIES:
1. EXECUTIVE BUSINESS INTELLIGENCE:
   - Provide concise financial summaries, revenue metrics, order volumes, customer numbers, and sales trends using 'get_executive_dashboard_metrics' and 'get_recent_orders_overview'.
   - Format numbers clearly in Rupiah (e.g. 'Rp 15.450.000') or USD ('$1,250.00').

2. INVENTORY & STOCK OPERATIONS (STRICT 2-STEP CONFIRMATION):
   - Identify low stock products using 'get_low_stock_products'.
   - Modifying inventory stock impacts real-world warehouse and database inventory. You MUST follow a strict 2-step confirmation workflow:
     * STEP 1 (Proposal & Preview): When the admin asks to change/adjust stock (e.g. "tambah stok", "kurangin stock", "set stock"), call 'adjust_product_stock' with 'confirmed=false'. Present the proposed details clearly: Product Name, SKU, Operation Type, Current Stock, Projected New Stock, and Audit Reason. Ask for the Admin's explicit confirmation.
     * STEP 2 (Execution): When the admin confirms or agrees to the adjustment (in any language or phrasing, e.g. "ok", "oke", "ya", "yes", "proceed", "lakukan", "setuju", "proses", "sure", etc.), YOU MUST EXECUTE the tool 'adjust_product_stock' with 'confirmed=true' using the exact SKU, adjustment type, quantity, and reason from the proposal.
     * If the admin cancels or disagrees (e.g. "batal", "cancel", "tidak"), acknowledge that the adjustment was cancelled without modifying any stock.
     * CRITICAL: Never claim or state that the stock has been updated without physically executing 'adjust_product_stock' with 'confirmed=true' and receiving the result.

3. CONFIDENTIAL WAREHOUSE & ADMIN SOP (RAG):
   - Consult internal merchant operations, warehouse picking/packing guidelines, stock audit protocols, and courier escalation rules using 'search_admin_internal_sop'.
   - Quote relevant sections accurately with document titles and page numbers.

4. BILINGUAL LANGUAGE POLICY:
   - Automatically detect and match the user's language from context.
   - If the admin communicates in BAHASA INDONESIA, respond 100% in professional Bahasa Indonesia.
   - If the admin communicates in ENGLISH, respond 100% in professional English.

5. SECURITY & ROLE INTEGRITY:
   - You are exclusively accessible by authenticated store administrators.
   - Never mutate inventory without explicit admin approval.
   - Always confirm executed actions clearly with SKU, new stock quantity, and audit reason.
`

// ==============================================================================
// Shopper Use Case
// ==============================================================================

type ShopperUseCase struct {
	ollamaClient  *ollama.Client
	sessionRepo   SessionRepository
	knowledgeRepo KnowledgeRepository
	ragCacheRepo  RAGCacheRepository
	db            *gorm.DB
	agent         *AgentHarness
}

func NewShopperUseCase(
	ollamaClient *ollama.Client,
	sessionRepo SessionRepository,
	knowledgeRepo KnowledgeRepository,
	ragCacheRepo RAGCacheRepository,
	db *gorm.DB,
	cfg *config.Config,
) *ShopperUseCase {
	toolCfg := tools.SearchProductsConfig{
		EnableHybridSearch:          true,
		HybridVectorWeight:          0.70,
		HybridTextWeight:            0.30,
		ChatSearchScoreThreshold:   0.20,
		ChatSearchFallbackThreshold: 0.10,
		ChatSearchLimit:             6,
	}
	toolTemp := 0.0
	chatTemp := 0.3

	if cfg != nil {
		toolCfg.EnableHybridSearch = cfg.EnableHybridSearch
		toolCfg.HybridVectorWeight = cfg.HybridVectorWeight
		toolCfg.HybridTextWeight = cfg.HybridTextWeight
		toolCfg.ChatSearchScoreThreshold = cfg.ChatSearchScoreThreshold
		toolCfg.ChatSearchFallbackThreshold = cfg.ChatSearchFallbackThreshold
		toolCfg.ChatSearchLimit = cfg.ChatSearchLimit
		toolTemp = cfg.LLMToolTemperature
		chatTemp = cfg.LLMChatTemperature
	}

	customerTools := []Tool{
		tools.NewSearchProductsTool(db, ollamaClient, toolCfg),
		tools.NewGetProductDetailTool(db),
		tools.NewCheckProductStockTool(db),
		tools.NewAddToCartTool(db),
		tools.NewViewCartTool(),
		tools.NewSearchStorePoliciesAndSOPTool(knowledgeRepo, ragCacheRepo, ollamaClient),
	}

	agent := NewAgentHarness(ollamaClient, customerTools, SHOPPER_SYSTEM_PROMPT, 6, toolTemp, chatTemp)

	return &ShopperUseCase{
		ollamaClient:  ollamaClient,
		sessionRepo:   sessionRepo,
		knowledgeRepo: knowledgeRepo,
		ragCacheRepo:  ragCacheRepo,
		db:            db,
		agent:         agent,
	}
}

func (uc *ShopperUseCase) Chat(ctx context.Context, newMessages []ChatMessage, sessionID string) (*ChatShopperResult, error) {
	var history []ChatMessage
	if sessionID != "" {
		h, err := uc.sessionRepo.GetHistory(ctx, sessionID, 10)
		if err == nil {
			history = h
		}
	}

	combinedMessages := append(history, newMessages...)
	contextMap := map[string]interface{}{"role": "shopper"}

	res, err := uc.agent.Run(ctx, combinedMessages, contextMap)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		toSave := append(newMessages, ChatMessage{
			Role:    "assistant",
			Content: res.Reply,
		})
		_ = uc.sessionRepo.AppendMessages(ctx, sessionID, toSave)
	}

	return res, nil
}

// ==============================================================================
// Admin Copilot Use Case
// ==============================================================================

type AdminUseCase struct {
	ollamaClient  *ollama.Client
	sessionRepo   SessionRepository
	knowledgeRepo KnowledgeRepository
	ragCacheRepo  RAGCacheRepository
	db            *gorm.DB
	agent         *AgentHarness
}

func NewAdminUseCase(
	ollamaClient *ollama.Client,
	sessionRepo SessionRepository,
	knowledgeRepo KnowledgeRepository,
	ragCacheRepo RAGCacheRepository,
	db *gorm.DB,
	cfg *config.Config,
) *AdminUseCase {
	toolTemp := 0.0
	chatTemp := 0.3
	if cfg != nil {
		toolTemp = cfg.LLMToolTemperature
		chatTemp = cfg.LLMChatTemperature
	}

	adminTools := []Tool{
		tools.NewGetExecutiveDashboardMetricsTool(db),
		tools.NewGetRecentOrdersOverviewTool(db),
		tools.NewGetLowStockProductsTool(db),
		tools.NewAdjustProductStockTool(db),
		tools.NewSearchAdminInternalSOPTool(knowledgeRepo, ragCacheRepo, ollamaClient),
	}

	agent := NewAgentHarness(ollamaClient, adminTools, ADMIN_SYSTEM_PROMPT, 6, toolTemp, chatTemp)

	return &AdminUseCase{
		ollamaClient:  ollamaClient,
		sessionRepo:   sessionRepo,
		knowledgeRepo: knowledgeRepo,
		ragCacheRepo:  ragCacheRepo,
		db:            db,
		agent:         agent,
	}
}

func (uc *AdminUseCase) Chat(ctx context.Context, newMessages []ChatMessage, sessionID string, adminID int64, adminEmail string) (*ChatShopperResult, error) {
	var history []ChatMessage
	if sessionID != "" {
		h, err := uc.sessionRepo.GetHistory(ctx, sessionID, 12)
		if err == nil {
			history = h
		}
	}

	combinedMessages := append(history, newMessages...)
	contextMap := map[string]interface{}{
		"role":        "admin",
		"admin_id":    adminID,
		"admin_email": adminEmail,
	}

	res, err := uc.agent.Run(ctx, combinedMessages, contextMap)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		toSave := append(newMessages, ChatMessage{
			Role:    "assistant",
			Content: res.Reply,
		})
		_ = uc.sessionRepo.AppendMessages(ctx, sessionID, toSave)
	}

	return res, nil
}

// ==============================================================================
// Knowledge RAG Use Case
// ==============================================================================

type KnowledgeUseCase struct {
	knowledgeRepo KnowledgeRepository
	ragCacheRepo  RAGCacheRepository
	ollamaClient  *ollama.Client
}

func NewKnowledgeUseCase(
	knowledgeRepo KnowledgeRepository,
	ragCacheRepo RAGCacheRepository,
	ollamaClient *ollama.Client,
) *KnowledgeUseCase {
	return &KnowledgeUseCase{
		knowledgeRepo: knowledgeRepo,
		ragCacheRepo:  ragCacheRepo,
		ollamaClient:  ollamaClient,
	}
}

func (uc *KnowledgeUseCase) IngestDocument(ctx context.Context, title, docType, filename, rawText string, totalPages int) (*KnowledgeDocument, error) {
	if title == "" {
		title = filename
	}
	if docType == "" {
		docType = "SOP_CUSTOMER"
	}
	if totalPages <= 0 {
		totalPages = 1
	}

	chunkTexts := chunkText(rawText, 500, 50)
	if len(chunkTexts) == 0 {
		return nil, fmt.Errorf("document text is empty")
	}

	doc := &KnowledgeDocument{
		Title:       title,
		DocType:     docType,
		Filename:    filename,
		TotalPages:  totalPages,
		TotalChunks: len(chunkTexts),
	}

	if err := uc.knowledgeRepo.CreateDocument(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed creating document record: %w", err)
	}

	chunks := make([]KnowledgeChunk, 0, len(chunkTexts))
	for i, text := range chunkTexts {
		vec, err := uc.ollamaClient.GenerateEmbedding(ctx, text)
		if err != nil {
			log.Printf("⚠️ [KnowledgeUseCase] Embedding failed for chunk %d: %v", i, err)
			vec = make([]float32, 384)
		}

		chunks = append(chunks, KnowledgeChunk{
			DocumentID: doc.ID,
			ChunkIndex: i,
			Content:    text,
			PageNumber: 1,
			Embedding:  pgvector.NewVector(vec),
		})
	}

	if err := uc.knowledgeRepo.InsertChunks(ctx, chunks); err != nil {
		return nil, fmt.Errorf("failed inserting chunks: %w", err)
	}

	uc.ragCacheRepo.InvalidateDocType(ctx, docType)
	return doc, nil
}

func (uc *KnowledgeUseCase) Search(ctx context.Context, query, docType string, topK int) ([]RAGSearchResult, error) {
	if docType == "" {
		docType = "SOP_CUSTOMER"
	}
	if topK <= 0 {
		topK = 3
	}

	if cached, ok := uc.ragCacheRepo.GetExact(ctx, docType, query); ok && len(cached) > 0 {
		return cached, nil
	}

	embedding, err := uc.ollamaClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed generating query embedding: %w", err)
	}

	results, err := uc.knowledgeRepo.SearchSimilarChunks(ctx, embedding, docType, topK)
	if err != nil {
		return nil, fmt.Errorf("knowledge vector search failed: %w", err)
	}

	uc.ragCacheRepo.SetExact(ctx, docType, query, results)
	return results, nil
}

func (uc *KnowledgeUseCase) ListDocuments(ctx context.Context, docType string) ([]KnowledgeDocument, error) {
	return uc.knowledgeRepo.ListDocuments(ctx, docType)
}

func (uc *KnowledgeUseCase) DeleteDocument(ctx context.Context, id int64, docType string) error {
	if err := uc.knowledgeRepo.DeleteDocument(ctx, id); err != nil {
		return err
	}
	if docType != "" {
		uc.ragCacheRepo.InvalidateDocType(ctx, docType)
	}
	return nil
}

func chunkText(text string, chunkSize, overlap int) []string {
	clean := strings.TrimSpace(text)
	if len(clean) == 0 {
		return []string{}
	}
	if len(clean) <= chunkSize {
		return []string{clean}
	}

	var chunks []string
	start := 0
	textLen := len(clean)

	for start < textLen {
		end := start + chunkSize
		if end > textLen {
			end = textLen
		}

		chunk := strings.TrimSpace(clean[start:end])
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		if end == textLen {
			break
		}
		start += chunkSize - overlap
	}

	return chunks
}
