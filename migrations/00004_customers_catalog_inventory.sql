-- +goose Up
CREATE TABLE IF NOT EXISTS customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(30),
    email VARCHAR(255),
    address TEXT,
    notes TEXT,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customers_business_created ON customers (business_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_customers_business_phone ON customers (business_id, phone);
CREATE INDEX IF NOT EXISTS idx_customers_business_name ON customers (business_id, name);

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    sku VARCHAR(64) NOT NULL,
    description TEXT,
    unit VARCHAR(20) NOT NULL DEFAULT 'unit',
    sale_price_minor BIGINT NOT NULL DEFAULT 0,
    cost_price_minor BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    tax_rate_basis_points INT NOT NULL DEFAULT 0,
    reorder_level INT NOT NULL DEFAULT 0,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_products_business_sku_active ON products (business_id, LOWER(sku)) WHERE archived_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_products_business_created ON products (business_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_products_business_name ON products (business_id, name);

CREATE TABLE IF NOT EXISTS inventory_balances (
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    location_id UUID,
    on_hand INT NOT NULL DEFAULT 0,
    reserved INT NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (business_id, product_id)
);

CREATE TABLE IF NOT EXISTS stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INT NOT NULL,
    movement_type VARCHAR(30) NOT NULL CHECK (movement_type IN ('initial', 'adjustment', 'received', 'sale', 'returned', 'damaged', 'transfer')),
    source_type VARCHAR(30) NOT NULL DEFAULT 'manual',
    source_id UUID,
    notes TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(128),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_movements_lookup ON stock_movements (business_id, product_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_movements_idempotency ON stock_movements (business_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS inventory_balances;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;
