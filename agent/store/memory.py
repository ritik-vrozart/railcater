from __future__ import annotations

import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any


@dataclass
class CartLine:
    product_id: str
    product_name: str
    unit: str
    quantity: int
    unit_price_cents: int

    @property
    def line_total_cents(self) -> int:
        return self.unit_price_cents * self.quantity


@dataclass
class Cart:
    user_id: str
    lines: list[CartLine] = field(default_factory=list)

    @property
    def total_cents(self) -> int:
        return sum(line.line_total_cents for line in self.lines)


@dataclass
class Order:
    id: str
    user_id: str
    status: str
    items: list[CartLine]
    total_cents: int
    payment_status: str
    payment_link: str | None
    created_at: datetime


# In-memory stores keyed by WhatsApp phone (user_id)
_carts: dict[str, Cart] = {}
_orders: dict[str, list[Order]] = {}
_payments: dict[str, dict[str, Any]] = {}


def get_cart(user_id: str) -> Cart:
    if user_id not in _carts:
        _carts[user_id] = Cart(user_id=user_id)
    return _carts[user_id]


def clear_cart(user_id: str) -> None:
    _carts[user_id] = Cart(user_id=user_id)


def add_order(user_id: str, items: list[CartLine], total_cents: int) -> Order:
    from config import settings

    order_id = str(uuid.uuid4())[:8]
    base = (settings.payment_link_base_url or "").strip().rstrip("/")
    if base:
        payment_link = f"{base}/order/{order_id}?amount={total_cents / 100:.2f}"
    else:
        payment_link = ""
    order = Order(
        id=order_id,
        user_id=user_id,
        status="confirmed",
        items=list(items),
        total_cents=total_cents,
        payment_status="pending",
        payment_link=payment_link,
        created_at=datetime.now(timezone.utc),
    )
    _orders.setdefault(user_id, []).append(order)
    _payments[order_id] = {"status": "pending", "link": payment_link, "amount_cents": total_cents}
    return order


def get_orders(user_id: str) -> list[Order]:
    return list(_orders.get(user_id, []))


def mark_payment_paid(order_id: str) -> dict[str, Any] | None:
    pay = _payments.get(order_id)
    if not pay:
        return None
    pay["status"] = "paid"
    for orders in _orders.values():
        for o in orders:
            if o.id == order_id:
                o.payment_status = "paid"
                o.status = "paid"
    return pay
