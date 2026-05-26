ALTER TABLE orders
    DROP COLUMN IF EXISTS expected_delivery_at,
    DROP COLUMN IF EXISTS payment_method,
    DROP COLUMN IF EXISTS payment_status;
