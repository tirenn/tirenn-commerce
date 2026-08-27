-- +goose Up
-- +goose StatementBegin
ALTER TABLE categories ADD FULLTEXT idx_categories_fulltext (name, description);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE categories DROP INDEX idx_categories_fulltext;
-- +goose StatementEnd
