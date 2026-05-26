-- Only three roles: super_admin, vendor_admin (pantry), passenger

UPDATE users SET role = 'passenger'
WHERE role IN ('kitchen_staff', 'delivery_agent', 'operations_manager');

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('super_admin', 'vendor_admin', 'passenger'));
