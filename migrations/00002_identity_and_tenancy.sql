-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(120) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'unverified', 'suspended')),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_lower ON users (LOWER(email));

CREATE TABLE IF NOT EXISTS businesses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL,
    legal_name VARCHAR(150),
    phone VARCHAR(30),
    address TEXT,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Kolkata',
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    invoice_prefix VARCHAR(10) NOT NULL DEFAULT 'INV-',
    invoice_sequence INT NOT NULL DEFAULT 1001,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(30) NOT NULL DEFAULT 'owner' CHECK (role IN ('owner', 'staff', 'admin')),
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'invited', 'suspended')),
    invited_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_memberships_business_user UNIQUE (business_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_id ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_business_role ON memberships (business_id, role);

CREATE TABLE IF NOT EXISTS refresh_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    device_info TEXT,
    ip_address VARCHAR(45),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user ON refresh_sessions (user_id, expires_at);

CREATE TABLE IF NOT EXISTS auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_type VARCHAR(30) NOT NULL CHECK (token_type IN ('email_verification', 'password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auth_tokens_lookup ON auth_tokens (token_hash, token_type);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID REFERENCES businesses(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(100),
    metadata JSONB NOT NULL DEFAULT '{}',
    request_id VARCHAR(128),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_business ON audit_logs (business_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS refresh_sessions;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS businesses;
DROP TABLE IF EXISTS users;
