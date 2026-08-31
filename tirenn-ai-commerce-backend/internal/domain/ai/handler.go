package ai

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"tirenn-ai-commerce/internal/client/ollama"
	"tirenn-ai-commerce/internal/config"
	"tirenn-ai-commerce/internal/domain"
	"tirenn-ai-commerce/internal/domain/ai/tools"
	"tirenn-ai-commerce/internal/middleware"
	"tirenn-ai-commerce/internal/response"
	"gorm.io/gorm"
)

// Handler handles all incoming HTTP AI routes
type Handler struct {
	shopperUC    *ShopperUseCase
	adminUC      *AdminUseCase
	knowledgeUC  *KnowledgeUseCase
	sessionRepo  SessionRepository
	ollamaClient *ollama.Client
	db           *gorm.DB
	cfg          *config.Config
}

// NewHandler initializes a new AI Handler
func NewHandler(
	shopperUC *ShopperUseCase,
	adminUC *AdminUseCase,
	knowledgeUC *KnowledgeUseCase,
	sessionRepo SessionRepository,
	ollamaClient *ollama.Client,
	db *gorm.DB,
	cfg *config.Config,
) *Handler {
	return &Handler{
		shopperUC:    shopperUC,
		adminUC:      adminUC,
		knowledgeUC:  knowledgeUC,
		sessionRepo:  sessionRepo,
		ollamaClient: ollamaClient,
		db:           db,
		cfg:          cfg,
	}
}

// RegisterRoutes registers all AI routes in Gin router
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	chat := r.Group("/chat")
	{
		chat.POST("/shopper", h.ChatShopper)
		chat.DELETE("/session/:id", h.DeleteSession)
		chat.POST("/admin", middleware.JWTAuth(h.cfg), middleware.RequireRole("ADMIN"), h.ChatAdmin)
	}

	catalog := r.Group("/catalog")
	{
		catalog.GET("/search", h.SearchCatalog)
	}

	knowledge := r.Group("/knowledge")
	{
		knowledge.POST("/upload", middleware.JWTAuth(h.cfg), middleware.RequireRole("ADMIN"), h.UploadKnowledge)
		knowledge.POST("/ask", h.AskKnowledge)
		knowledge.GET("/documents", h.ListKnowledgeDocuments)
		knowledge.DELETE("/documents/:id", middleware.JWTAuth(h.cfg), middleware.RequireRole("ADMIN"), h.DeleteKnowledgeDocument)
	}
}

// ChatShopper handles public AI customer shopping assistant turns
func (h *Handler) ChatShopper(c *gin.Context) {
	var req ChatShopperRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid request payload: "+err.Error(), domain.ErrBadRequest)
		return
	}

	res, err := h.shopperUC.Chat(c.Request.Context(), req.Messages, req.SessionID)
	if err != nil {
		response.Error(c, "AI service error: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":             "success",
		"reply":              res.Reply,
		"tool_calls":         res.ToolCalls,
		"suggested_products": res.SuggestedProducts,
		"cart_action":        res.CartAction,
		"session_id":         req.SessionID,
		"latency_ms":         res.LatencyMs,
	})
}

// ChatAdmin handles authenticated store administrator AI copilot turns
func (h *Handler) ChatAdmin(c *gin.Context) {
	var req ChatAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid request payload: "+err.Error(), domain.ErrBadRequest)
		return
	}

	adminIDVal, _ := c.Get("userID")
	adminEmailVal, _ := c.Get("userEmail")

	adminID := int64(1)
	if uid, ok := adminIDVal.(uint); ok && uid > 0 {
		adminID = int64(uid)
	} else if idF, ok := adminIDVal.(float64); ok && idF > 0 {
		adminID = int64(idF)
	}
	adminEmail, _ := adminEmailVal.(string)

	res, err := h.adminUC.Chat(c.Request.Context(), req.Messages, req.SessionID, adminID, adminEmail)
	if err != nil {
		response.Error(c, "Admin AI service error: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"reply":       res.Reply,
		"tool_calls":  res.ToolCalls,
		"session_id":  req.SessionID,
		"admin_email": adminEmail,
		"latency_ms":  res.LatencyMs,
	})
}

// DeleteSession purges conversation memory from Redis
func (h *Handler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		response.Error(c, "Session ID is required.", domain.ErrBadRequest)
		return
	}

	if err := h.sessionRepo.DeleteSession(c.Request.Context(), sessionID); err != nil {
		response.Error(c, "Failed to purge session: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"message":    fmt.Sprintf("Session %s successfully purged from Redis.", sessionID),
		"session_id": sessionID,
	})
}

// SearchCatalog executes hybrid semantic search
func (h *Handler) SearchCatalog(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		response.Error(c, "Query parameter is required.", domain.ErrBadRequest)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	category := c.Query("category")

	toolCfg := tools.SearchProductsConfig{
		EnableHybridSearch:          h.cfg.EnableHybridSearch,
		HybridVectorWeight:          h.cfg.HybridVectorWeight,
		HybridTextWeight:            h.cfg.HybridTextWeight,
		ChatSearchScoreThreshold:   h.cfg.ChatSearchScoreThreshold,
		ChatSearchFallbackThreshold: h.cfg.ChatSearchFallbackThreshold,
		ChatSearchLimit:             h.cfg.SearchLimit,
	}
	tool := tools.NewSearchProductsTool(h.db, h.ollamaClient, toolCfg)
	args := map[string]interface{}{
		"query":    query,
		"category": category,
		"limit":    limit,
	}

	res, err := tool.Execute(c.Request.Context(), args, nil)
	if err != nil {
		response.Error(c, err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// UploadKnowledge uploads and indexes a new SOP text or markdown file
func (h *Handler) UploadKnowledge(c *gin.Context) {
	title := c.PostForm("title")
	docType := c.DefaultPostForm("doc_type", "SOP_CUSTOMER")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, "File upload error: "+err.Error(), domain.ErrBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		response.Error(c, "Failed to read file: "+err.Error(), err)
		return
	}

	rawText := string(fileBytes)
	if strings.TrimSpace(rawText) == "" {
		response.Error(c, "Uploaded file is empty.", domain.ErrBadRequest)
		return
	}

	doc, err := h.knowledgeUC.IngestDocument(c.Request.Context(), title, docType, header.Filename, rawText, 1)
	if err != nil {
		response.Error(c, "Failed to ingest document: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":   "success",
		"message":  "Document successfully uploaded, chunked, and vector indexed.",
		"document": doc,
	})
}

// AskKnowledge queries the RAG knowledge base
func (h *Handler) AskKnowledge(c *gin.Context) {
	var req AskKnowledgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "Invalid request: "+err.Error(), domain.ErrBadRequest)
		return
	}

	docType := req.DocType
	if docType == "" {
		docType = "SOP_CUSTOMER"
	}
	topK := req.TopK
	if topK <= 0 {
		topK = 3
	}

	results, err := h.knowledgeUC.Search(c.Request.Context(), req.Query, docType, topK)
	if err != nil {
		response.Error(c, "RAG search error: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "success",
		"query":       req.Query,
		"doc_type":    docType,
		"found_count": len(results),
		"results":     results,
	})
}

// ListKnowledgeDocuments lists all indexed knowledge documents
func (h *Handler) ListKnowledgeDocuments(c *gin.Context) {
	docType := c.Query("doc_type")
	docs, err := h.knowledgeUC.ListDocuments(c.Request.Context(), docType)
	if err != nil {
		response.Error(c, "Failed to fetch documents: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"documents": docs,
	})
}

// DeleteKnowledgeDocument deletes a document and chunks
func (h *Handler) DeleteKnowledgeDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.Error(c, "Invalid document ID.", domain.ErrBadRequest)
		return
	}

	docType := c.Query("doc_type")
	if err := h.knowledgeUC.DeleteDocument(c.Request.Context(), id, docType); err != nil {
		response.Error(c, "Failed to delete document: "+err.Error(), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("Document #%d and all associated vector chunks successfully deleted.", id),
	})
}
