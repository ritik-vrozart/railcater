"""ADK tool functions — operate on in-memory dummy catalog & carts."""

from __future__ import annotations

from typing import Any

from data.products import CATALOG, format_product_line, get_product
from store.memory import CartLine, add_order, clear_cart, get_cart, get_orders, mark_payment_paid

# Set by agent runner before each invocation (WhatsApp phone / test user id)
_current_user_id: str = "demo_user"


def set_current_user(user_id: str) -> None:
    global _current_user_id
    _current_user_id = user_id


def _uid() -> str:
    return _current_user_id


def list_all_products(category: str | None = None) -> dict[str, Any]:
    """
    List all products in the store. Optionally filter by category
    (oil, rice, dal, dairy, essentials, household, snacks, beverages, vegetables, flour, bakery).

    Returns:
        status, products (list of id, name, unit, price_inr, stock, category), count
    """
    items = CATALOG
    if category:
        cat = category.strip().lower()
        items = [p for p in items if p.category == cat or cat in p.name.lower()]
    rows = [
        {
            "id": p.id,
            "sku": p.sku,
            "name": p.name,
            "unit": p.unit,
            "price_inr": p.price_inr,
            "stock": p.quantity,
            "category": p.category,
        }
        for p in items
    ]
    return {"status": "success", "count": len(rows), "products": rows}


def search_products(query: str) -> dict[str, Any]:
    """
    Search products by natural language, e.g. '1 litre oil', 'basmati rice', 'maggi'.

    Args:
        query: What the customer is looking for.

    Returns:
        status, matches (list), formatted_list (human-readable lines)
    """
    q = query.strip().lower()
    if not q:
        return {"status": "error", "message": "Query cannot be empty"}

    tokens = [t for t in q.replace(",", " ").split() if len(t) > 1]
    matches = []
    for p in CATALOG:
        hay = f"{p.name} {p.description} {p.unit} {p.category} {p.sku}".lower()
        score = sum(1 for t in tokens if t in hay)
        if score > 0:
            matches.append((score, p))
    matches.sort(key=lambda x: (-x[0], x[1].name))
    products = [m[1] for m in matches[:8]]

    if not products:
        return {
            "status": "success",
            "count": 0,
            "matches": [],
            "formatted_list": "No products found. Try 'oil', 'rice', or 'list all products'.",
        }

    lines = [format_product_line(p) for p in products]
    return {
        "status": "success",
        "count": len(products),
        "matches": [
            {"id": p.id, "name": p.name, "unit": p.unit, "price_inr": p.price_inr, "stock": p.quantity}
            for p in products
        ],
        "formatted_list": "\n".join(lines),
    }


def add_to_cart(product_id: str, quantity: int = 1) -> dict[str, Any]:
    """
    Add a product to the customer's cart.

    Args:
        product_id: Product id from search/list (e.g. p1).
        quantity: How many units (default 1).

    Returns:
        status, cart_summary, message
    """
    if quantity < 1:
        return {"status": "error", "message": "Quantity must be at least 1"}

    product = get_product(product_id.strip())
    if not product:
        return {"status": "error", "message": f"Product {product_id} not found. Use search_products first."}
    if product.quantity < quantity:
        return {
            "status": "error",
            "message": f"Only {product.quantity} units of {product.name} in stock.",
        }

    cart = get_cart(_uid())
    existing = next((l for l in cart.lines if l.product_id == product.id), None)
    if existing:
        existing.quantity += quantity
    else:
        cart.lines.append(
            CartLine(
                product_id=product.id,
                product_name=product.name,
                unit=product.unit,
                quantity=quantity,
                unit_price_cents=product.price_cents,
            )
        )

    total = cart.total_cents / 100
    lines = [
        f"  {l.product_name} x{l.quantity} ({l.unit}) = ₹{l.line_total_cents / 100:.2f}"
        for l in cart.lines
    ]
    return {
        "status": "success",
        "message": f"Added {quantity}x {product.name} to cart.",
        "cart_summary": "\n".join(lines) + f"\n\nCart total: ₹{total:.2f}",
        "cart_total_inr": total,
        "item_count": sum(l.quantity for l in cart.lines),
    }


def view_cart() -> dict[str, Any]:
    """Show the current cart for this customer."""
    cart = get_cart(_uid())
    if not cart.lines:
        return {"status": "success", "empty": True, "message": "Your cart is empty."}
    lines = [
        f"  {l.product_name} x{l.quantity} ({l.unit}) = ₹{l.line_total_cents / 100:.2f}"
        for l in cart.lines
    ]
    return {
        "status": "success",
        "empty": False,
        "cart_summary": "\n".join(lines) + f"\n\nTotal: ₹{cart.total_cents / 100:.2f}",
        "cart_total_inr": cart.total_cents / 100,
    }


def remove_from_cart(product_id: str) -> dict[str, Any]:
    """Remove a product line from the cart by product id."""
    cart = get_cart(_uid())
    before = len(cart.lines)
    cart.lines = [l for l in cart.lines if l.product_id != product_id.strip()]
    if len(cart.lines) == before:
        return {"status": "error", "message": "Product not in cart."}
    return {"status": "success", "message": "Item removed.", **view_cart()}


def clear_customer_cart() -> dict[str, Any]:
    """Empty the customer's cart."""
    clear_cart(_uid())
    return {"status": "success", "message": "Cart cleared."}


def place_order(notes: str | None = None) -> dict[str, Any]:
    """
    Place order from current cart and generate a dummy payment link (UPI-style).

    Args:
        notes: Optional delivery notes.

    Returns:
        status, order_id, total_inr, payment_link, items
    """
    cart = get_cart(_uid())
    if not cart.lines:
        return {"status": "error", "message": "Cart is empty. Add items before checkout."}

    order = add_order(_uid(), cart.lines, cart.total_cents)
    clear_cart(_uid())

    item_lines = [
        f"  {i.product_name} x{i.quantity} = ₹{i.line_total_cents / 100:.2f}" for i in order.items
    ]
    return {
        "status": "success",
        "order_id": order.id,
        "total_inr": order.total_cents / 100,
        "payment_status": order.payment_status,
        "payment_link": order.payment_link,
        "notes": notes,
        "items_summary": "\n".join(item_lines),
        "message": (
            f"Order #{order.id} placed! Total ₹{order.total_cents / 100:.2f}. "
            f"Pay here: {order.payment_link}"
        ),
    }


def get_my_orders() -> dict[str, Any]:
    """List all orders for this customer."""
    orders = get_orders(_uid())
    if not orders:
        return {"status": "success", "count": 0, "orders": [], "message": "No orders yet."}
    rows = []
    for o in reversed(orders[-5:]):
        rows.append(
            {
                "order_id": o.id,
                "status": o.status,
                "total_inr": o.total_cents / 100,
                "payment_status": o.payment_status,
                "payment_link": o.payment_link,
                "created_at": o.created_at.isoformat(),
            }
        )
    return {"status": "success", "count": len(rows), "orders": rows}


def confirm_payment(order_id: str) -> dict[str, Any]:
    """
    Mark an order as paid (simulates customer completing UPI/card payment).

    Args:
        order_id: Order id from place_order.
    """
    result = mark_payment_paid(order_id.strip())
    if not result:
        return {"status": "error", "message": f"Order {order_id} not found."}
    return {
        "status": "success",
        "order_id": order_id,
        "payment_status": "paid",
        "message": f"Payment received for order #{order_id}. Thank you!",
    }
