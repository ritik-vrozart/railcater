# WhatsApp Train Food Agent (FastAPI + Google ADK)

AI train food ordering bot for WhatsApp. Uses **Google ADK** with **Gemini** and the **Go backend API** for PNR lookup, vendor menus, and train orders.

## Features

- **PNR lookup** → choose delivery station → vendor → menu portions → cart → checkout
- **WhatsApp interactive UI**: buttons + list menus for stations, vendors, and food items
- **Go API integration**: `GET /pnr`, `POST /orders/validate-delivery`, `POST /orders/train`
- Free-text still uses Gemini with train-food tools
- Hindi / English / Hinglish
- Webhook: `GET` + `POST /webhook`
- Test without WhatsApp: `POST /chat`

## Prerequisites

1. **Go API** running on port 8080 with migrations applied (see `api/README.md`)
2. **Gemini API key** for ADK
3. **WhatsApp Cloud API** tokens (for live WhatsApp)

Demo PNR: `1234567890` (seeded in migration `002_train_food`)

## Setup

```bash
cd agent
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
cp .env.example .env
# GOOGLE_API_KEY from https://aistudio.google.com/app/apikey
# API_BASE_URL=http://localhost:8080
# WhatsApp tokens from Meta Developer Console
```

## Run

```bash
# Terminal 1 — Go API
cd api && go run ./cmd/server

# Terminal 2 — Agent
cd agent && source .venv/bin/activate
uvicorn main:app --reload --port 8000
```

- Health: http://localhost:8000/health (includes `api_reachable`)
- Chat test:

```bash
curl -s -X POST http://localhost:8000/chat \
  -H "Content-Type: application/json" \
  -d '{"user_id":"919876543210","message":"My PNR is 1234567890"}'
```

## WhatsApp webhook

1. Expose port 8000 (ngrok: `ngrok http 8000`)
2. Meta App → WhatsApp → Configuration:
   - Callback URL: `https://YOUR-NGROK/webhook`
   - Verify token: same as `WHATSAPP_VERIFY_TOKEN` in `.env`
3. Set `WHATSAPP_PHONE_NUMBER_ID` in `.env`

### Permanent WhatsApp token (recommended)

Graph API Explorer tokens expire in **~24 hours** and cause `401 Authentication Error`.

Use a **System User** token (does not expire):

1. [Meta Business Suite](https://business.facebook.com/) → **Business settings** → **System users**
2. Add system user → **Generate token** → select your app
3. Permissions: `whatsapp_business_messaging`, `whatsapp_business_management`
4. Save the token locally (never commit it):

```bash
cd agent
source .venv/bin/activate
python scripts/save_whatsapp_token.py "YOUR_PERMANENT_TOKEN_HERE"
```

The token is stored in `data/whatsapp_token.txt` (gitignored) and used on every restart.

Optional: set `WHATSAPP_APP_ID` + `WHATSAPP_APP_SECRET` in `.env` to auto-exchange short-lived tokens for **~60-day** tokens on startup and after 401.

You can leave `WHATSAPP_ACCESS_TOKEN` empty in `.env` once the file is saved.

## ADK agent tools

| Tool | Backend API |
|------|-------------|
| `lookup_pnr` | `GET /api/v1/pnr/{pnr}` |
| `select_delivery_station` | `POST /api/v1/orders/validate-delivery` |
| `list_vendors_at_station` | `GET /api/v1/stations/{id}/vendors` |
| `select_vendor` / `browse_menu` | `GET /api/v1/vendors/{id}/menu` |
| `add_meal_to_cart` | (session) |
| `place_train_order` | `POST /api/v1/orders/train` |
| `get_recent_orders` | `GET /api/v1/orders` |

Orders use `menu_portion_id` (UUID from menu portions), not `menu_item_id`.

## Architecture

```
WhatsApp → agent:8000/webhook → train_menu_handler (buttons)
                              → agent_runner + Gemini (free text)
                              → train_tools → api_client → Go API :8080
```
