-- +goose Up
-- 1. Create sub_categories table
CREATE TABLE IF NOT EXISTS sub_categories (
    id SERIAL PRIMARY KEY,
    category_id INT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE,
    description TEXT,
    icon VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sub_categories_category_id ON sub_categories(category_id);
CREATE INDEX IF NOT EXISTS idx_sub_categories_slug ON sub_categories(slug);

-- 2. Add sub_category_id and currency to products table
ALTER TABLE products ADD COLUMN IF NOT EXISTS sub_category_id INT REFERENCES sub_categories(id) ON DELETE SET NULL;
ALTER TABLE products ADD COLUMN IF NOT EXISTS currency VARCHAR(10) NOT NULL DEFAULT 'IDR';
CREATE INDEX IF NOT EXISTS idx_products_sub_category_id ON products(sub_category_id);

-- 3. Add currency to orders and order_items table
ALTER TABLE orders ADD COLUMN IF NOT EXISTS currency VARCHAR(10) NOT NULL DEFAULT 'IDR';
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS currency VARCHAR(10) NOT NULL DEFAULT 'IDR';

-- +goose Down
ALTER TABLE order_items DROP COLUMN IF EXISTS currency;
ALTER TABLE orders DROP COLUMN IF EXISTS currency;
ALTER TABLE products DROP COLUMN IF EXISTS currency;
ALTER TABLE products DROP COLUMN IF EXISTS sub_category_id;
DROP TABLE IF EXISTS sub_categories CASCADE;
