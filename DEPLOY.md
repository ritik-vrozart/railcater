# Deploy — environment variables

Copy each `.env.example` to `.env` and set **production** values (no hardcoded URLs in code).

## 1. API (`api/.env`)

| Variable | Description |
|----------|-------------|
| `PORT` | HTTP port (e.g. `8080`) |
| `DATABASE_URL` | Postgres connection string |
| `CORS_ORIGINS` | Web app URL(s), comma-separated |
| `JWT_SECRET` | Strong random secret |
| `AGENT_NOTIFY_URL` | Agent notify endpoint, e.g. `https://agent.yourdomain.com/internal/notify/delivery` |
| `AGENT_NOTIFY_SECRET` | Shared secret (same as agent) |

## 2. Agent (`agent/.env`)

| Variable | Description |
|----------|-------------|
| `PUBLIC_BASE_URL` | HTTPS public URL of agent (ngrok/domain), no trailing slash |
| `API_BASE_URL` | Go API URL, e.g. `https://api.yourdomain.com` |
| `AGENT_NOTIFY_SECRET` | Same as api `.env` |
| `WHATSAPP_*` | Meta WhatsApp Cloud API credentials |
| `GOOGLE_API_KEY` | Gemini ADK |
| `DEFAULT_MENU_IMAGE_URL` | HTTPS image when menu item has no photo |

## 3. Web (`web/.env`)

| Variable | Description |
|----------|-------------|
| `VITE_API_URL` | Production API URL (build-time), e.g. `https://api.yourdomain.com` |

Dev only (optional):

| Variable | Description |
|----------|-------------|
| `VITE_DEV_PORT` | Vite port (default `5173`) |
| `VITE_API_PROXY_TARGET` | Proxy target for `/api` in dev |

## Build web for production

```bash
cd web
cp .env.example .env
# Set VITE_API_URL=https://api.yourdomain.com
npm run build
```

## Cross-service checklist

- `api` `CORS_ORIGINS` includes your web URL
- `api` `AGENT_NOTIFY_URL` points to running agent
- `agent` `API_BASE_URL` points to running api
- Meta webhook callback: `{PUBLIC_BASE_URL}/webhook`
