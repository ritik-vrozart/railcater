"""HTTP client for the Go backend API (train food + orders)."""

from __future__ import annotations

import logging
from typing import Any

import httpx

from config import settings

logger = logging.getLogger(__name__)

_client: httpx.Client | None = None


class APIError(Exception):
    def __init__(self, message: str, status_code: int | None = None) -> None:
        super().__init__(message)
        self.status_code = status_code


def api_enabled() -> bool:
    return bool(settings.api_base_url.strip())


def get_api() -> httpx.Client:
    global _client
    if _client is None:
        base = settings.api_base_url.rstrip("/") + "/api/v1"
        _client = httpx.Client(base_url=base, timeout=30.0)
    return _client


def _parse_error(resp: httpx.Response) -> str:
    try:
        body = resp.json()
        if isinstance(body, dict) and body.get("error"):
            return str(body["error"])
    except Exception:
        pass
    return resp.text or f"HTTP {resp.status_code}"


def _request(method: str, path: str, **kwargs: Any) -> Any:
    if not api_enabled():
        raise APIError("Backend API not configured (set API_BASE_URL in .env)")
    client = get_api()
    try:
        resp = client.request(method, path, **kwargs)
    except httpx.RequestError as exc:
        logger.exception("API request failed: %s %s", method, path)
        raise APIError(f"Cannot reach backend API: {exc}") from exc
    if resp.status_code >= 400:
        raise APIError(_parse_error(resp), resp.status_code)
    if resp.status_code == 204 or not resp.content:
        return None
    return resp.json()


def health_check() -> bool:
    if not api_enabled():
        return False
    try:
        root = settings.api_base_url.rstrip("/")
        r = httpx.get(f"{root}/health", timeout=5.0)
        return r.status_code == 200
    except Exception:
        return False


# --- Train food ---


def lookup_train_by_number(number: str) -> dict[str, Any]:
    return _request("GET", f"/trains/number/{number.strip()}")


def list_menu_categories(vendor_id: str, *, active_only: bool = True) -> list[dict[str, Any]]:
    params = {}
    if not active_only:
        params["active_only"] = "false"
    data = _request("GET", f"/vendors/{vendor_id}/menu/categories", params=params or None)
    return data.get("data", []) if isinstance(data, dict) else []


def lookup_pnr(pnr: str) -> dict[str, Any]:
    return _request("GET", f"/pnr/{pnr.strip()}")


def validate_delivery(pnr: str, station_id: str) -> dict[str, Any]:
    return _request(
        "POST",
        "/orders/validate-delivery",
        json={"pnr": pnr.strip(), "station_id": station_id},
    )


def list_station_vendors(station_id: str) -> list[dict[str, Any]]:
    data = _request("GET", f"/stations/{station_id}/vendors")
    return data.get("data", []) if isinstance(data, dict) else []


def get_vendor_menu(vendor_id: str, *, active_only: bool = True, menu_date: str | None = None) -> list[dict[str, Any]]:
    from datetime import date

    params: dict[str, str] = {}
    if not active_only:
        params["active_only"] = "false"
    params["date"] = menu_date or date.today().isoformat()
    data = _request("GET", f"/vendors/{vendor_id}/menu", params=params)
    return data.get("data", []) if isinstance(data, dict) else []


def create_whatsapp_train_order(
    *,
    train_number: str,
    vendor_id: str,
    passenger_name: str,
    coach: str,
    berth: str,
    items: list[dict[str, Any]],
    customer_id: str | None = None,
    train_id: str | None = None,
    station_id: str | None = None,
    notes: str | None = None,
) -> dict[str, Any]:
    body: dict[str, Any] = {
        "train_number": train_number.strip(),
        "vendor_id": vendor_id,
        "passenger_name": passenger_name.strip(),
        "coach": coach.strip(),
        "berth": berth.strip(),
        "items": items,
    }
    if train_id:
        body["train_id"] = train_id
    if station_id:
        body["station_id"] = station_id
    if customer_id:
        body["customer_id"] = customer_id
    if notes:
        body["notes"] = notes
    return _request("POST", "/orders/train/whatsapp", json=body)


def create_train_order(
    *,
    pnr: str,
    station_id: str,
    vendor_id: str,
    items: list[dict[str, Any]],
    customer_id: str | None = None,
    coach: str | None = None,
    berth: str | None = None,
    notes: str | None = None,
) -> dict[str, Any]:
    body: dict[str, Any] = {
        "pnr": pnr.strip(),
        "station_id": station_id,
        "vendor_id": vendor_id,
        "items": items,
    }
    if customer_id:
        body["customer_id"] = customer_id
    if coach:
        body["coach"] = coach
    if berth:
        body["berth"] = berth
    if notes:
        body["notes"] = notes
    return _request("POST", "/orders/train", json=body)


def get_order(order_id: str) -> dict[str, Any]:
    return _request("GET", f"/orders/{order_id}")


def list_orders(
    *,
    page: int = 1,
    per_page: int = 10,
    status: str | None = None,
    customer_id: str | None = None,
) -> list[dict[str, Any]]:
    params: dict[str, Any] = {"page": page, "per_page": per_page}
    if status:
        params["status"] = status
    if customer_id:
        params["customer_id"] = customer_id
    data = _request("GET", "/orders", params=params)
    return data.get("data", []) if isinstance(data, dict) else []


def get_customer_by_phone(phone: str) -> dict[str, Any] | None:
    """Return customer dict or None if not found."""
    if not api_enabled():
        return None
    client = get_api()
    try:
        resp = client.get("/customers/by-phone", params={"phone": phone.strip()})
    except httpx.RequestError:
        return None
    if resp.status_code == 404:
        return None
    if resp.status_code >= 400:
        raise APIError(_parse_error(resp), resp.status_code)
    return resp.json()


def ensure_customer(phone: str, *, name: str | None = None) -> dict[str, Any]:
    """Find or create a customer for a WhatsApp phone number."""
    phone = phone.strip()
    existing = get_customer_by_phone(phone)
    if existing:
        return existing
    display = name or f"WhatsApp {phone[-4:]}"
    return create_customer(display, phone)


def create_customer(name: str, phone: str, *, preferred_language: str = "en") -> dict[str, Any]:
    return _request(
        "POST",
        "/customers",
        json={"name": name, "phone": phone, "preferred_language": preferred_language},
    )


# --- Legacy grocery (optional) ---


def list_products(*, page: int = 1, per_page: int = 100, active_only: bool = True) -> list[dict[str, Any]]:
    data = _request(
        "GET",
        "/products",
        params={"page": page, "per_page": per_page, "active_only": str(active_only).lower()},
    )
    return data.get("data", []) if isinstance(data, dict) else []


def check_stock(items: list[dict[str, Any]]) -> dict[str, Any]:
    return _request("POST", "/inventory/check", json={"items": items})


def create_whatsapp_order(
    items: list[dict[str, Any]],
    *,
    customer_id: str | None = None,
    notes: str | None = None,
) -> dict[str, Any]:
    body: dict[str, Any] = {"items": items, "source": "whatsapp"}
    if customer_id:
        body["customer_id"] = customer_id
    if notes:
        body["notes"] = notes
    return _request("POST", "/orders/whatsapp", json=body)
