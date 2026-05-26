"""
WhatsApp train food bot — FastAPI + Google ADK (Gemini).
Orders flow through the Go backend API (train number → pantry menu → orders).
"""

from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager

from pathlib import Path

from fastapi import FastAPI, HTTPException, Query, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import PlainTextResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel, Field

from config import settings
from routes.notify_api import router as notify_api_router
from routes.shop_api import router as shop_api_router
from services.agent_runner import get_agent_runner
from services import api_client
from services import train_menu_handler as menu_handler
from services.whatsapp import get_whatsapp
from services.whatsapp_token import (
    WhatsAppAuthError,
    ensure_token_ready,
    get_access_token,
    is_token_expired,
    token_file_path,
)

_STATIC_SHOP = Path(__file__).resolve().parent / "static" / "shop"

logging.basicConfig(level=logging.DEBUG if settings.debug else logging.INFO)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(_app: FastAPI):
    if settings.google_api_key:
        os.environ["GOOGLE_API_KEY"] = settings.google_api_key
    if not settings.api_base_url.strip():
        logger.warning("API_BASE_URL is not set in agent/.env")
    if not settings.public_base_url.strip():
        logger.warning("PUBLIC_BASE_URL is not set in agent/.env (required for WhatsApp webhook/shop)")
    try:
        await ensure_token_ready()
    except Exception as exc:
        logger.warning("WhatsApp token startup check failed: %s", exc)
    yield


app = FastAPI(
    title="WhatsApp Train Food Agent",
    description="ADK-powered train food ordering over WhatsApp (Go API backend)",
    version="2.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(shop_api_router)
app.include_router(notify_api_router)
if _STATIC_SHOP.is_dir():
    app.mount("/shop", StaticFiles(directory=str(_STATIC_SHOP), html=True), name="shop")


class ChatRequest(BaseModel):
    user_id: str = Field(default="demo_user", description="Phone or test user id")
    message: str


class ChatResponse(BaseModel):
    user_id: str
    reply: str


@app.get("/health")
async def health():
    return {
        "status": "ok",
        "whatsapp_configured": get_whatsapp().configured,
        "whatsapp_token_file": str(token_file_path()),
        "whatsapp_has_token": bool(get_access_token()),
        "whatsapp_token_expired": is_token_expired(),
        "gemini_configured": bool(settings.google_api_key or os.environ.get("GOOGLE_API_KEY")),
        "api_base_url": settings.api_base_url,
        "api_reachable": api_client.health_check() if api_client.api_enabled() else False,
    }


@app.post("/chat", response_model=ChatResponse)
async def chat(req: ChatRequest):
    if not (settings.google_api_key or os.environ.get("GOOGLE_API_KEY")):
        raise HTTPException(
            status_code=503,
            detail="Set GOOGLE_API_KEY in .env for Gemini ADK agents",
        )
    runner = get_agent_runner()
    reply = await runner.chat(req.user_id, req.message)
    return ChatResponse(user_id=req.user_id, reply=reply)


def _verify_whatsapp_webhook(
    hub_mode: str | None,
    hub_verify_token: str | None,
    hub_challenge: str | None,
) -> PlainTextResponse:
    """Meta sends GET with hub.mode, hub.verify_token, hub.challenge — respond with challenge as plain text."""
    if hub_mode != "subscribe":
        logger.warning("Webhook verify: bad hub.mode=%s", hub_mode)
        raise HTTPException(status_code=403, detail="Invalid hub.mode")
    if hub_verify_token != settings.whatsapp_verify_token:
        logger.warning(
            "Webhook verify: token mismatch (got %r, expected %r)",
            hub_verify_token,
            settings.whatsapp_verify_token,
        )
        raise HTTPException(status_code=403, detail="Verification failed")
    if not hub_challenge:
        raise HTTPException(status_code=400, detail="Missing hub.challenge")
    logger.info("Webhook verified successfully")
    return PlainTextResponse(content=hub_challenge)


@app.get("/webhook")
@app.get("/webhooks")  # common typo in Meta callback URL
async def verify_webhook(
    request: Request,
    hub_mode: str | None = Query(None, alias="hub.mode"),
    hub_verify_token: str | None = Query(None, alias="hub.verify_token"),
    hub_challenge: str | None = Query(None, alias="hub.challenge"),
):
    logger.info(
        "Webhook verify attempt from %s mode=%s token_match=%s",
        request.client.host if request.client else "?",
        hub_mode,
        hub_verify_token == settings.whatsapp_verify_token,
    )
    return _verify_whatsapp_webhook(hub_mode, hub_verify_token, hub_challenge)


# Meta often hits the callback URL root (without /webhook) — support both paths
@app.get("/")
async def verify_webhook_root(
    hub_mode: str | None = Query(None, alias="hub.mode"),
    hub_verify_token: str | None = Query(None, alias="hub.verify_token"),
    hub_challenge: str | None = Query(None, alias="hub.challenge"),
):
    if hub_mode == "subscribe":
        return _verify_whatsapp_webhook(hub_mode, hub_verify_token, hub_challenge)
    return {
        "service": "whatsapp_train_food_agent",
        "webhook": "/webhook",
        "health": "/health",
        "api_base_url": settings.api_base_url,
    }


def _gemini_configured() -> bool:
    return bool(settings.google_api_key or os.environ.get("GOOGLE_API_KEY"))


@app.post("/webhook")
@app.post("/webhooks")
async def whatsapp_webhook(request: Request):
    body = await request.json()
    if body.get("object") != "whatsapp_business_account":
        return {"status": "ignored"}

    wa = get_whatsapp()
    runner = get_agent_runner() if _gemini_configured() else None

    for entry in body.get("entry", []):
        for change in entry.get("changes", []):
            value = change.get("value", {})
            for message in value.get("messages", []):
                from_id = message.get("from")
                if not from_id:
                    continue

                msg_type = message.get("type")

                try:
                    if msg_type == "interactive":
                        reply_id = menu_handler.parse_interactive_reply(message)
                        if reply_id:
                            logger.info("WhatsApp interactive from %s: %s", from_id, reply_id)
                            await menu_handler.handle_interactive(wa, from_id, reply_id)
                        continue

                    if msg_type == "text":
                        text_body = message.get("text", {}).get("body", "").strip()
                        if not text_body:
                            continue
                        logger.info("WhatsApp message from %s: %s", from_id, text_body[:100])
                        # Guided flow: train → name → seat → category → menu
                        await menu_handler.handle_text(wa, from_id, text_body, runner)
                        continue

                    # Stickers, images, etc.
                    await wa.send_buttons(
                        from_id,
                        "I understand text and menu buttons. Use the menu below 👇",
                        [
                            ("menu_home", "Main menu"),
                            ("menu_train", "Order food"),
                        ],
                    )
                except WhatsAppAuthError as exc:
                    logger.error("WhatsApp auth error for %s: %s", from_id, exc)
                    # Return 200 so Meta does not retry; fix token in data/whatsapp_token.txt
                    return {"status": "ok", "whatsapp_auth": "failed"}
                except Exception as exc:
                    logger.exception("Failed to handle message from %s", from_id)
                    try:
                        if wa.configured:
                            await wa.send_text(from_id, "Sorry, something went wrong. Please try again.")
                            await menu_handler.send_main_menu(wa, from_id)
                    except WhatsAppAuthError:
                        logger.error("Could not send error reply — WhatsApp token invalid")
                    return {"status": "ok", "handled": "error", "detail": str(exc)}

    return {"status": "ok"}


@app.post("/")
async def whatsapp_webhook_root(request: Request):
    return await whatsapp_webhook(request)


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("main:app", host=settings.host, port=settings.port, reload=settings.debug)
