-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE products ADD COLUMN IF NOT EXISTS embedding vector(1024) NULL;
ALTER TABLE products ALTER COLUMN embedding TYPE vector(1024);
CREATE INDEX IF NOT EXISTS idx_products_embedding_hnsw ON products USING hnsw (embedding vector_cosine_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_embedding_hnsw;
ALTER TABLE products DROP COLUMN IF EXISTS embedding;
-- +goose StatementEnd
