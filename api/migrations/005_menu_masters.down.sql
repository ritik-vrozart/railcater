ALTER TABLE order_items DROP COLUMN IF EXISTS menu_portion_id;
DROP TABLE IF EXISTS menu_item_portions;
ALTER TABLE menu_items DROP COLUMN IF EXISTS image_url;
ALTER TABLE menu_categories
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS food_type,
    DROP COLUMN IF EXISTS description;
