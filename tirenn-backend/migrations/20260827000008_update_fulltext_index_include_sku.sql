-- +goose Up
-- +goose StatementBegin
ALTER TABLE products DROP INDEX idx_products_fulltext, ADD FULLTEXT idx_products_fulltext (name, description, sku);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP INDEX idx_products_fulltext, ADD FULLTEXT idx_products_fulltext (name, description);
-- +goose StatementEnd
