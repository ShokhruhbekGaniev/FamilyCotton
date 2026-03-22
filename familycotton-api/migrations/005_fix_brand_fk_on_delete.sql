-- +goose Up
ALTER TABLE products DROP CONSTRAINT products_brand_id_fkey;
ALTER TABLE products ADD CONSTRAINT products_brand_id_fkey
    FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE products DROP CONSTRAINT products_brand_id_fkey;
ALTER TABLE products ADD CONSTRAINT products_brand_id_fkey
    FOREIGN KEY (brand_id) REFERENCES brands(id);
