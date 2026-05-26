"""Resolve menu item image URLs for WhatsApp (requires public HTTPS)."""

from __future__ import annotations

import logging

from config import settings

logger = logging.getLogger(__name__)


def default_food_image_url() -> str:
    url = (settings.default_menu_image_url or "").strip()
    if url:
        return url
    logger.warning(
        "DEFAULT_MENU_IMAGE_URL not set — menu items without image_url may fail on WhatsApp"
    )
    return ""


def resolve_menu_image_url(raw: str | None) -> str | None:
    """Return a public HTTPS URL suitable for WhatsApp image messages, or None."""
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
            if base:
                return f"{base}{url}"
            logger.warning("Relative image path %s but API_BASE_URL is not set", url)
            return None

    fallback = default_food_image_url()
    return fallback or None
