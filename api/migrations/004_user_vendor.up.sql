ALTER TABLE users ADD COLUMN vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL;

UPDATE users
SET vendor_id = 'c3000001-0000-4000-8000-000000000001'
WHERE role = 'vendor_admin' AND vendor_id IS NULL;
