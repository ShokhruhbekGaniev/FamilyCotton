-- +goose Up
ALTER TABLE sales ADD COLUMN discount_type VARCHAR(10) NOT NULL DEFAULT 'none';
ALTER TABLE sales ADD COLUMN discount_value DECIMAL(15,2) NOT NULL DEFAULT 0;
ALTER TABLE sales ADD COLUMN discount_amount DECIMAL(15,2) NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sales DROP COLUMN discount_type;
ALTER TABLE sales DROP COLUMN discount_value;
ALTER TABLE sales DROP COLUMN discount_amount;
