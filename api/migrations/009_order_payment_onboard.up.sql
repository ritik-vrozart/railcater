-- Onboard train orders: payment tracking + optional delivery time (no station required)

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS payment_status TEXT NOT NULL DEFAULT 'pending'
        CHECK (payment_status IN ('pending', 'paid')),
    ADD COLUMN IF NOT EXISTS payment_method TEXT
        CHECK (payment_method IS NULL OR payment_method IN ('cash', 'upi')),
    ADD COLUMN IF NOT EXISTS expected_delivery_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_orders_payment_status ON orders (vendor_id, payment_status)
    WHERE vendor_id IS NOT NULL;
