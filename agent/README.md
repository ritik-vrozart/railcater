# WhatsApp Train Food Agent (FastAPI + Google ADK)

AI train food ordering bot for WhatsApp. Uses **Google ADK** with **Gemini** and the **Go backend API**.

## Features

- **Train number** → pantry auto-selected → name → seat → category / smart search → cart → checkout
- **WhatsApp interactive UI**: buttons + list menus for categories and food items
- **Natural language**: "paneer thali chahiye", "2 chai" — menu search without station/PNR
- **Go API**: `GET /trains/number/{n}`, `POST /orders/train/whatsapp`
- Hindi / English / Hinglish
- Webhook: `GET` + `POST /webhook`
- Test without WhatsApp: `POST /chat`

## Prerequisites

1. **Go API** running on port 8080 with migrations applied (see `api/README.md`)
2. **Gemini API key** for ADK
3. **WhatsApp Cloud API** tokens (for live WhatsApp)

Demo train number: `12951`

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
  -d '{"user_id":"919876543210","message":"How do I order food on this train?"}'
```

## WhatsApp webhook

1. Expose port 8000 (ngrok: `ngrok http 8000`)
2. Meta App → WhatsApp → Configuration:
   - Callback URL: `https://YOUR-NGROK/webhook`
   - Verify token: same as `WHATSAPP_VERIFY_TOKEN` in `.env`
3. Set `WHATSAPP_PHONE_NUMBER_ID` in `.env`

### Permanent WhatsApp token (recommended)

See `scripts/save_whatsapp_token.py` — store token in `data/whatsapp_token.txt` (gitignored).

## ADK agent tools (current flow)

| Tool | Purpose |
|------|---------|
| `lookup_train` | Train number → pantry |
| `set_passenger_name` / `set_delivery_seat` | Passenger details |
| `list_menu_categories` / `select_category` | Category pick |
| `search_menu` | Natural-language menu search |
| `browse_menu` | List items in category |
| `add_meal_to_cart` / `view_train_cart` | Cart |
| `place_train_order` | Checkout via `POST /orders/train/whatsapp` |

No PNR or station tools — pantry is tied to the train.

## Architecture

```
WhatsApp → agent:8000/webhook → train_menu_handler (guided flow + NLP)
                              → agent_runner + Gemini (help / edge cases)
                              → train_tools → api_client → Go API :8080
```
