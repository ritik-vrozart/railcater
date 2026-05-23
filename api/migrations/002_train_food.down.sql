ALTER TABLE order_items DROP COLUMN IF EXISTS menu_item_id;
ALTER TABLE order_items ALTER COLUMN product_id SET NOT NULL;

ALTER TABLE orders
    DROP COLUMN IF EXISTS delivery_window_end,
    DROP COLUMN IF EXISTS delivery_window_start,
    DROP COLUMN IF EXISTS passenger_name,
    DROP COLUMN IF EXISTS berth,
    DROP COLUMN IF EXISTS coach,
    DROP COLUMN IF EXISTS vendor_id,
    DROP COLUMN IF EXISTS station_id,
    DROP COLUMN IF EXISTS train_id,
    DROP COLUMN IF EXISTS pnr;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_source_check;
ALTER TABLE orders ADD CONSTRAINT orders_source_check
    CHECK (source IN ('dashboard', 'whatsapp', 'api'));

DROP TABLE IF EXISTS pnr_records;
DROP TABLE IF EXISTS menu_items;
DROP TABLE IF EXISTS menu_categories;
DROP TABLE IF EXISTS vendor_stations;
DROP TABLE IF EXISTS vendors;
DROP TABLE IF EXISTS train_runs;
DROP TABLE IF EXISTS train_route_stops;
DROP TABLE IF EXISTS trains;
DROP TABLE IF EXISTS stations;
