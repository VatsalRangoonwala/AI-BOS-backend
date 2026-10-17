-- +goose Up
ALTER TABLE products ADD COLUMN IF NOT EXISTS category VARCHAR(100) NOT NULL DEFAULT 'General';
ALTER TABLE products ADD COLUMN IF NOT EXISTS barcode VARCHAR(100);
CREATE INDEX IF NOT EXISTS idx_products_business_category ON products (business_id, category);
CREATE INDEX IF NOT EXISTS idx_products_business_barcode ON products (business_id, barcode) WHERE barcode IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_products_business_barcode;
DROP INDEX IF EXISTS idx_products_business_category;
ALTER TABLE products DROP COLUMN IF EXISTS barcode;
ALTER TABLE products DROP COLUMN IF EXISTS category;
