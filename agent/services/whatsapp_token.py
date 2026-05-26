"""Persistent WhatsApp access token — file-backed + optional long-lived exchange."""

from __future__ import annotations

import logging
import time
from pathlib import Path

import httpx

from config import settings

logger = logging.getLogger(__name__)

_TOKEN_PATH = Path(__file__).resolve().parent.parent / "data" / "whatsapp_token.txt"


class WhatsAppAuthError(Exception):
    """Meta Graph API rejected the access token (expired or invalid)."""


_EXPIRED_HINT = (
    "WhatsApp token expire ho chuka hai. Purane token se refresh nahi hota.\n"
    "1) Meta Business Suite → System users → Generate token (permanent), YA\n"
    "2) developers.facebook.com → Graph API Explorer → naya token → turant chalao:\n"
    "   python scripts/renew_whatsapp_token.py \"NAYA_TOKEN_YAHA\""
)

_token_marked_expired = False


def is_token_expired() -> bool:
    return _token_marked_expired


def mark_token_expired() -> None:
    global _token_marked_expired
    _token_marked_expired = True


def _is_already_expired(err_text: str) -> bool:
    t = err_text.lower()
    compact = err_text.replace(" ", "")
    return (
        "session has expired" in t
        or '"error_subcode":463' in compact
        or "error_subcode\":463" in err_text
    )


def token_file_path() -> Path:
    if settings.whatsapp_token_file:
        return Path(settings.whatsapp_token_file).expanduser()
    return _TOKEN_PATH


def _normalize(token: str) -> str:
    return token.strip().strip('"').strip("'")


def read_token_file() -> str:
    path = token_file_path()
    if not path.is_file():
        return ""
    try:
        raw = path.read_text(encoding="utf-8")
        for line in raw.splitlines():
            line = line.strip()
            if line and not line.startswith("#"):
                return _normalize(line)
        return ""
    except OSError as exc:
        logger.warning("Could not read WhatsApp token file %s: %s", path, exc)
        return ""


def save_token_file(token: str) -> None:
    path = token_file_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(_normalize(token) + "\n", encoding="utf-8")
    logger.info("WhatsApp access token saved to %s", path)


def get_access_token() -> str:
    """Env wins if set; otherwise persisted file."""
    env = _normalize(settings.whatsapp_access_token or "")
    if env:
        return env
    return read_token_file()


def persist_token(token: str) -> str:
    """Store token in file so it survives restarts (use System User permanent token)."""
    t = _normalize(token)
    if t:
        save_token_file(t)
    return t


async def exchange_long_lived(short_token: str) -> str:
    """Exchange short-lived Meta token for ~60-day token."""
    app_id = settings.whatsapp_app_id.strip()
    app_secret = settings.whatsapp_app_secret.strip()
    if not app_id or not app_secret:
        raise WhatsAppAuthError(
            "Set WHATSAPP_APP_ID and WHATSAPP_APP_SECRET in .env to auto-refresh tokens"
        )

    base = settings.whatsapp_graph_api_base.rstrip("/")
    url = f"{base}/oauth/access_token"
    params = {
        "grant_type": "fb_exchange_token",
        "client_id": app_id,
        "client_secret": app_secret,
        "fb_exchange_token": _normalize(short_token),
    }
    async with httpx.AsyncClient(timeout=30.0) as client:
        resp = await client.get(url, params=params)
        if resp.status_code >= 400:
            logger.error("Token exchange failed %s: %s", resp.status_code, resp.text)
            if _is_already_expired(resp.text):
                raise WhatsAppAuthError(_EXPIRED_HINT)
            raise WhatsAppAuthError(f"Token exchange failed: {resp.text[:200]}")
        data = resp.json()
        token = data.get("access_token", "")
        if not token:
            raise WhatsAppAuthError("Token exchange returned no access_token")
        expires_in = data.get("expires_in")
        logger.info("WhatsApp long-lived token obtained (expires_in=%s)", expires_in)
        return _normalize(token)


async def debug_token_info(token: str) -> dict:
    """Inspect token expiry (expires_at=0 means non-expiring System User token)."""
    app_id = settings.whatsapp_app_id.strip()
    app_secret = settings.whatsapp_app_secret.strip()
    if not app_id or not app_secret:
        return {}

    app_token = f"{app_id}|{app_secret}"
    base = settings.whatsapp_graph_api_base.rstrip("/")
    url = f"{base}/{settings.whatsapp_api_version}/debug_token"
    params = {"input_token": token, "access_token": app_token}
    async with httpx.AsyncClient(timeout=15.0) as client:
        resp = await client.get(url, params=params)
        if resp.status_code >= 400:
            return {}
        return resp.json().get("data", {})


async def ensure_token_ready() -> str:
    """
    On startup: use env/file token; optionally exchange for long-lived and persist.
  """
    token = get_access_token()
    if not token:
        logger.warning(
            "WhatsApp access token missing. Set WHATSAPP_ACCESS_TOKEN or create %s",
            token_file_path(),
        )
        return ""

    # Persist env token to file so one source survives redeploys
    if settings.whatsapp_access_token and not read_token_file():
        save_token_file(token)

    info = await debug_token_info(token)
    now = int(time.time())
    if info:
        expires_at = int(info.get("expires_at") or 0)
        is_valid = info.get("is_valid")

        if expires_at == 0 and is_valid:
            logger.info("WhatsApp token: non-expiring (System User / permanent)")
            return token

        if expires_at > 0:
            remaining = expires_at - now
            if remaining <= 0:
                mark_token_expired()
                logger.error("%s", _EXPIRED_HINT.replace("\n", " "))
                return token
            if remaining < 3600:
                logger.warning("WhatsApp token expires in %s min — exchanging for long-lived", remaining // 60)
                try:
                    token = await refresh_token(token)
                except WhatsAppAuthError as exc:
                    mark_token_expired()
                    logger.error("%s", exc)
                return token
            logger.info("WhatsApp token valid for ~%s hours", remaining // 3600)
            return token

        if is_valid is False:
            mark_token_expired()
            logger.error("%s", _EXPIRED_HINT.replace("\n", " "))
            return token

    return token


async def refresh_token(current: str | None = None) -> str:
    """Exchange for long-lived token and save to file."""
    if is_token_expired():
        raise WhatsAppAuthError(_EXPIRED_HINT)

    current = _normalize(current or get_access_token())
    if not current:
        raise WhatsAppAuthError("No WhatsApp token to refresh")

    new_token = await exchange_long_lived(current)
    global _token_marked_expired
    _token_marked_expired = False
    save_token_file(new_token)
    # Keep runtime env in sync for this process
    settings.whatsapp_access_token = new_token
    return new_token
