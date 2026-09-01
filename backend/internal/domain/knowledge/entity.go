package knowledge

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type KnowledgeDocument struct {
	ID          uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string           `gorm:"type:text;not null" json:"title"`
	DocType     string           `gorm:"size:50;not null;default:'GENERAL';index" json:"doc_type"`
	Filename    string           `gorm:"type:text;not null" json:"filename"`
	TotalPages  int              `gorm:"not null;default:1" json:"total_pages"`
	TotalChunks int              `gorm:"not null;default:0" json:"total_chunks"`
	Chunks      []KnowledgeChunk `gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE" json:"chunks,omitempty"`
	CreatedAt   time.Time        `gorm:"autoCreateTime" json:"created_at"`
}

func (KnowledgeDocument) TableName() string {
	return "knowledge_documents"
}

type KnowledgeChunk struct {
	ID         uint             `gorm:"primaryKey;autoIncrement" json:"id"`
	DocumentID uint             `gorm:"not null;index" json:"document_id"`
	Document   *KnowledgeDocument `gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE" json:"document,omitempty"`
	ChunkIndex int              `gorm:"not null" json:"chunk_index"`
	Content    string           `gorm:"type:text;not null" json:"content"`
	PageNumber int              `gorm:"not null;default:1" json:"page_number"`
	Embedding  *pgvector.Vector `gorm:"type:vector(1024)" json:"-"`
	CreatedAt  time.Time        `gorm:"autoCreateTime" json:"created_at"`
}

func (KnowledgeChunk) TableName() string {
	return "knowledge_chunks"
}
