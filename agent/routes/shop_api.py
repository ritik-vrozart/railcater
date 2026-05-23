"""REST API for the mobile shop web app (same in-memory cart as WhatsApp)."""

from __future__ import annotations

import logging

from fastapi import APIRouter, BackgroundTasks, HTTPException, Query
from pydantic import BaseModel, Field

from config import settings
from data.products import CATALOG, get_product
from store.memory import clear_cart, get_cart, get_orders, mark_payment_paid
from services.whatsapp import get_whatsapp
from tools import shop_tools
from tools.shop_tools import add_to_cart, place_order, remove_from_cart

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/api/shop", tags=["shop-web"])


@router.get("/config")
def api_config():
    num = settings.whatsapp_wa_me_number.strip()
    return {
        "wa_me_number": num,
        "wa_back_url": f"https://wa.me/{num}?text=menu" if num else None,
        "public_shop_url": settings.public_base_url.rstrip("/") + "/shop",
    }


async def _notify_order_on_whatsapp(user_id: str, result: dict) -> None:
    wa = get_whatsapp()
    if not wa.configured:
        return
    msg = (
        f"✅ Order #{result['order_id']} placed from shop!\n"
        f"Total: ₹{result['total_inr']:.2f}\n"
        f"Pay: {result.get('payment_link', '')}\n\n"
        "You're back in chat — tap *menu* or *cart* anytime."
    )
    try:
        await wa.send_text(user_id, msg)
        await wa.send_buttons(
            user_id,
            "Continue in chat:",
            [
                ("menu_home", "Menu"),
                ("menu_cart", "My cart"),
                ("menu_browse", "Browse"),
            ],
        )
    except Exception:
        logger.exception("Failed to notify %s on WhatsApp after checkout", user_id)


def _uid(user_id: str) -> None:
    shop_tools.set_current_user(user_id)


@router.get("/products")
def api_products(category: str | None = None):
    items = CATALOG if not category else [p for p in CATALOG if p.category == category]
    return {
        "products": [
            {
                "id": p.id,
                "name": p.name,
                "unit": p.unit,
                "price_inr": p.price_inr,
                "stock": p.quantity,
                "category": p.category,
                "description": p.description,
            }
            for p in items
        ],
        "categories": sorted({p.category for p in CATALOG}),
    }


@router.get("/cart")
def api_get_cart(user_id: str = Query(..., min_length=1)):
    _uid(user_id)
    cart = get_cart(user_id)
    return {
        "lines": [
            {
                "product_id": l.product_id,
                "name": l.product_name,
                "unit": l.unit,
                "quantity": l.quantity,
                "unit_price_inr": l.unit_price_cents / 100,
                "line_total_inr": l.line_total_cents / 100,
            }
            for l in cart.lines
        ],
        "total_inr": cart.total_cents / 100,
        "item_count": sum(l.quantity for l in cart.lines),
    }


class AddCartBody(BaseModel):
    user_id: str
    product_id: str
    quantity: int = Field(default=1, ge=1, le=99)


@router.post("/cart/add")
def api_add_cart(body: AddCartBody):
    _uid(body.user_id)
    result = add_to_cart(body.product_id, body.quantity)
    if result.get("status") == "error":
        raise HTTPException(status_code=400, detail=result.get("message"))
    return result


class UpdateCartBody(BaseModel):
    user_id: str
    product_id: str
    quantity: int = Field(ge=0, le=99)


@router.put("/cart/update")
def api_update_cart(body: UpdateCartBody):
    _uid(body.user_id)
    if body.quantity == 0:
        return remove_from_cart(body.product_id)
    cart = get_cart(body.user_id)
    line = next((l for l in cart.lines if l.product_id == body.product_id), None)
    if line:
        p = get_product(body.product_id)
        if p and p.quantity < body.quantity:
            raise HTTPException(status_code=400, detail=f"Only {p.quantity} in stock")
        line.quantity = body.quantity
        return {"status": "success", "total_inr": cart.total_cents / 100}
    return add_to_cart(body.product_id, body.quantity)


@router.delete("/cart/clear")
def api_clear_cart(user_id: str = Query(...)):
    _uid(user_id)
    clear_cart(user_id)
    return {"status": "success"}


@router.post("/checkout")
async def api_checkout(
    background_tasks: BackgroundTasks,
    user_id: str = Query(...),
):
    _uid(user_id)
    result = place_order()
    if result.get("status") == "error":
        raise HTTPException(status_code=400, detail=result.get("message"))
    background_tasks.add_task(_notify_order_on_whatsapp, user_id, result)
    return result


@router.get("/orders")
def api_orders(user_id: str = Query(...)):
    _uid(user_id)
    orders = get_orders(user_id)
    return {
        "orders": [
            {
                "order_id": o.id,
                "total_inr": o.total_cents / 100,
                "status": o.status,
                "payment_status": o.payment_status,
                "payment_link": o.payment_link,
            }
            for o in reversed(orders)
        ]
    }


@router.post("/orders/{order_id}/pay")
def api_confirm_pay(order_id: str, user_id: str = Query(...)):
    _uid(user_id)
    r = mark_payment_paid(order_id)
    if not r:
        raise HTTPException(status_code=404, detail="Order not found")
    return {"status": "success", "payment_status": "paid"}
