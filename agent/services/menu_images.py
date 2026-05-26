"""Resolve menu item image URLs for WhatsApp (requires public HTTPS)."""

from __future__ import annotations

import logging

from config import settings

logger = logging.getLogger(__name__)

# Shown when vendor has not set image_url (WhatsApp needs HTTPS)
DEFAULT_FOOD_IMAGE = (
    "https://images.unsplash.com/photo-1546069901-ba9599a7e63c?w=600&h=400&fit=crop&q=80"
)


def resolve_menu_image_url(raw: str | None) -> str:
    """Return a public HTTPS URL suitable for WhatsApp image messages."""
    if raw:
        url = raw.strip()
        if url.startswith("//"):
            url = "https:" + url
        if url.startswith("http://"):
            url = "https://" + url[7:]
        if url.startswith("https://"):
            return url
        if url.startswith("/"):
            base = settings.api_base_url.rstrip("/")
            return f"{base}{url}"

    return DEFAULT_FOOD_IMAGE
