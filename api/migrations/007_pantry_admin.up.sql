-- Pantry (vendor) train links, daily menus, extended order workflow

CREATE TABLE vendor_trains (
    vendor_id UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    train_id  UUID NOT NULL REFERENCES trains(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (vendor_id, train_id)
);

CREATE INDEX idx_vendor_trains_train ON vendor_trains (train_id);

CREATE TABLE daily_menus (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id  UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    menu_date  DATE NOT NULL,
    notes      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (vendor_id, menu_date)
);

CREATE TABLE daily_menu_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_menu_id   UUID NOT NULL REFERENCES daily_menus(id) ON DELETE CASCADE,
    menu_item_id    UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    is_available    BOOLEAN NOT NULL DEFAULT true,
    stock_override  INT CHECK (stock_override IS NULL OR stock_override >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (daily_menu_id, menu_item_id)
);

CREATE INDEX idx_daily_menu_items_menu ON daily_menu_items (daily_menu_id);

-- Extend order status workflow for train pantry
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN (
        'pending', 'confirmed', 'preparing', 'ready', 'dispatched',
        'processing', 'shipped', 'delivered', 'cancelled'
    ));

-- Default department head (super admin) — password: SuperAdmin1
INSERT INTO users (tenant_id, name, email, password_hash, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Train Department Head',
    'superadmin@railcater.in',
    '$2a$10$nM3bsTPZjGQQNUpBjHYN.uUuiK1y4n.U1lxnpJfLJjLr7CJa0mlnK',
    'super_admin'
)
ON CONFLICT (tenant_id, email) DO NOTHING;
