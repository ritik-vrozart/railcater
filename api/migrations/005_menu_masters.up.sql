-- Category master enhancements
ALTER TABLE menu_categories
    ADD COLUMN description TEXT,
    ADD COLUMN food_type TEXT NOT NULL DEFAULT 'veg'
        CHECK (food_type IN ('veg', 'non_veg')),
    ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Menu item: optional image
ALTER TABLE menu_items ADD COLUMN image_url TEXT;

-- Portion-wise pricing & stock (quarter / half / full / single)
CREATE TABLE menu_item_portions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_item_id    UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    portion         TEXT NOT NULL
        CHECK (portion IN ('quarter', 'half', 'full', 'single')),
    label           TEXT NOT NULL,
    price_cents     BIGINT NOT NULL CHECK (price_cents >= 0),
    stock_quantity  INT NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (menu_item_id, portion)
);

CREATE INDEX idx_menu_portions_item ON menu_item_portions (menu_item_id, sort_order);

-- Migrate existing single prices → full portion
INSERT INTO menu_item_portions (menu_item_id, portion, label, price_cents, stock_quantity, sort_order)
SELECT id, 'full', 'Full', price_cents, 50, 3
FROM menu_items
ON CONFLICT (menu_item_id, portion) DO NOTHING;

ALTER TABLE order_items ADD COLUMN menu_portion_id UUID REFERENCES menu_item_portions(id);

UPDATE menu_categories SET food_type = 'veg' WHERE name IN ('Meals', 'Snacks');
