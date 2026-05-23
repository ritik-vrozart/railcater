# Golang API — Train Food Ordering Backend

Core backend for train catering: PNR lookup, trains/routes, vendors/menus, delivery windows, and orders.

## Prerequisites

- Go 1.23+
- PostgreSQL 16 (Docker recommended)

## Quick start

```bash
# From project root — start Postgres
docker compose up -d postgres   # Postgres on host port 5433

# API setup
cd api
cp .env.example .env
go mod tidy
make run
```

Server: `http://localhost:8080`

## Health

| Endpoint   | Description        |
|-----------|--------------------|
| `GET /health` | Liveness         |
| `GET /ready`  | DB connectivity  |

## API (`/api/v1`)

### PNR & trains

| Method | Path | Description |
|--------|------|-------------|
| GET | `/pnr/{pnr}` | PNR lookup + orderable stations (stub DB) |
| GET | `/stations` | List railway stations |
| GET | `/trains` | List trains (`?active_only=true`) |
| GET | `/trains/{id}` | Train detail + route (`?run_date=YYYY-MM-DD`) |
| GET | `/trains/{id}/stations` | Route stops |
| PATCH | `/trains/{id}/delay` | Set run delay `{ "run_date", "delay_minutes" }` |

### Vendors & menu

| Method | Path | Description |
|--------|------|-------------|
| GET | `/vendors` | List vendors |
| GET | `/vendors/{id}` | Get vendor |
| GET | `/stations/{stationId}/vendors` | Vendors serving a station |
| GET | `/vendors/{vendorId}/menu` | Vendor menu items |

### Train orders

| Method | Path | Description |
|--------|------|-------------|
| POST | `/orders/validate-delivery` | Check delivery window `{ "pnr", "station_id" }` |
| POST | `/orders/train` | Place train food order |

### Products & inventory

| Method | Path | Description |
|--------|------|-------------|
| GET | `/products` | List products (`?page=1&per_page=20&active_only=true`) |
| POST | `/products` | Create product + initial stock |
| GET | `/products/{id}` | Get product |
| PUT | `/products/{id}` | Update product |
| POST | `/products/{id}/stock` | Adjust stock `{ "delta": 10, "reason": "restock" }` |
| GET | `/products/{id}/stock/movements` | Stock history |

### CRM

| Method | Path | Description |
|--------|------|-------------|
| GET | `/customers` | List (`?q=search`) |
| POST | `/customers` | Create customer |
| GET | `/customers/{id}` | Get customer |
| PUT | `/customers/{id}` | Update customer |

### Orders

| Method | Path | Description |
|--------|------|-------------|
| GET | `/orders` | List (`?status=confirmed`) |
| POST | `/orders` | Create order (deducts stock) |
| POST | `/orders/whatsapp` | Create from WhatsApp agent |
| GET | `/orders/{id}` | Order with line items |
| PATCH | `/orders/{id}/status` | Update status |
| POST | `/inventory/check` | Pre-check stock (for FastAPI) |

## Example: train food order flow

Demo PNR: `1234567890` (seeded in migration `002_train_food`).

```bash
# 1. Lookup PNR → train, journey, orderable stations
curl -s http://localhost:8080/api/v1/pnr/1234567890

# 2. Vendors at Kota (use station_id from available_stops)
curl -s http://localhost:8080/api/v1/stations/a1000001-0000-4000-8000-000000000003/vendors

# 3. Vendor menu
curl -s http://localhost:8080/api/v1/vendors/c3000001-0000-4000-8000-000000000001/menu

# 4. Validate delivery window
curl -s -X POST http://localhost:8080/api/v1/orders/validate-delivery \
  -H "Content-Type: application/json" \
  -d '{"pnr":"1234567890","station_id":"a1000001-0000-4000-8000-000000000003"}'

# 5. Place train order
curl -s -X POST http://localhost:8080/api/v1/orders/train \
  -H "Content-Type: application/json" \
  -d '{
    "pnr":"1234567890",
    "station_id":"a1000001-0000-4000-8000-000000000003",
    "vendor_id":"c3000001-0000-4000-8000-000000000001",
    "items":[{"menu_item_id":"e5000001-0000-4000-8000-000000000001","quantity":1}]
  }'
```

## Project layout

```
api/
├── cmd/server/main.go
├── internal/
│   ├── config/
│   ├── database/
│   ├── handler/
│   ├── middleware/
│   ├── models/
│   ├── repository/
│   └── router/
└── migrations/
```

Migrations run automatically on startup.
