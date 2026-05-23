"""
WhatsApp guided flow for train food ordering.

Step 1: PNR
Step 2: Delivery station (with vendors preview per stop)
Step 3: Vendor at that station
Step 4: Menu → cart → checkout
"""

from __future__ import annotations

import logging
import re
from datetime import datetime

from services import api_client
from services.whatsapp import WhatsAppClient, _clip
from store.session import FlowStep, get_session
from tools import train_tools

logger = logging.getLogger(__name__)

MENU_HOME = "menu_home"
MENU_TRAIN_ORDER = "menu_train"
MENU_TRAIN_CART = "menu_train_cart"
MENU_TRAIN_CHECKOUT = "train_checkout"
MENU_ORDERS = "menu_orders"
MENU_PNR_PROMPT = "menu_pnr"
MENU_CHANGE_PNR = "menu_change_pnr"

STATION_PREFIX = "st_"
VENDOR_PREFIX = "ven_"
PORTION_PREFIX = "mp_"
TRAIN_ADD1_PREFIX = "tadd1_"
TRAIN_ADD2_PREFIX = "tadd2_"
SEAT_USE_PNR = "seat_use_pnr"
SEAT_MANUAL = "seat_manual"

PNR_PATTERN = re.compile(r"\b(\d{10})\b")
SEAT_PATTERN = re.compile(r"(?i)([A-Z]\d+)\s*[/\s,\-]+\s*(\d+)")

MENU_TRIGGERS = {
    "hi", "hello", "hey", "start", "menu", "help", "namaste", "hii", "hola",
}
ORDER_TRIGGERS = {
    "order", "food", "khana", "order food", "train food", "book food", "khana order",
    "khana mangao", "food order",
}


def _set_user(user_id: str) -> None:
    train_tools.set_current_user(user_id)
    train_tools.ensure_whatsapp_customer()


def _api_ok() -> bool:
    return api_client.api_enabled() and api_client.health_check()


def _format_delivery_window(dw: dict | None) -> str:
    if not dw:
        return ""
    start = dw.get("delivery_window_start", "")
    end = dw.get("delivery_window_end", "")
    if not start:
        return ""
    try:
        s = datetime.fromisoformat(str(start).replace("Z", "+00:00"))
        e = datetime.fromisoformat(str(end).replace("Z", "+00:00")) if end else None
        if e:
            return f"{s.strftime('%d %b, %I:%M %p')} – {e.strftime('%I:%M %p')}"
        return s.strftime("%d %b, %I:%M %p")
    except (ValueError, TypeError):
        return f"{start} – {end}" if end else str(start)


def _build_vendor_route_summary(stops: list[dict]) -> str:
    """Text block: each station + vendor names on route."""
    lines = ["*Stations on your route & kitchens:*\n"]
    for s in stops:
        code = s.get("station_code") or "?"
        name = s.get("station_name") or ""
        vendors = s.get("vendors") or []
        if not vendors:
            lines.append(f"• *{code}* {name} — ❌ koi vendor nahi")
            continue
        vnames = ", ".join(v.get("name", "") for v in vendors[:3])
        extra = f" +{len(vendors) - 3} more" if len(vendors) > 3 else ""
        lines.append(f"• *{code}* {name} — {vnames}{extra}")
    lines.append("\n_Neeche list se station choose karein jahan food chahiye._")
    return "\n".join(lines)


async def send_main_menu(wa: WhatsAppClient, to: str) -> None:
    if not api_client.api_enabled():
        await wa.send_text(
            to,
            "⚠️ Backend API is not configured. Set *API_BASE_URL* in agent/.env "
            "(e.g. http://localhost:8080) and start the Go server.",
        )
        return

    if not _api_ok():
        await wa.send_text(
            to,
            "⚠️ Cannot reach the backend API. Start the Go server on port 8080, then try again.",
        )
        return

    sess = get_session(to)
    sess.flow_step = FlowStep.IDLE
    sess.awaiting_pnr = False

    await wa.send_buttons(
        to,
        "Welcome to *Train Food* 🚂🍱\n\n"
        "Order meals at your train stop — step by step:\n"
        "1️⃣ PNR → 2️⃣ Station → 3️⃣ Vendor → 4️⃣ Seat → 5️⃣ Menu",
        [
            (MENU_TRAIN_ORDER, "Order food"),
            (MENU_TRAIN_CART, "My cart"),
            (MENU_ORDERS, "My orders"),
        ],
        header="RailFood",
        footer="Tap Order food to start",
    )


async def send_pnr_prompt(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)
    sess.reset_journey()
    sess.flow_step = FlowStep.AWAITING_PNR
    sess.awaiting_pnr = True

    await wa.send_text(
        to,
        "🎫 *Step 1 — PNR*\n\n"
        "Apna *10-digit PNR* bhejein (ticket / IRCTC app par milta hai).\n\n"
        "Example: `1234567890`",
    )


async def send_station_list(wa: WhatsAppClient, to: str) -> None:
    """Re-send station picker from session PNR data."""
    _set_user(to)
    sess = get_session(to)
    if not sess.pnr_lookup:
        await send_pnr_prompt(wa, to)
        return

    preview = train_tools.get_stops_with_vendors()
    stops = preview.get("stops") or []
    if not stops:
        await wa.send_text(to, "Is route par koi delivery stop nahi mila.")
        await send_main_menu(wa, to)
        return

    await wa.send_text(to, _build_vendor_route_summary(stops))

    rows = []
    for s in stops[:10]:
        vendors = s.get("vendors") or []
        vhint = ", ".join(v.get("name", "")[:12] for v in vendors[:2])
        if len(vendors) > 2:
            vhint += f" +{len(vendors) - 2}"
        desc = _clip(vhint or "No vendors", 72)
        rows.append(
            {
                "id": f"{STATION_PREFIX}{s.get('station_id')}",
                "title": _clip(f"{s.get('station_code', '')} {s.get('station_name', '')}", 24),
                "description": desc,
            }
        )

    await wa.send_list(
        to,
        "🚉 *Step 2 — Delivery station*\n\nKahan food deliver karna hai? List se station choose karein:",
        "Select station",
        [{"title": "Your route stops", "rows": rows}],
        header="Delivery stop",
        footer="Sirf apni journey ke stops dikhenge",
    )


async def handle_pnr_lookup(wa: WhatsAppClient, to: str, pnr: str) -> None:
    _set_user(to)
    result = train_tools.lookup_pnr(pnr)
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message', 'PNR lookup failed')}\n\nDobara 10-digit PNR bhejein.")
        await send_pnr_prompt(wa, to)
        return

    sess = get_session(to)
    train = sess.pnr_lookup.get("train", {}) if sess.pnr_lookup else {}

    await wa.send_text(
        to,
        f"✅ *PNR verified*\n\n"
        f"👤 {result.get('passenger_name')} · Coach {result.get('coach')}/{result.get('berth')}\n"
        f"🚂 {train.get('number')} {train.get('name')}\n"
        f"📍 {result.get('from_station')} → {result.get('to_station')}\n\n"
        "Ab dekho kis station par kaun-kaun se vendor hain 👇",
    )

    await send_station_list(wa, to)


async def send_vendors_list(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)

    result = train_tools.list_vendors_at_station()
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return

    vendors = result.get("vendors") or []
    if not vendors:
        await wa.send_text(
            to,
            f"*{sess.station_name}* par abhi koi vendor nahi hai.\n\nDusra station choose karein.",
        )
        await send_station_list(wa, to)
        return

    dw_text = _format_delivery_window(sess.delivery_window)
    intro = f"🏪 *Step 3 — Vendor*\n\n📍 *{sess.station_name}*"
    if dw_text:
        intro += f"\n🕐 Delivery window: {dw_text}"
    intro += f"\n\n*{len(vendors)} vendor(s)* yahan deliver karte hain:"

    vendor_lines = "\n".join(f"• {v.get('name')} ({v.get('code', '')})" for v in vendors[:10])
    await wa.send_text(to, f"{intro}\n\n{vendor_lines}\n\n_Neeche list se vendor choose karein._")

    rows = [
        {
            "id": f"{VENDOR_PREFIX}{v['id']}",
            "title": _clip(v.get("name", "Vendor"), 24),
            "description": _clip(v.get("code", "") or "Kitchen", 72),
        }
        for v in vendors[:10]
    ]
    await wa.send_list(
        to,
        "Kaunse kitchen se order karna hai?",
        "Select vendor",
        [{"title": "Vendors", "rows": rows}],
        header="Choose vendor",
    )


async def send_seat_prompt(wa: WhatsAppClient, to: str) -> None:
    """Ask coach/berth — confirm PNR seat or enter manually."""
    _set_user(to)
    sess = get_session(to)
    sess.flow_step = FlowStep.AWAITING_SEAT

    lookup = sess.pnr_lookup or {}
    coach = lookup.get("coach") or "?"
    berth = lookup.get("berth") or "?"

    await wa.send_buttons(
        to,
        f"🪑 *Seat / Coach*\n\n"
        f"Food kahan deliver karna hai?\n\n"
        f"PNR par likha hai: *Coach {coach} · Seat {berth}*\n\n"
        f"Kya yahi sahi hai?",
        [
            (SEAT_USE_PNR, f"Yes {coach}/{berth}"[:20]),
            (SEAT_MANUAL, "Dusri seat"),
        ],
        header="Delivery location",
        footer="Bogie + seat number",
    )


async def send_manual_seat_prompt(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)
    sess.flow_step = FlowStep.AWAITING_SEAT_TEXT
    await wa.send_text(
        to,
        "✏️ Apna *coach* aur *seat/berth* bhejein.\n\n"
        "Examples:\n"
        "• `A1 12`\n"
        "• `B2/45`\n"
        "• `S3 8`",
    )


def parse_seat(text: str) -> tuple[str, str] | None:
    m = SEAT_PATTERN.search(text.strip())
    if m:
        return m.group(1).upper(), m.group(2)
    parts = text.strip().split()
    if len(parts) >= 2 and parts[0] and parts[1].isdigit():
        return parts[0].upper(), parts[1]
    return None


async def confirm_seat_and_menu(wa: WhatsAppClient, to: str, coach: str, berth: str) -> None:
    result = train_tools.set_delivery_seat(coach, berth)
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return
    await wa.send_text(
        to,
        f"✅ Delivery *Coach {coach} · Seat {berth}* par confirm ho gaya.\n\nAb menu se items choose karein:",
    )
    await send_menu_list(wa, to)


async def send_menu_list(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    result = train_tools.browse_menu()
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return

    portions = result.get("portions") or []
    if not portions:
        await wa.send_text(to, "Menu khali hai.")
        return

    sess = get_session(to)
    rows = []
    for p in portions[:10]:
        rows.append(
            {
                "id": f"{PORTION_PREFIX}{p['menu_portion_id']}",
                "title": _clip(f"{p['item_name']}", 24),
                "description": _clip(
                    f"{p['portion_label']} · ₹{p['price_inr']:.0f} · stock {p['stock']}",
                    72,
                ),
            }
        )

    seat = f" · Coach {sess.coach}/{sess.berth}" if sess.coach and sess.berth else ""
    await wa.send_text(
        to,
        f"🍽️ *Step 5 — Menu*\n\n"
        f"*{sess.vendor_name}* · PNR {sess.pnr}\n"
        f"📍 {sess.station_name}{seat}\n\n"
        "Item choose karke cart mein add karein:",
    )
    await wa.send_list(
        to,
        "Tap an item to add:",
        "Select item",
        [{"title": "Menu", "rows": rows}],
        header="Food menu",
    )
    await wa.send_buttons(
        to,
        "Cart ready ho to:",
        [
            (MENU_TRAIN_CART, "View cart"),
            (MENU_TRAIN_CHECKOUT, "Place order"),
            (MENU_HOME, "Main menu"),
        ],
    )


async def send_portion_actions(wa: WhatsAppClient, to: str, portion_id: str) -> None:
    _set_user(to)
    result = train_tools.browse_menu()
    portion = next(
        (p for p in (result.get("portions") or []) if p["menu_portion_id"] == portion_id),
        None,
    )
    if not portion:
        await wa.send_text(to, "Item not found.")
        return

    await wa.send_buttons(
        to,
        f"*{portion['item_name']}* ({portion['portion_label']})\n"
        f"₹{portion['price_inr']:.2f} · stock {portion['stock']}\n\nKitne add karein?",
        [
            (f"{TRAIN_ADD1_PREFIX}{portion_id}", "Add 1"),
            (f"{TRAIN_ADD2_PREFIX}{portion_id}", "Add 2"),
            (MENU_TRAIN_CART, "View cart"),
        ],
        header="Add to cart",
    )


async def send_train_cart(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)

    if not sess.pnr:
        await send_pnr_prompt(wa, to)
        return

    cart = train_tools.view_train_cart()
    if cart.get("empty"):
        await wa.send_buttons(
            to,
            "Cart khali hai 🛒\n\nPehle menu se items add karein.",
            [(MENU_TRAIN_ORDER, "Order food"), (MENU_HOME, "Main menu")],
        )
        if sess.vendor_id:
            await send_menu_list(wa, to)
        elif sess.station_id:
            await send_vendors_list(wa, to)
        elif sess.pnr:
            await send_station_list(wa, to)
        return

    body = cart.get("cart_summary", "")
    header = f"PNR {sess.pnr} · {sess.station_name}"
    await wa.send_buttons(
        to,
        f"🛒 *Your cart*\n*{header}*\n\n{body}",
        [
            (MENU_TRAIN_CHECKOUT, "Place order"),
            (MENU_TRAIN_ORDER, "Add more"),
            (MENU_HOME, "Main menu"),
        ],
        header="Cart",
    )


async def send_orders_summary(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    data = train_tools.get_recent_orders()
    if not data.get("orders"):
        await wa.send_buttons(
            to,
            "Abhi koi order nahi.",
            [(MENU_TRAIN_ORDER, "Order food"), (MENU_HOME, "Main menu")],
        )
        return

    lines = []
    for o in data["orders"]:
        oid = str(o.get("order_id", ""))[:8]
        lines.append(
            f"#{oid}… — ₹{o['total_inr']:.2f} ({o['status']})"
            + (f" · PNR {o['pnr']}" if o.get("pnr") else "")
            + (f" · {o['station']}" if o.get("station") else "")
        )
    await wa.send_text(to, "*Recent orders*\n\n" + "\n".join(lines))
    await wa.send_buttons(to, "Options:", [(MENU_HOME, "Main menu")])


async def handle_checkout(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)

    if not sess.pnr or not sess.station_id or not sess.vendor_id:
        await wa.send_text(to, "❌ Pehle PNR → station → vendor → cart complete karein.")
        if not sess.pnr:
            await send_pnr_prompt(wa, to)
        elif not sess.station_id:
            await send_station_list(wa, to)
        elif not sess.vendor_id:
            await send_vendors_list(wa, to)
        return

    if not api_client.api_enabled():
        await wa.send_text(to, "⚠️ Backend API not configured. Set API_BASE_URL in agent/.env")
        return

    result = train_tools.place_train_order()
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message', 'Checkout failed')}")
        await send_train_cart(wa, to)
        return

    oid = str(result.get("order_id", ""))[:8]
    msg = (
        f"✅ *Order placed!*\n"
        f"ID: `{result.get('order_id')}`\n"
        f"Total: ₹{result.get('total_inr', 0):.2f}\n"
        f"Station: {result.get('station')}\n"
        f"Vendor: {result.get('vendor')}\n"
        f"Status: {result.get('order_status')}"
    )
    if result.get("delivery_window"):
        msg += f"\nDelivery: {result['delivery_window']}"
    if result.get("items_summary"):
        msg += f"\n\n*Items:*\n{result['items_summary']}"

    await wa.send_text(to, msg)
    await wa.send_buttons(
        to,
        "Aur kuch?",
        [(MENU_ORDERS, "My orders"), (MENU_TRAIN_ORDER, "New order"), (MENU_HOME, "Menu")],
    )


async def handle_interactive(wa: WhatsAppClient, to: str, reply_id: str) -> None:
    _set_user(to)
    logger.info("Train interactive from %s: %s", to, reply_id)

    if reply_id == MENU_HOME:
        await send_main_menu(wa, to)
        return
    if reply_id in {MENU_TRAIN_ORDER, MENU_PNR_PROMPT}:
        await send_pnr_prompt(wa, to)
        return
    if reply_id == MENU_CHANGE_PNR:
        await send_pnr_prompt(wa, to)
        return
    if reply_id == MENU_TRAIN_CART:
        await send_train_cart(wa, to)
        return
    if reply_id == MENU_ORDERS:
        await send_orders_summary(wa, to)
        return
    if reply_id == MENU_TRAIN_CHECKOUT:
        await handle_checkout(wa, to)
        return

    if reply_id.startswith(STATION_PREFIX):
        station_id = reply_id[len(STATION_PREFIX) :]
        result = train_tools.select_delivery_station(station_id)
        if result.get("status") == "error":
            await wa.send_text(
                to,
                f"❌ {result.get('message')}\n\n"
                "Is station par delivery possible nahi — koi aur station try karein.",
            )
            await send_station_list(wa, to)
            return

        sess = get_session(to)
        dw_text = _format_delivery_window(sess.delivery_window)
        confirm = f"✅ *{result.get('station_name')}* par delivery confirm!"
        if dw_text:
            confirm += f"\n🕐 Window: {dw_text}"
        await wa.send_text(to, confirm)
        await send_vendors_list(wa, to)
        return

    if reply_id.startswith(VENDOR_PREFIX):
        vendor_id = reply_id[len(VENDOR_PREFIX) :]
        result = train_tools.select_vendor(vendor_id)
        if result.get("status") == "error":
            await wa.send_text(to, f"❌ {result.get('message')}")
            return
        await wa.send_text(to, f"✅ *{result.get('vendor_name')}* selected")
        await send_seat_prompt(wa, to)
        return

    if reply_id == SEAT_USE_PNR:
        sess = get_session(to)
        lookup = sess.pnr_lookup or {}
        coach = str(lookup.get("coach") or "").strip()
        berth = str(lookup.get("berth") or "").strip()
        if not coach or not berth:
            await send_manual_seat_prompt(wa, to)
            return
        await confirm_seat_and_menu(wa, to, coach, berth)
        return

    if reply_id == SEAT_MANUAL:
        await send_manual_seat_prompt(wa, to)
        return

    if reply_id.startswith(PORTION_PREFIX):
        await send_portion_actions(wa, to, reply_id[len(PORTION_PREFIX) :])
        return

    if reply_id.startswith(TRAIN_ADD1_PREFIX):
        pid = reply_id[len(TRAIN_ADD1_PREFIX) :]
        r = train_tools.add_meal_to_cart(pid, 1)
        await wa.send_text(to, r.get("message", "Added ✅"))
        await send_train_cart(wa, to)
        return

    if reply_id.startswith(TRAIN_ADD2_PREFIX):
        pid = reply_id[len(TRAIN_ADD2_PREFIX) :]
        r = train_tools.add_meal_to_cart(pid, 2)
        await wa.send_text(to, r.get("message", "Added ✅"))
        await send_train_cart(wa, to)
        return

    await wa.send_text(to, "Option samajh nahi aaya. Menu khol rahe hain…")
    await send_main_menu(wa, to)


def _normalize(text: str) -> str:
    return re.sub(r"\s+", " ", text.strip().lower())


def wants_menu(text: str) -> bool:
    t = _normalize(text)
    return t in MENU_TRIGGERS or t.startswith(("menu", "help"))


def wants_order(text: str) -> bool:
    t = _normalize(text)
    return t in ORDER_TRIGGERS or "order food" in t or "khana" in t


def extract_pnr(text: str) -> str | None:
    m = PNR_PATTERN.search(text.replace(" ", ""))
    return m.group(1) if m else None


def _wants_ai_help(text: str) -> bool:
    t = _normalize(text)
    return "?" in text or any(
        t.startswith(p)
        for p in ("what", "how", "why", "kya", "kaise", "kab", "explain")
    )


async def _prompt_for_current_step(wa: WhatsAppClient, to: str) -> None:
    sess = get_session(to)
    if sess.flow_step == FlowStep.AWAITING_PNR or sess.awaiting_pnr:
        await wa.send_text(to, "🎫 Pehle apna *10-digit PNR* bhejein.")
        return
    if sess.flow_step == FlowStep.AWAITING_STATION:
        await wa.send_text(to, "🚉 Neeche wali *station list* se delivery stop choose karein.")
        await send_station_list(wa, to)
        return
    if sess.flow_step == FlowStep.AWAITING_VENDOR:
        await wa.send_text(to, "🏪 Ab *vendor list* se kitchen choose karein.")
        await send_vendors_list(wa, to)
        return
    if sess.flow_step in {FlowStep.AWAITING_SEAT, FlowStep.AWAITING_SEAT_TEXT}:
        await wa.send_text(to, "🪑 Pehle *coach aur seat* confirm karein.")
        if sess.flow_step == FlowStep.AWAITING_SEAT_TEXT:
            await send_manual_seat_prompt(wa, to)
        else:
            await send_seat_prompt(wa, to)
        return
    if sess.flow_step == FlowStep.ORDERING:
        await wa.send_text(to, "🍽️ Menu se item choose karein ya cart dekhein.")
        await send_menu_list(wa, to)
        return
    await send_pnr_prompt(wa, to)


async def handle_text(wa: WhatsAppClient, to: str, text: str, runner) -> None:
    """Guided ordering flow; Gemini only for explicit help questions when idle."""
    _set_user(to)
    sess = get_session(to)

    if wants_menu(text):
        await send_main_menu(wa, to)
        return

    if wants_order(text):
        await send_pnr_prompt(wa, to)
        return

    if _normalize(text) in {"cart", "my cart", "mera cart"}:
        await send_train_cart(wa, to)
        return

    if _normalize(text) in {"orders", "my orders", "mera order"}:
        await send_orders_summary(wa, to)
        return

    pnr = extract_pnr(text)
    if pnr:
        await handle_pnr_lookup(wa, to, pnr)
        return

    if sess.awaiting_pnr or sess.flow_step == FlowStep.AWAITING_PNR:
        await wa.send_text(
            to,
            "❌ Valid 10-digit PNR nahi mila.\n\nExample: `1234567890`",
        )
        return

    if sess.flow_step == FlowStep.AWAITING_SEAT_TEXT:
        seat = parse_seat(text)
        if not seat:
            await wa.send_text(
                to,
                "❌ Format samajh nahi aaya.\n\nBhejein jaise: `A1 12` ya `B2/45`",
            )
            return
        await confirm_seat_and_menu(wa, to, seat[0], seat[1])
        return

    # Mid-flow: keep user on the guided path (no Gemini)
    if sess.flow_step != FlowStep.IDLE:
        await _prompt_for_current_step(wa, to)
        return

    # Idle: optional AI for questions only; otherwise start order with PNR
    if runner and _wants_ai_help(text):
        try:
            reply = await runner.chat(to, text)
            await wa.send_text(to, reply[:4000])
        except Exception:
            logger.exception("Agent failed for %s", to)
            await wa.send_text(to, "Sorry, samajh nahi aaya. Order ke liye PNR bhejein.")
        await wa.send_buttons(
            to,
            "Order shuru karein:",
            [(MENU_TRAIN_ORDER, "Order food"), (MENU_HOME, "Menu")],
        )
        return

    # Default for new / casual messages: welcome + ask PNR first
    await wa.send_text(
        to,
        "Namaste! 🚂 *Train Food* mein aapka swagat hai.\n\n"
        "Order ke liye sabse pehle apna *10-digit PNR* bhejein.",
    )
    await send_pnr_prompt(wa, to)


def parse_interactive_reply(message: dict) -> str | None:
    interactive = message.get("interactive") or {}
    if interactive.get("type") == "list_reply":
        return interactive.get("list_reply", {}).get("id")
    if interactive.get("type") == "button_reply":
        return interactive.get("button_reply", {}).get("id")
    return None
