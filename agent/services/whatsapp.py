from __future__ import annotations

import logging
from typing import Any

import httpx

from config import settings
from services.whatsapp_token import WhatsAppAuthError, get_access_token, is_token_expired, refresh_token

logger = logging.getLogger(__name__)

# WhatsApp limits
MAX_BUTTON_TITLE = 20
MAX_LIST_ROW_TITLE = 24
MAX_LIST_ROW_DESC = 72
MAX_LIST_BUTTON = 20


def _clip(text: str, limit: int) -> str:
    text = text.strip()
    if len(text) <= limit:
        return text
    return text[: limit - 1] + "…"


class WhatsAppClient:
    def __init__(self) -> None:
        self._phone_id = settings.whatsapp_phone_number_id
        self._version = settings.whatsapp_api_version

    @property
    def configured(self) -> bool:
        return bool(get_access_token() and self._phone_id)

    def _url(self) -> str:
        base = settings.whatsapp_graph_api_base.rstrip("/")
        return f"{base}/{self._version}/{self._phone_id}/messages"

    async def _send(self, to: str, payload: dict[str, Any], *, _retried: bool = False) -> dict | None:
        token = get_access_token()
        if not token or not self._phone_id:
            logger.warning("WhatsApp not configured; would send to %s", to)
            return None

        body = {
            "messaging_product": "whatsapp",
            "recipient_type": "individual",
            "to": to,
            **payload,
        }
        headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        }
        async with httpx.AsyncClient(timeout=30.0) as client:
            resp = await client.post(self._url(), json=body, headers=headers)
            if resp.status_code == 401 and not _retried and not is_token_expired():
                logger.warning("WhatsApp 401 — attempting token refresh")
                try:
                    await refresh_token(token)
                    return await self._send(to, payload, _retried=True)
                except WhatsAppAuthError as exc:
                    logger.error("WhatsApp auth failed after refresh: %s", exc)
                    raise
            if resp.status_code == 401:
                from services.whatsapp_token import mark_token_expired

                mark_token_expired()
                raise WhatsAppAuthError(
                    "WhatsApp token invalid/expired — run: python scripts/renew_whatsapp_token.py \"NAYA_TOKEN\""
                )

            if resp.status_code >= 400:
                logger.error("WhatsApp API error %s: %s", resp.status_code, resp.text)
                if resp.status_code == 401:
                    raise WhatsAppAuthError(resp.text[:300])
                resp.raise_for_status()
            return resp.json()

    async def send_text(self, to: str, body: str) -> dict | None:
        return await self._send(to, {"type": "text", "text": {"preview_url": False, "body": body[:4096]}})

    async def send_image(
        self,
        to: str,
        image_url: str,
        *,
        caption: str | None = None,
    ) -> dict | None:
        """Send an image by public HTTPS link."""
        image: dict[str, Any] = {"link": image_url}
        if caption:
            image["caption"] = caption[:1024]
        return await self._send(to, {"type": "image", "image": image})

    async def send_buttons(
        self,
        to: str,
        body: str,
        buttons: list[tuple[str, str]],
        *,
        header: str | None = None,
        header_image_url: str | None = None,
        footer: str | None = None,
    ) -> dict | None:
        """Reply buttons (max 3). buttons = [(id, title), ...]"""
        if len(buttons) > 3:
            raise ValueError("WhatsApp allows max 3 reply buttons")
        interactive: dict[str, Any] = {
            "type": "button",
            "body": {"text": body[:1024]},
            "action": {
                "buttons": [
                    {
                        "type": "reply",
                        "reply": {"id": bid, "title": _clip(title, MAX_BUTTON_TITLE)},
                    }
                    for bid, title in buttons
                ]
            },
        }
        if header_image_url:
            interactive["header"] = {"type": "image", "image": {"link": header_image_url}}
        elif header:
            interactive["header"] = {"type": "text", "text": header[:60]}
        if footer:
            interactive["footer"] = {"text": footer[:60]}
        return await self._send(to, {"type": "interactive", "interactive": interactive})

    async def send_list(
        self,
        to: str,
        body: str,
        button_label: str,
        sections: list[dict[str, Any]],
        *,
        header: str | None = None,
        footer: str | None = None,
    ) -> dict | None:
        """List message (dropdown style, single select). Max 10 rows total."""
        interactive: dict[str, Any] = {
            "type": "list",
            "body": {"text": body[:1024]},
            "action": {
                "button": _clip(button_label, MAX_LIST_BUTTON),
                "sections": sections,
            },
        }
        if header:
            interactive["header"] = {"type": "text", "text": header[:60]}
        if footer:
            interactive["footer"] = {"text": footer[:60]}
        return await self._send(to, {"type": "interactive", "interactive": interactive})

    async def send_cta_url(
        self,
        to: str,
        body: str,
        display_text: str,
        url: str,
        *,
        header: str | None = None,
        footer: str | None = None,
    ) -> dict | None:
        """Open website button (works inside WhatsApp in-app browser)."""
        interactive: dict[str, Any] = {
            "type": "cta_url",
            "body": {"text": body[:1024]},
            "action": {
                "name": "cta_url",
                "parameters": {
                    "display_text": _clip(display_text, 20),
                    "url": url,
                },
            },
        }
        if header:
            interactive["header"] = {"type": "text", "text": header[:60]}
        if footer:
            interactive["footer"] = {"text": footer[:60]}
        return await self._send(to, {"type": "interactive", "interactive": interactive})


_whatsapp: WhatsAppClient | None = None


def get_whatsapp() -> WhatsAppClient:
    global _whatsapp
    if _whatsapp is None:
        _whatsapp = WhatsAppClient()
    return _whatsapp
