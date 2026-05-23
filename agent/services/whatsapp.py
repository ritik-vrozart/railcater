from __future__ import annotations

import logging
from typing import Any

import httpx

from config import settings

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
        self._token = settings.whatsapp_access_token
        self._phone_id = settings.whatsapp_phone_number_id
        self._version = settings.whatsapp_api_version

    @property
    def configured(self) -> bool:
        return bool(self._token and self._phone_id)

    def _url(self) -> str:
        return f"https://graph.facebook.com/{self._version}/{self._phone_id}/messages"

    async def _send(self, to: str, payload: dict[str, Any]) -> dict | None:
        if not self.configured:
            logger.warning("WhatsApp not configured; would send to %s", to)
            return None

        body = {
            "messaging_product": "whatsapp",
            "recipient_type": "individual",
            "to": to,
            **payload,
        }
        headers = {
            "Authorization": f"Bearer {self._token}",
            "Content-Type": "application/json",
        }
        async with httpx.AsyncClient(timeout=30.0) as client:
            resp = await client.post(self._url(), json=body, headers=headers)
            if resp.status_code >= 400:
                logger.error("WhatsApp API error %s: %s", resp.status_code, resp.text)
                resp.raise_for_status()
            return resp.json()

    async def send_text(self, to: str, body: str) -> dict | None:
        return await self._send(to, {"type": "text", "text": {"preview_url": False, "body": body[:4096]}})

    async def send_buttons(
        self,
        to: str,
        body: str,
        buttons: list[tuple[str, str]],
        *,
        header: str | None = None,
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
        if header:
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
