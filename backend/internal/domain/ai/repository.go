package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/redis/go-redis/v9"
	"tirenn-ai-commerce/internal/domain/ai/tools"
	"gorm.io/gorm"
)

// ==============================================================================
// Session Repository (Redis Conversation State)
// ==============================================================================

type SessionRepository interface {
	GetHistory(ctx context.Context, sessionID string, limit int) ([]ChatMessage, error)
	AppendMessages(ctx context.Context, sessionID string, messages []ChatMessage) error
	DeleteSession(ctx context.Context, sessionID string) error
}

type sessionRepository struct {
	client     *redis.Client
	ttl        time.Duration
	maxHistory int
	maxStored  int
}

func NewSessionRepository(client *redis.Client) SessionRepository {
	return &sessionRepository{
		client:     client,
		ttl:        24 * time.Hour,
		maxHistory: 10,
		maxStored:  40,
	}
}

func (r *sessionRepository) getKey(sessionID string) string {
	return fmt.Sprintf("chat:session:%s", sessionID)
}

func (r *sessionRepository) GetHistory(ctx context.Context, sessionID string, limit int) ([]ChatMessage, error) {
	if r.client == nil || sessionID == "" {
		return []ChatMessage{}, nil
	}

	key := r.getKey(sessionID)
	n := limit
	if n <= 0 {
		n = r.maxHistory
	}

	rawList, err := r.client.LRange(ctx, key, int64(-n), -1).Result()
	if err != nil {
		log.Printf("⚠️ [SessionRepository] Error reading session %s from Redis: %v", sessionID, err)
		return []ChatMessage{}, nil
	}

	results := make([]ChatMessage, 0, len(rawList))
	for _, raw := range rawList {
		var msg ChatMessage
		if err := json.Unmarshal([]byte(raw), &msg); err == nil {
			results = append(results, msg)
		}
	}

	return results, nil
}

func (r *sessionRepository) AppendMessages(ctx context.Context, sessionID string, messages []ChatMessage) error {
	if r.client == nil || sessionID == "" || len(messages) == 0 {
		return nil
	}

	key := r.getKey(sessionID)
	pipe := r.client.Pipeline()

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		pipe.RPush(ctx, key, string(data))
	}

	pipe.LTrim(ctx, key, int64(-r.maxStored), -1)
	pipe.Expire(ctx, key, r.ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("⚠️ [SessionRepository] Error saving messages to Redis: %v", err)
	}

	return err
}

func (r *sessionRepository) DeleteSession(ctx context.Context, sessionID string) error {
	if r.client == nil || sessionID == "" {
		return nil
	}
	return r.client.Del(ctx, r.getKey(sessionID)).Err()
}

// ==============================================================================
// Knowledge Repository (PostgreSQL pgvector)
// ==============================================================================

type KnowledgeRepository interface {
	CreateDocument(ctx context.Context, doc *KnowledgeDocument) error
	InsertChunks(ctx context.Context, chunks []KnowledgeChunk) error
	ListDocuments(ctx context.Context, docType string) ([]KnowledgeDocument, error)
	SearchSimilarChunks(ctx context.Context, queryEmbedding []float32, docType string, topK int) ([]tools.RAGSearchResult, error)
	DeleteDocument(ctx context.Context, id int64) error
}

type knowledgeRepository struct {
	db *gorm.DB
}

func NewKnowledgeRepository(db *gorm.DB) KnowledgeRepository {
	return &knowledgeRepository{db: db}
}

func (r *knowledgeRepository) CreateDocument(ctx context.Context, doc *KnowledgeDocument) error {
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *knowledgeRepository) InsertChunks(ctx context.Context, chunks []KnowledgeChunk) error {
	if len(chunks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&chunks).Error
}

func (r *knowledgeRepository) ListDocuments(ctx context.Context, docType string) ([]KnowledgeDocument, error) {
	var docs []KnowledgeDocument
	query := r.db.WithContext(ctx).Order("id DESC")
	if docType != "" {
		query = query.Where("doc_type = ?", docType)
	}
	err := query.Find(&docs).Error
	return docs, err
}

func (r *knowledgeRepository) SearchSimilarChunks(ctx context.Context, queryEmbedding []float32, docType string, topK int) ([]tools.RAGSearchResult, error) {
	if topK <= 0 {
		topK = 3
	}

	vec := pgvector.NewVector(queryEmbedding)

	var results []tools.RAGSearchResult
	sqlQuery := `
		SELECT 
			kc.id as chunk_id,
			kc.document_id,
			kd.title as document_title,
			kd.doc_type,
			kc.page_number,
			kc.content,
			(1 - (kc.embedding <=> ?)) as similarity
		FROM knowledge_chunks kc
		INNER JOIN knowledge_documents kd ON kd.id = kc.document_id
		WHERE kd.doc_type = ?
		ORDER BY kc.embedding <=> ?
		LIMIT ?
	`

	err := r.db.WithContext(ctx).Raw(sqlQuery, vec, docType, vec, topK).Scan(&results).Error
	return results, err
}

func (r *knowledgeRepository) DeleteDocument(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("document_id = ?", id).Delete(&KnowledgeChunk{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&KnowledgeDocument{}).Error
	})
}

// ==============================================================================
// RAG Cache Repository (Redis Exact Cache)
// ==============================================================================

type RAGCacheRepository interface {
	GetExact(ctx context.Context, docType, query string) ([]tools.RAGSearchResult, bool)
	SetExact(ctx context.Context, docType, query string, results []tools.RAGSearchResult)
	InvalidateDocType(ctx context.Context, docType string)
}

type ragCacheRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRAGCacheRepository(client *redis.Client) RAGCacheRepository {
	return &ragCacheRepository{
		client: client,
		ttl:    2 * time.Hour,
	}
}

func (r *ragCacheRepository) getExactKey(docType, query string) string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	hash := sha256.Sum256([]byte(normalized))
	hashStr := hex.EncodeToString(hash[:])
	return fmt.Sprintf("rag:exact:%s:%s", docType, hashStr)
}

func (r *ragCacheRepository) GetExact(ctx context.Context, docType, query string) ([]tools.RAGSearchResult, bool) {
	if r.client == nil {
		return nil, false
	}

	key := r.getExactKey(docType, query)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var results []tools.RAGSearchResult
	if err := json.Unmarshal([]byte(val), &results); err != nil {
		return nil, false
	}

	return results, true
}

func (r *ragCacheRepository) SetExact(ctx context.Context, docType, query string, results []tools.RAGSearchResult) {
	if r.client == nil || len(results) == 0 {
		return
	}

	key := r.getExactKey(docType, query)
	data, err := json.Marshal(results)
	if err != nil {
		return
	}

	r.client.Set(ctx, key, string(data), r.ttl)
}

func (r *ragCacheRepository) InvalidateDocType(ctx context.Context, docType string) {
	if r.client == nil {
		return
	}

	pattern := fmt.Sprintf("rag:exact:%s:*", docType)
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			r.client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}
