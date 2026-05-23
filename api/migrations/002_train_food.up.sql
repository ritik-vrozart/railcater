-- Phase 1: Train food ordering domain

CREATE TABLE stations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    city       TEXT NOT NULL,
    state      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE trains (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    number     TEXT NOT NULL,
    name       TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, number)
);

CREATE TABLE train_route_stops (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    train_id            UUID NOT NULL REFERENCES trains(id) ON DELETE CASCADE,
    station_id          UUID NOT NULL REFERENCES stations(id),
    stop_order          INT NOT NULL CHECK (stop_order > 0),
    scheduled_arrival   TIME,
    scheduled_departure TIME,
    halt_minutes        INT NOT NULL DEFAULT 0 CHECK (halt_minutes >= 0),
    platform            TEXT,
    UNIQUE (train_id, stop_order),
    UNIQUE (train_id, station_id)
);

CREATE INDEX idx_train_route_stops_train ON train_route_stops (train_id, stop_order);

CREATE TABLE train_runs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    train_id       UUID NOT NULL REFERENCES trains(id) ON DELETE CASCADE,
    run_date       DATE NOT NULL,
    delay_minutes  INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (train_id, run_date)
);

CREATE TABLE vendors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    code        TEXT NOT NULL,
    phone       TEXT,
    is_active   BOOLEAN NOT NULL DEFAULT false,
    is_approved BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, code)
);

CREATE TABLE vendor_stations (
    vendor_id   UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    station_id  UUID NOT NULL REFERENCES stations(id) ON DELETE CASCADE,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY (vendor_id, station_id)
);

CREATE TABLE menu_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id  UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (vendor_id, name)
);

CREATE TABLE menu_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id    UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    category_id  UUID REFERENCES menu_categories(id) ON DELETE SET NULL,
    product_id   UUID REFERENCES products(id) ON DELETE SET NULL,
    name         TEXT NOT NULL,
    description  TEXT,
    price_cents  BIGINT NOT NULL CHECK (price_cents >= 0),
    is_veg       BOOLEAN NOT NULL DEFAULT true,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_menu_items_vendor ON menu_items (vendor_id, is_active);

CREATE TABLE pnr_records (
    pnr              TEXT PRIMARY KEY,
    train_id         UUID NOT NULL REFERENCES trains(id),
    passenger_name   TEXT NOT NULL,
    coach            TEXT NOT NULL,
    berth            TEXT NOT NULL,
    journey_date     DATE NOT NULL,
    from_station_id  UUID NOT NULL REFERENCES stations(id),
    to_station_id    UUID NOT NULL REFERENCES stations(id),
    booking_status   TEXT NOT NULL DEFAULT 'CONFIRMED'
        CHECK (booking_status IN ('CONFIRMED', 'WAITLIST', 'CANCELLED')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Extend orders for train delivery
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_source_check;
ALTER TABLE orders ADD CONSTRAINT orders_source_check
    CHECK (source IN ('dashboard', 'whatsapp', 'api', 'train'));

ALTER TABLE orders
    ADD COLUMN pnr TEXT,
    ADD COLUMN train_id UUID REFERENCES trains(id),
    ADD COLUMN station_id UUID REFERENCES stations(id),
    ADD COLUMN vendor_id UUID REFERENCES vendors(id),
    ADD COLUMN coach TEXT,
    ADD COLUMN berth TEXT,
    ADD COLUMN passenger_name TEXT,
    ADD COLUMN delivery_window_start TIMESTAMPTZ,
    ADD COLUMN delivery_window_end TIMESTAMPTZ;

CREATE INDEX idx_orders_train ON orders (train_id, station_id);

ALTER TABLE order_items
    ADD COLUMN menu_item_id UUID REFERENCES menu_items(id);

ALTER TABLE order_items ALTER COLUMN product_id DROP NOT NULL;

-- Seed: stations
INSERT INTO stations (id, code, name, city, state) VALUES
    ('a1000001-0000-4000-8000-000000000001', 'NDLS', 'New Delhi', 'New Delhi', 'DL'),
    ('a1000001-0000-4000-8000-000000000002', 'JP', 'Jaipur Junction', 'Jaipur', 'RJ'),
    ('a1000001-0000-4000-8000-000000000003', 'KOTA', 'Kota Junction', 'Kota', 'RJ'),
    ('a1000001-0000-4000-8000-000000000004', 'BPL', 'Bhopal Junction', 'Bhopal', 'MP');

-- Seed: train 12951 Mumbai Rajdhani (demo)
INSERT INTO trains (id, tenant_id, number, name) VALUES
    ('b2000001-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001', '12951', 'Mumbai Rajdhani Express');

INSERT INTO train_route_stops (train_id, station_id, stop_order, scheduled_arrival, scheduled_departure, halt_minutes, platform) VALUES
    ('b2000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000001', 1, NULL, '16:55:00', 0, '1'),
    ('b2000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000002', 2, '20:30:00', '20:35:00', 5, '2'),
    ('b2000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000003', 3, '22:10:00', '22:15:00', 5, '3'),
    ('b2000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000004', 4, '01:30:00', '01:40:00', 10, '4');

INSERT INTO train_runs (train_id, run_date, delay_minutes)
VALUES ('b2000001-0000-4000-8000-000000000001', CURRENT_DATE + 1, 15);

-- Seed: vendor + menu
INSERT INTO vendors (id, tenant_id, name, code, phone, is_active, is_approved) VALUES
    ('c3000001-0000-4000-8000-000000000001', '00000000-0000-0000-0000-000000000001', 'RailKitchen Express', 'RKE', '919800000001', true, true);

INSERT INTO vendor_stations (vendor_id, station_id) VALUES
    ('c3000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000002'),
    ('c3000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000003'),
    ('c3000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000004');

INSERT INTO menu_categories (id, vendor_id, name, sort_order) VALUES
    ('d4000001-0000-4000-8000-000000000001', 'c3000001-0000-4000-8000-000000000001', 'Meals', 1),
    ('d4000001-0000-4000-8000-000000000002', 'c3000001-0000-4000-8000-000000000001', 'Snacks', 2);

INSERT INTO menu_items (id, vendor_id, category_id, name, description, price_cents, is_veg) VALUES
    ('e5000001-0000-4000-8000-000000000001', 'c3000001-0000-4000-8000-000000000001', 'd4000001-0000-4000-8000-000000000001', 'Veg Thali', 'Rice, dal, sabzi, roti', 18000, true),
    ('e5000001-0000-4000-8000-000000000002', 'c3000001-0000-4000-8000-000000000001', 'd4000001-0000-4000-8000-000000000001', 'Paneer Thali', 'Paneer curry with rice and roti', 22000, true),
    ('e5000001-0000-4000-8000-000000000003', 'c3000001-0000-4000-8000-000000000001', 'd4000001-0000-4000-8000-000000000002', 'Veg Sandwich', 'Grilled sandwich', 8000, true),
    ('e5000001-0000-4000-8000-000000000004', 'c3000001-0000-4000-8000-000000000001', 'd4000001-0000-4000-8000-000000000002', 'Masala Chai', 'Hot tea', 3000, true);

-- Seed: demo PNR (use journey_date matching train_runs.run_date)
INSERT INTO pnr_records (pnr, train_id, passenger_name, coach, berth, journey_date, from_station_id, to_station_id) VALUES
    ('1234567890', 'b2000001-0000-4000-8000-000000000001', 'Rajesh Kumar', 'A1', '12', CURRENT_DATE + 1, 'a1000001-0000-4000-8000-000000000001', 'a1000001-0000-4000-8000-000000000004');
