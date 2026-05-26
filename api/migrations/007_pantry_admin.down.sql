ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('pending', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled'));

DROP TABLE IF EXISTS daily_menu_items;
DROP TABLE IF EXISTS daily_menus;
DROP TABLE IF EXISTS vendor_trains;
