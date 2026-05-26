"""Internal callbacks from Go API (delivery WhatsApp notifications)."""

from __future__ import annotations

import logging
from datetime import datetime

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from config import settings
from services.whatsapp import get_whatsapp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/internal/notify", tags=["internal"])


class DeliveryNotifyRequest(BaseModel):
    secret: str
    order_id: str
    phone: str
    passenger_name: str | None = None
    station_name: str | None = None
    station_code: str | None = None
    coach: str | None = None
    berth: str | None = None
    delivery_window_start: str
    delivery_window_end: str = ""


def _format_time(iso: str) -> str:
    if not iso:
        return ""
    try:
        dt = datetime.fromisoformat(iso.replace("Z", "+00:00"))
        return dt.strftime("%I:%M %p on %d %b")
    except (ValueError, TypeError):
        return iso


def _normalize_phone(phone: str) -> str:
    return "".join(c for c in phone.strip() if c.isdigit())


class OrderStatusNotifyRequest(BaseModel):
    secret: str
    order_id: str
    phone: str
    passenger_name: str | None = None
    status: str
    train_number: str | None = None
    station_name: str | None = None


_STATUS_MESSAGES = {
    "preparing": ("👨‍🍳 *Order update*", "Your food is being prepared at the pantry."),
    "ready": ("✅ *Order update*", "Your order is ready and will be dispatched soon."),
    "dispatched": ("🚚 *Dispatched*", "Your order has left the pantry and is on the way to your seat."),
    "delivered": ("🍱 *Delivered*", "Your order has been delivered. Enjoy your meal!"),
}


@router.post("/order-status")
async def notify_order_status(req: OrderStatusNotifyRequest):
    expected = settings.agent_notify_secret
    if not expected or req.secret != expected:
        raise HTTPException(status_code=403, detail="Forbidden")

    wa = get_whatsapp()
    if not wa.configured:
        logger.warning("WhatsApp not configured; skip status notify for order %s", req.order_id)
        return {"status": "skipped", "reason": "whatsapp_not_configured"}

    to = _normalize_phone(req.phone)
    if not to:
        raise HTTPException(status_code=400, detail="invalid phone")

    title, detail = _STATUS_MESSAGES.get(
        req.status,
        ("📦 *Order update*", f"Status: *{req.status}*"),
    )
    name = req.passenger_name or "Customer"
    train = f" on train *{req.train_number}*" if req.train_number else ""
    station = f" at *{req.station_name}*" if req.station_name else ""

    body = (
        f"{title}\n\n"
        f"Hi {name},\n"
        f"Order `{req.order_id[:8]}…`{train}{station}.\n"
        f"{detail}\n\n"
        f"Thank you for ordering with RailCater!"
    )

    await wa.send_text(to, body)
    logger.info("Sent order status %s to %s for order %s", req.status, to, req.order_id)
    return {"status": "sent", "to": to}


@router.post("/delivery")
async def notify_delivery(req: DeliveryNotifyRequest):
    expected = settings.agent_notify_secret
    if not expected or req.secret != expected:
        raise HTTPException(status_code=403, detail="Forbidden")

    wa = get_whatsapp()
    if not wa.configured:
        logger.warning("WhatsApp not configured; skip delivery notify for order %s", req.order_id)
        return {"status": "skipped", "reason": "whatsapp_not_configured"}

    to = _normalize_phone(req.phone)
    if not to:
        raise HTTPException(status_code=400, detail="invalid phone")

    station = req.station_name or req.station_code or "your station"
    start_fmt = _format_time(req.delivery_window_start)
    end_fmt = _format_time(req.delivery_window_end) if req.delivery_window_end else ""

    seat = ""
    if req.coach and req.berth:
        seat = f"\n🪑 Seat: *{req.coach} / {req.berth}*"
    elif req.coach:
        seat = f"\n🪑 Coach: *{req.coach}*"

    name = req.passenger_name or "Customer"
    time_line = f"by *{start_fmt}*" if start_fmt else "soon"
    if end_fmt and end_fmt != start_fmt:
        time_line = f"between *{start_fmt}* and *{end_fmt}*"

    body = (
        f"🚂 *Delivery update*\n\n"
        f"Hi {name},\n"
        f"Your train food order `{req.order_id[:8]}…` will be delivered at "
        f"*{station}* {time_line}.{seat}\n\n"
        f"Please be at your seat. Thank you! 🍱"
    )

    await wa.send_text(to, body)
    logger.info("Sent delivery notify to %s for order %s", to, req.order_id)
    return {"status": "sent", "to": to}
