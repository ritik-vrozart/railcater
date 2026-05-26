ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN (
        'super_admin', 'operations_manager', 'vendor_admin',
        'kitchen_staff', 'delivery_agent', 'passenger'
    ));
