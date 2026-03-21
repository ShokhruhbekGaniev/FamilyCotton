-- +goose Up

-- 1. Create brands table.
CREATE TABLE brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Migrate existing brand text values into brands table.
INSERT INTO brands (name)
SELECT DISTINCT brand FROM products
WHERE brand IS NOT NULL AND brand != ''
ON CONFLICT (name) DO NOTHING;

-- 3. Add brand_id column to products.
ALTER TABLE products ADD COLUMN brand_id UUID REFERENCES brands(id);

-- 4. Populate brand_id from existing brand text.
UPDATE products p
SET brand_id = b.id
FROM brands b
WHERE p.brand = b.name;

-- 5. Drop old brand text column.
ALTER TABLE products DROP COLUMN brand;

-- 6. Add index.
CREATE INDEX idx_products_brand_id ON products(brand_id);

-- +goose Down
ALTER TABLE products ADD COLUMN brand VARCHAR(255);
UPDATE products p SET brand = b.name FROM brands b WHERE p.brand_id = b.id;
ALTER TABLE products DROP COLUMN brand_id;
DROP INDEX IF EXISTS idx_products_brand_id;
DROP TABLE IF EXISTS brands;
