"""ADK tools — train food ordering via Go backend API."""

from __future__ import annotations

import re
from typing import Any

from services import api_client
from store.session import TrainCartLine, get_session

_current_user_id: str = "demo_user"


def set_current_user(user_id: str) -> None:
    global _current_user_id
    _current_user_id = user_id


def _uid() -> str:
    return _current_user_id


def _session():
    return get_session(_uid())


def ensure_whatsapp_customer(name: str | None = None) -> dict[str, Any]:
    """Link WhatsApp user id (phone) to a backend customer record."""
    sess = _session()
    if sess.customer_id:
        return {"status": "success", "customer_id": sess.customer_id}

    phone = _uid()
    try:
        customer = api_client.ensure_customer(phone, name=name)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    sess.customer_id = str(customer.get("id", ""))
    return {"status": "success", "customer_id": sess.customer_id}


def _inr(cents: int) -> float:
    return cents / 100


def _format_portion_row(item: dict[str, Any], portion: dict[str, Any]) -> str:
    veg = "🟢" if item.get("is_veg") else "🔴"
    price = _inr(int(portion.get("price_cents", 0)))
    stock = portion.get("stock_quantity", 0)
    return (
        f"{veg} {item.get('name')} ({portion.get('label', portion.get('portion'))}) "
        f"— ₹{price:.0f} · stock {stock} · id={portion.get('id')}"
    )


def lookup_train(number: str) -> dict[str, Any]:
    """
    Resolve train by number and auto-select pantry (vendor) for that train.

    Args:
        number: Train number e.g. 12951
    """
    number = re.sub(r"\D", "", number.strip())
    if not number:
        return {"status": "error", "message": "Train number is required."}

    try:
        data = api_client.lookup_train_by_number(number)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    from store.session import FlowStep

    train = data.get("train") or {}
    pantries = data.get("pantries") or []
    if not pantries:
        return {
            "status": "error",
            "message": f"Train {number} par abhi koi pantry nahi hai.",
        }

    pantry = pantries[0]
    if len(pantries) > 1:
        # Prefer explicitly linked pantry; first is fine for now
        pass

    sess = _session()
    sess.reset_journey()
    sess.train_number = train.get("number") or number
    sess.train_id = str(train.get("id", ""))
    sess.train_name = train.get("name", "")
    sess.vendor_id = str(pantry.get("id", ""))
    sess.vendor_name = pantry.get("name", "")
    sess.flow_step = FlowStep.AWAITING_NAME
    sess.touch_journey()

    return {
        "status": "success",
        "train_number": sess.train_number,
        "train_name": sess.train_name,
        "pantry_name": sess.vendor_name,
        "message": (
            f"Train {sess.train_number} {sess.train_name} — "
            f"Pantry: {sess.vendor_name}. Ab apna naam bhejein."
        ),
    }


def set_passenger_name(name: str) -> dict[str, Any]:
    """Save passenger name and ask for seat."""
    name = name.strip()
    if len(name) < 2:
        return {"status": "error", "message": "Please send your name (at least 2 characters)."}

    from store.session import FlowStep

    sess = _session()
    if not sess.train_number:
        return {"status": "error", "message": "Pehle train number bhejein."}

    sess.passenger_name = name
    sess.flow_step = FlowStep.AWAITING_SEAT
    sess.touch_journey()
    ensure_whatsapp_customer(name=name)
    return {
        "status": "success",
        "passenger_name": name,
        "message": f"Namaste {name}! Ab apna coach aur seat number bhejein.",
    }


def set_delivery_seat(coach: str, berth: str) -> dict[str, Any]:
    """Save coach / berth (bogie & seat) for delivery."""
    coach = coach.strip().upper()
    berth = berth.strip()
    if not coach or not berth:
        return {"status": "error", "message": "Coach and seat/berth are required."}

    from store.session import FlowStep

    sess = _session()
    if not sess.train_number:
        return {"status": "error", "message": "Pehle train number bhejein."}
    sess.coach = coach
    sess.berth = berth
    sess.flow_step = FlowStep.AWAITING_CATEGORY
    sess.touch_journey()
    return {
        "status": "success",
        "coach": coach,
        "berth": berth,
        "message": f"Seat {coach}/{berth} saved. Ab category choose karein.",
    }


def list_menu_categories() -> dict[str, Any]:
    """List menu categories for the train's pantry."""
    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "Pehle train number bhejein."}

    try:
        cats = api_client.list_menu_categories(sess.vendor_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    rows = [
        {
            "id": str(c.get("id", "")),
            "name": c.get("name", ""),
            "description": c.get("description", ""),
            "food_type": c.get("food_type", ""),
        }
        for c in cats
        if c.get("is_active", True)
    ]
    if not rows:
        return {"status": "error", "message": "Koi category nahi mili."}

    return {"status": "success", "categories": rows, "count": len(rows)}


def select_category(category_id: str) -> dict[str, Any]:
    """Pick a menu category before browsing items."""
    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "Pehle train number bhejein."}

    try:
        cats = api_client.list_menu_categories(sess.vendor_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    cat = next((c for c in cats if str(c.get("id")) == category_id.strip()), None)
    if not cat:
        return {"status": "error", "message": "Category not found."}

    from store.session import FlowStep

    sess.category_id = category_id.strip()
    sess.category_name = cat.get("name", "")
    sess.flow_step = FlowStep.ORDERING
    sess.touch_journey()
    return {
        "status": "success",
        "category_name": sess.category_name,
        "message": f"Category *{sess.category_name}* — ab items choose karein.",
    }


def _build_portions_index(
    items: list[dict[str, Any]],
    *,
    category_filter: str | None = None,
    veg_filter: str | None = None,
    item_query: str | None = None,
    limit: int | None = None,
) -> list[dict[str, Any]]:
    from services.food_intent import category_name_matches, score_item_match

    portions_index: list[dict[str, Any]] = []
    scored: list[tuple[float, dict[str, Any]]] = []

    for item in items:
        cat_name = str(item.get("category") or "")
        if category_filter and str(item.get("category_id") or "") != category_filter:
            continue
        is_veg = bool(item.get("is_veg", True))
        if veg_filter == "veg" and not is_veg:
            continue
        if veg_filter == "non_veg" and is_veg:
            continue

        for portion in item.get("portions") or []:
            if not portion.get("is_active", True):
                continue
            row = {
                "menu_portion_id": portion["id"],
                "menu_item_id": str(item.get("id", "")),
                "item_name": item.get("name"),
                "item_description": item.get("description") or "",
                "category_id": str(item.get("category_id") or ""),
                "category_name": cat_name,
                "portion_label": portion.get("label"),
                "price_inr": _inr(int(portion.get("price_cents", 0))),
                "stock": portion.get("stock_quantity", 0),
                "image_url": item.get("image_url"),
                "is_veg": is_veg,
            }
            if item_query:
                sc = score_item_match(
                    item_query,
                    str(item.get("name", "")),
                    str(item.get("description") or ""),
                )
                if sc < 0.3:
                    continue
                scored.append((sc, row))
            else:
                portions_index.append(row)

    if item_query and scored:
        scored.sort(key=lambda x: x[0], reverse=True)
        portions_index = [r for _, r in scored]

    if limit and len(portions_index) > limit:
        portions_index = portions_index[:limit]
    return portions_index


def search_menu(query: str, *, veg: str | None = None, limit: int = 10) -> dict[str, Any]:
    """
    Search full pantry menu by natural language (ignores current category).

    Pass veg only for meal-type filters ("veg" | "non_veg").
    Omit veg for water, drinks, snacks — they are not veg/non-veg categories.
    """
    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "Pehle train number bhejein."}

    try:
        items = api_client.get_vendor_menu(sess.vendor_id)
        cats = api_client.list_menu_categories(sess.vendor_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    from services.food_intent import FoodIntent, category_name_matches, parse_food_intent

    intent = parse_food_intent(query) or FoodIntent(item_query=query, category_hints=[], veg_preference=veg, quantity=None)
    # Only apply veg filter when caller passes veg (agent follows prompt rules).
    veg_filter = veg

    # Narrow by category hints
    filtered_items = items
    if intent.category_hints:
        filtered_items = [
            it
            for it in items
            if category_name_matches(intent.category_hints, str(it.get("category") or ""))
        ]
        if not filtered_items:
            filtered_items = items

    portions = _build_portions_index(
        filtered_items,
        veg_filter=veg_filter,
        item_query=intent.item_query,
        limit=limit,
    )

    if not portions and intent.category_hints:
        portions = _build_portions_index(
            items,
            veg_filter=veg_filter,
            item_query=intent.item_query,
            limit=limit,
        )

    return {
        "status": "success",
        "query": intent.item_query,
        "category_hints": intent.category_hints,
        "veg_preference": veg_filter,
        "count": len(portions),
        "portions": portions,
        "categories": [
            {"id": str(c["id"]), "name": c.get("name"), "food_type": c.get("food_type")}
            for c in cats
            if c.get("is_active", True)
        ],
    }


def browse_menu(
    *,
    category_id: str | None = None,
    item_query: str | None = None,
    veg_filter: str | None = None,
) -> dict[str, Any]:
    """List menu items and portions for the selected vendor (optional category filter)."""
    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "No vendor selected. Call select_vendor first."}

    cat_filter = category_id or sess.category_id

    try:
        items = api_client.get_vendor_menu(sess.vendor_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    if cat_filter:
        items = [it for it in items if str(it.get("category_id") or "") == cat_filter]

    portions_index = _build_portions_index(
        items,
        item_query=item_query,
        veg_filter=veg_filter,
    )
    lines = [
        _format_portion_row(
            {"name": p["item_name"], "is_veg": p["is_veg"], "description": p.get("item_description")},
            {"label": p["portion_label"], "price_cents": int(p["price_inr"] * 100), "stock_quantity": p["stock"], "id": p["menu_portion_id"]},
        )
        for p in portions_index
    ]

    if not portions_index:
        return {"status": "success", "count": 0, "message": "Menu is empty for this vendor."}

    return {
        "status": "success",
        "vendor_name": sess.vendor_name,
        "count": len(portions_index),
        "portions": portions_index,
        "formatted_menu": "\n".join(lines),
        "message": f"Menu from {sess.vendor_name} ({len(portions_index)} options). Use add_meal_to_cart with menu_portion_id.",
    }


def add_meal_to_cart(menu_portion_id: str, quantity: int = 1) -> dict[str, Any]:
    """
    Add a menu portion to the cart.

    Args:
        menu_portion_id: UUID from browse_menu portions list.
        quantity: Number of servings (default 1).
    """
    if quantity < 1:
        return {"status": "error", "message": "Quantity must be at least 1."}

    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "Select a vendor first."}

    try:
        items = api_client.get_vendor_menu(sess.vendor_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    portion_info: tuple[dict, dict] | None = None
    for item in items:
        for portion in item.get("portions") or []:
            if str(portion.get("id")) == menu_portion_id.strip():
                portion_info = (item, portion)
                break
        if portion_info:
            break

    if not portion_info:
        return {"status": "error", "message": f"Portion {menu_portion_id} not found on menu."}

    item, portion = portion_info
    stock = int(portion.get("stock_quantity", 0))
    if stock < quantity:
        return {
            "status": "error",
            "message": f"Only {stock} available for {item.get('name')} ({portion.get('label')}).",
        }

    pid = str(portion["id"])
    existing = next((l for l in sess.cart_lines if l.menu_portion_id == pid), None)
    if existing:
        if existing.quantity + quantity > stock:
            return {"status": "error", "message": f"Cannot add more than {stock} in stock."}
        existing.quantity += quantity
    else:
        sess.cart_lines.append(
            TrainCartLine(
                menu_portion_id=pid,
                item_name=str(item.get("name", "")),
                portion_label=str(portion.get("label", portion.get("portion", ""))),
                quantity=quantity,
                unit_price_cents=int(portion.get("price_cents", 0)),
            )
        )

    sess.touch_journey()
    return {"status": "success", "message": f"Added {quantity}x {item.get('name')}.", **view_train_cart()}


def view_train_cart() -> dict[str, Any]:
    """Show the current train food cart."""
    sess = _session()
    if not sess.cart_lines:
        return {"status": "success", "empty": True, "message": "Cart is empty."}

    lines = [
        f"  {l.item_name} ({l.portion_label}) x{l.quantity} = ₹{_inr(l.line_total_cents):.2f}"
        for l in sess.cart_lines
    ]
    return {
        "status": "success",
        "empty": False,
        "cart_summary": "\n".join(lines) + f"\n\nTotal: ₹{_inr(sess.cart_total_cents):.2f}",
        "cart_total_inr": _inr(sess.cart_total_cents),
        "train": sess.train_number,
        "seat": f"{sess.coach}/{sess.berth}" if sess.coach and sess.berth else None,
        "vendor": sess.vendor_name,
    }


def remove_from_train_cart(menu_portion_id: str) -> dict[str, Any]:
    """Remove a line from the train food cart."""
    sess = _session()
    before = len(sess.cart_lines)
    sess.cart_lines = [l for l in sess.cart_lines if l.menu_portion_id != menu_portion_id.strip()]
    if len(sess.cart_lines) == before:
        return {"status": "error", "message": "Item not in cart."}
    return {"status": "success", "message": "Removed.", **view_train_cart()}


def clear_train_cart() -> dict[str, Any]:
    """Empty the train food cart."""
    _session().clear_cart()
    return {"status": "success", "message": "Cart cleared."}


def place_train_order(notes: str | None = None) -> dict[str, Any]:
    """
    Place train food order via backend API.

    Uses WhatsApp flow (train number + name + seat) when PNR is not set.
    """
    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "Complete train → name → seat → category → cart before checkout."}
    if not sess.cart_lines:
        return {"status": "error", "message": "Cart is empty."}

    items = [{"menu_portion_id": l.menu_portion_id, "quantity": l.quantity} for l in sess.cart_lines]
    lookup = sess.pnr_lookup or {}
    coach = (sess.coach or lookup.get("coach") or "").strip()
    berth = (sess.berth or lookup.get("berth") or "").strip()
    if not coach or not berth:
        return {"status": "error", "message": "Delivery seat (coach/berth) not set."}

    customer_id = sess.customer_id
    if not customer_id:
        cust = ensure_whatsapp_customer(name=sess.passenger_name)
        if cust.get("status") == "error":
            return cust
        customer_id = cust.get("customer_id")

    try:
        if sess.train_number and not sess.pnr:
            if not sess.passenger_name:
                return {"status": "error", "message": "Passenger name is required."}
            order = api_client.create_whatsapp_train_order(
                train_number=sess.train_number,
                train_id=sess.train_id,
                vendor_id=sess.vendor_id,
                passenger_name=sess.passenger_name,
                coach=coach,
                berth=berth,
                items=items,
                customer_id=customer_id,
                notes=notes or "Ordered via WhatsApp",
            )
        else:
            return {"status": "error", "message": "Train number complete karein."}
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    from store.session import FlowStep

    order_id = str(order.get("id", ""))
    sess.last_order_id = order_id
    sess.clear_cart()
    sess.flow_step = FlowStep.IDLE
    sess.touch_journey()  # keep train/name/seat for "Add more" within TTL

    total = _inr(int(order.get("total_cents", 0)))
    dw_start = order.get("delivery_window_start", "")
    dw_end = order.get("delivery_window_end", "")

    item_lines = []
    for it in order.get("items") or []:
        item_lines.append(
            f"  {it.get('product_name', 'Item')} x{it.get('quantity')} = ₹{_inr(int(it.get('line_total_cents', 0))):.2f}"
        )

    seat = f"{sess.coach}/{sess.berth}" if sess.coach and sess.berth else ""
    train = sess.train_number or order.get("train_number") or ""
    return {
        "status": "success",
        "order_id": order_id,
        "order_status": order.get("status"),
        "total_inr": total,
        "train": train,
        "seat": seat,
        "vendor": order.get("vendor_name") or sess.vendor_name,
        "payment_status": order.get("payment_status", "pending"),
        "delivery_window": f"{dw_start} – {dw_end}" if dw_start else None,
        "items_summary": "\n".join(item_lines) if item_lines else None,
        "message": (
            f"Order placed! ID {order_id[:8]}… Total ₹{total:.2f}. "
            f"Train {train} seat {seat}. Pantry will confirm delivery time. Status: {order.get('status')}."
        ),
    }


def get_train_order_status(order_id: str) -> dict[str, Any]:
    """Fetch order status from backend by order UUID."""
    try:
        order = api_client.get_order(order_id.strip())
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    return {
        "status": "success",
        "order_id": str(order.get("id")),
        "order_status": order.get("status"),
        "total_inr": _inr(int(order.get("total_cents", 0))),
        "train": order.get("train_number"),
        "seat": f"{order.get('coach')}/{order.get('berth')}" if order.get("coach") else None,
        "payment_status": order.get("payment_status"),
        "payment_method": order.get("payment_method"),
        "vendor": order.get("vendor_name"),
        "delivery_window_start": order.get("delivery_window_start"),
        "delivery_window_end": order.get("delivery_window_end"),
    }


def get_recent_orders() -> dict[str, Any]:
    """List recent orders from backend for this WhatsApp user (last 5)."""
    cust = ensure_whatsapp_customer()
    if cust.get("status") == "error":
        return cust

    try:
        orders = api_client.list_orders(per_page=5, customer_id=cust.get("customer_id"))
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    if not orders:
        return {"status": "success", "count": 0, "orders": [], "message": "No orders found."}

    rows = []
    for o in orders[:5]:
        rows.append(
            {
                "order_id": str(o.get("id")),
                "status": o.get("status"),
                "total_inr": _inr(int(o.get("total_cents", 0))),
                "train": o.get("train_number"),
                "payment_status": o.get("payment_status"),
                "source": o.get("source"),
            }
        )
    return {"status": "success", "count": len(rows), "orders": rows}
