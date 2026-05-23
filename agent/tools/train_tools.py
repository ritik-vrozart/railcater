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


def get_stops_with_vendors() -> dict[str, Any]:
    """For each stop on the loaded PNR route, list vendors serving that station."""
    sess = _session()
    if not sess.pnr_lookup:
        return {"status": "error", "message": "No PNR loaded. Call lookup_pnr first."}

    stops_out: list[dict[str, Any]] = []
    for s in sess.pnr_lookup.get("available_stops") or []:
        sid = str(s.get("station_id", ""))
        try:
            vendors = api_client.list_station_vendors(sid) if sid else []
        except api_client.APIError:
            vendors = []
        stops_out.append(
            {
                "station_id": sid,
                "station_code": s.get("station_code"),
                "station_name": s.get("station_name"),
                "stop_order": s.get("stop_order"),
                "vendor_count": len(vendors),
                "vendors": [{"id": v["id"], "name": v.get("name"), "code": v.get("code")} for v in vendors],
            }
        )

    return {"status": "success", "pnr": sess.pnr, "stops": stops_out}


def lookup_pnr(pnr: str) -> dict[str, Any]:
    """
    Look up a 10-digit railway PNR and load journey details (train, passenger, stops).

    Args:
        pnr: 10-digit PNR number.

    Returns:
        status, passenger, train, from/to stations, available_stops, message
    """
    pnr = re.sub(r"\D", "", pnr.strip())
    if len(pnr) != 10:
        return {"status": "error", "message": "PNR must be exactly 10 digits."}

    try:
        data = api_client.lookup_pnr(pnr)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    from store.session import FlowStep

    sess = _session()
    sess.reset_journey()
    sess.pnr = pnr
    sess.pnr_lookup = data
    sess.awaiting_pnr = False
    sess.flow_step = FlowStep.AWAITING_STATION

    train = data.get("train", {})
    stops = data.get("available_stops", [])
    stop_rows = [
        {
            "station_id": s.get("station_id"),
            "code": s.get("station_code"),
            "name": s.get("station_name"),
            "stop_order": s.get("stop_order"),
        }
        for s in stops
    ]

    return {
        "status": "success",
        "pnr": pnr,
        "passenger_name": data.get("passenger_name"),
        "coach": data.get("coach"),
        "berth": data.get("berth"),
        "journey_date": data.get("journey_date"),
        "train_number": train.get("number"),
        "train_name": train.get("name"),
        "from_station": data.get("from_station", {}).get("name"),
        "to_station": data.get("to_station", {}).get("name"),
        "available_stops": stop_rows,
        "message": (
            f"PNR {pnr} — {data.get('passenger_name')} on {train.get('number')} {train.get('name')}. "
            f"{len(stop_rows)} delivery stop(s) available. Use select_delivery_station next."
        ),
    }


def select_delivery_station(station_id: str) -> dict[str, Any]:
    """
    Choose delivery station for the loaded PNR and validate delivery window.

    Args:
        station_id: UUID of the station from available_stops.
    """
    sess = _session()
    if not sess.pnr:
        return {"status": "error", "message": "No PNR loaded. Call lookup_pnr first."}

    try:
        result = api_client.validate_delivery(sess.pnr, station_id.strip())
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    dw = result.get("delivery_window", {})
    if not dw.get("feasible", True):
        return {
            "status": "error",
            "message": dw.get("feasibility_message", "Delivery not feasible at this station."),
        }

    from store.session import FlowStep

    sess.station_id = station_id.strip()
    sess.station_name = dw.get("station_name") or dw.get("station_code", "")
    sess.delivery_window = dw
    sess.vendor_id = None
    sess.vendor_name = None
    sess.clear_cart()
    sess.flow_step = FlowStep.AWAITING_VENDOR

    return {
        "status": "success",
        "station_id": sess.station_id,
        "station_name": sess.station_name,
        "delivery_window_start": dw.get("delivery_window_start"),
        "delivery_window_end": dw.get("delivery_window_end"),
        "message": f"Delivery at {sess.station_name} is available. Call list_vendors_at_station next.",
    }


def list_vendors_at_station() -> dict[str, Any]:
    """List food vendors serving the selected delivery station."""
    sess = _session()
    if not sess.station_id:
        return {"status": "error", "message": "No station selected. Call select_delivery_station first."}

    try:
        vendors = api_client.list_station_vendors(sess.station_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    if not vendors:
        return {
            "status": "success",
            "count": 0,
            "vendors": [],
            "message": "No vendors at this station yet.",
        }

    rows = [{"id": v["id"], "name": v["name"], "code": v.get("code")} for v in vendors]
    lines = [f"  {v['name']} (id={v['id']})" for v in vendors]
    return {
        "status": "success",
        "count": len(rows),
        "vendors": rows,
        "formatted_list": "\n".join(lines),
        "message": f"{len(rows)} vendor(s) at {sess.station_name}. Use select_vendor with vendor id.",
    }


def set_delivery_seat(coach: str, berth: str) -> dict[str, Any]:
    """Save coach / berth (bogie & seat) for delivery."""
    coach = coach.strip().upper()
    berth = berth.strip()
    if not coach or not berth:
        return {"status": "error", "message": "Coach and seat/berth are required."}

    from store.session import FlowStep

    sess = _session()
    sess.coach = coach
    sess.berth = berth
    sess.flow_step = FlowStep.ORDERING
    return {
        "status": "success",
        "coach": coach,
        "berth": berth,
        "message": f"Delivery seat set to {coach} / {berth}.",
    }


def select_vendor(vendor_id: str) -> dict[str, Any]:
    """Select vendor for ordering; loads their menu."""
    sess = _session()
    if not sess.station_id:
        return {"status": "error", "message": "Select a delivery station first."}

    try:
        vendors = api_client.list_station_vendors(sess.station_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    vendor = next((v for v in vendors if str(v.get("id")) == vendor_id.strip()), None)
    if not vendor:
        return {"status": "error", "message": f"Vendor {vendor_id} not found at this station."}

    from store.session import FlowStep

    sess.vendor_id = vendor_id.strip()
    sess.vendor_name = vendor.get("name", "")
    sess.clear_cart()
    sess.flow_step = FlowStep.AWAITING_SEAT

    return {
        "status": "success",
        "vendor_id": sess.vendor_id,
        "vendor_name": sess.vendor_name,
        "message": "Vendor selected. Confirm delivery seat (coach/berth) next.",
    }


def browse_menu() -> dict[str, Any]:
    """List menu items and portions for the selected vendor."""
    sess = _session()
    if not sess.vendor_id:
        return {"status": "error", "message": "No vendor selected. Call select_vendor first."}

    try:
        items = api_client.get_vendor_menu(sess.vendor_id)
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    lines = []
    portions_index: list[dict[str, Any]] = []
    for item in items:
        for portion in item.get("portions") or []:
            if not portion.get("is_active", True):
                continue
            lines.append(_format_portion_row(item, portion))
            portions_index.append(
                {
                    "menu_portion_id": portion["id"],
                    "item_name": item.get("name"),
                    "portion_label": portion.get("label"),
                    "price_inr": _inr(int(portion.get("price_cents", 0))),
                    "stock": portion.get("stock_quantity", 0),
                }
            )

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
        "pnr": sess.pnr,
        "station": sess.station_name,
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
    Place train food order via backend API (deducts portion stock, validates PNR/delivery).

    Args:
        notes: Optional delivery notes (coach/berth seat details, etc.).
    """
    sess = _session()
    if not sess.pnr or not sess.station_id or not sess.vendor_id:
        return {"status": "error", "message": "Complete PNR → station → vendor → cart before checkout."}
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
        cust = ensure_whatsapp_customer()
        if cust.get("status") == "error":
            return cust
        customer_id = cust.get("customer_id")

    try:
        order = api_client.create_train_order(
            pnr=sess.pnr,
            station_id=sess.station_id,
            vendor_id=sess.vendor_id,
            items=items,
            customer_id=customer_id,
            coach=coach,
            berth=berth,
            notes=notes or "Ordered via WhatsApp",
        )
    except api_client.APIError as exc:
        return {"status": "error", "message": str(exc)}

    from store.session import FlowStep

    order_id = str(order.get("id", ""))
    sess.last_order_id = order_id
    sess.clear_cart()
    sess.flow_step = FlowStep.IDLE

    total = _inr(int(order.get("total_cents", 0)))
    dw_start = order.get("delivery_window_start", "")
    dw_end = order.get("delivery_window_end", "")

    item_lines = []
    for it in order.get("items") or []:
        item_lines.append(
            f"  {it.get('product_name', 'Item')} x{it.get('quantity')} = ₹{_inr(int(it.get('line_total_cents', 0))):.2f}"
        )

    return {
        "status": "success",
        "order_id": order_id,
        "order_status": order.get("status"),
        "total_inr": total,
        "pnr": sess.pnr,
        "station": order.get("station_name") or sess.station_name,
        "vendor": order.get("vendor_name") or sess.vendor_name,
        "delivery_window": f"{dw_start} – {dw_end}" if dw_start else None,
        "items_summary": "\n".join(item_lines) if item_lines else None,
        "message": (
            f"Order placed! ID {order_id[:8]}… Total ₹{total:.2f}. "
            f"Delivery at {sess.station_name} between scheduled window. Status: {order.get('status')}."
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
        "pnr": order.get("pnr"),
        "station": order.get("station_name"),
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
                "pnr": o.get("pnr"),
                "station": o.get("station_name"),
                "source": o.get("source"),
            }
        )
    return {"status": "success", "count": len(rows), "orders": rows}
