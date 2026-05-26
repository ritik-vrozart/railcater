"""
WhatsApp guided flow for train food ordering.

Step 1: Train number → pantry auto-selected
Step 2: Passenger name
Step 3: Coach / seat
Step 4: Menu category
Step 5: Items → cart → checkout
"""

from __future__ import annotations

import logging
import re

from config import settings
from services import api_client
from services.food_handler import (
    handle_food_message,
    send_smart_order_prompt,
)
from services.food_intent import looks_like_food_order
from services.menu_images import resolve_menu_image_url
from services.whatsapp import WhatsAppClient, _clip
from store.session import FlowStep, get_session
from tools import train_tools

logger = logging.getLogger(__name__)

MENU_HOME = "menu_home"
MENU_TRAIN_ORDER = "menu_train"
MENU_ADD_MORE = "menu_add_more"
MENU_TRAIN_CART = "menu_train_cart"
MENU_TRAIN_CHECKOUT = "train_checkout"
MENU_ORDERS = "menu_orders"

CAT_PREFIX = "cat_"
PORTION_PREFIX = "mp_"
MENU_LIST_PAGE_PREFIX = "mlp_"
MENU_IMAGE_PAGE_PREFIX = "mimg_"
MENU_SHOW_IMAGES = "menu_show_img"
ITEM_PHOTO_PREFIX = "mphoto_"
TRAIN_ADD1_PREFIX = "tadd1_"
TRAIN_ADD2_PREFIX = "tadd2_"

MENU_LIST_ITEMS_PER_PAGE = 10  # WhatsApp list max rows
MENU_IMAGE_ITEMS_PER_PAGE = 5  # only when user asks for photos

SHOW_IMAGE_TRIGGERS = {
    "show image", "show images", "image dikhao", "photo dikhao", "photos dikhao",
    "image dikha", "photo dikha", "pics dikhao", "show photo", "show photos",
    "image show", "photo show",
}

TRAIN_NUMBER_PATTERN = re.compile(r"\b(\d{4,6})\b")
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
        from datetime import datetime

        s = datetime.fromisoformat(str(start).replace("Z", "+00:00"))
        e = datetime.fromisoformat(str(end).replace("Z", "+00:00")) if end else None
        if e:
            return f"{s.strftime('%d %b, %I:%M %p')} – {e.strftime('%I:%M %p')}"
        return s.strftime("%d %b, %I:%M %p")
    except (ValueError, TypeError):
        return f"{start} – {end}" if end else str(start)


async def send_main_menu(wa: WhatsAppClient, to: str) -> None:
    if not api_client.api_enabled():
        await wa.send_text(
            to,
            "⚠️ Backend API is not configured. Set *API_BASE_URL* in agent/.env and start the Go API.",
        )
        return

    if not _api_ok():
        await wa.send_text(
            to,
            f"⚠️ Cannot reach the backend API at {settings.api_base_url or '(API_BASE_URL not set)'}.",
        )
        return

    sess = get_session(to)
    sess.flow_step = FlowStep.IDLE
    sess.awaiting_pnr = False

    await wa.send_buttons(
        to,
        "Welcome to *Train Food* 🚂🍱\n\n"
        "Order meals on your train — step by step:\n"
        "1️⃣ Train no → 2️⃣ Naam → 3️⃣ Seat → 4️⃣ Category → 5️⃣ Menu",
        [
            (MENU_TRAIN_ORDER, "Order food"),
            (MENU_TRAIN_CART, "My cart"),
            (MENU_ORDERS, "My orders"),
        ],
        header="RailFood",
        footer="Tap Order food to start",
    )


async def send_train_prompt(wa: WhatsAppClient, to: str) -> None:
    """Start a fresh order (clears previous train context)."""
    _set_user(to)
    sess = get_session(to)
    sess.reset_journey()
    sess.flow_step = FlowStep.AWAITING_TRAIN_NUMBER

    await wa.send_text(
        to,
        "🚂 *Step 1 — Train number*\n\n"
        "Apni train ka *number* bhejein (ticket par likha hota hai).\n\n"
        "Example: `12951`",
    )


async def resume_ordering(wa: WhatsAppClient, to: str) -> None:
    """Continue on same train — no train number / naam / seat again (15 min window)."""
    _set_user(to)
    sess = get_session(to)

    if not sess.has_active_journey():
        await wa.send_text(
            to,
            "⏱️ Session expire ho gaya (15 min). Naya order ke liye train number bhejein.",
        )
        await send_train_prompt(wa, to)
        return

    sess.touch_journey()
    sess.flow_step = FlowStep.ORDERING
    # Fresh browse — don't reopen last category (e.g. water after user wants thali)
    sess.category_id = None
    sess.category_name = None

    seat = f" · Coach {sess.coach}/{sess.berth}" if sess.coach and sess.berth else ""
    await wa.send_text(
        to,
        f"🚂 Train *{sess.train_number}* · {sess.vendor_name}\n"
        f"👤 {sess.passenger_name or 'Guest'}{seat}\n",
    )
    await send_smart_order_prompt(wa, to)
    await send_category_list(wa, to)


async def handle_train_lookup(wa: WhatsAppClient, to: str, train_number: str) -> None:
    _set_user(to)
    result = train_tools.lookup_train(train_number)
    if result.get("status") == "error":
        await wa.send_text(
            to,
            f"❌ {result.get('message', 'Train not found')}\n\nDobara train number bhejein.",
        )
        await send_train_prompt(wa, to)
        return

    await wa.send_text(
        to,
        f"✅ *Train {result.get('train_number')}* — {result.get('train_name')}\n"
        f"🏪 Pantry: *{result.get('pantry_name')}*\n\n"
        "📝 *Step 2 — Apna naam* bhejein:",
    )


async def send_name_prompt(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)
    sess.flow_step = FlowStep.AWAITING_NAME
    await wa.send_text(to, "📝 Apna *poora naam* bhejein (jaise ticket par hai):")


async def handle_name(wa: WhatsAppClient, to: str, name: str) -> None:
    _set_user(to)
    result = train_tools.set_passenger_name(name)
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return

    await wa.send_text(
        to,
        f"✅ Namaste *{result.get('passenger_name')}*!\n\n"
        "🪑 *Step 3 — Coach aur seat*\n\n"
        "Apna coach + seat number bhejein.\n\n"
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


async def send_category_list(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    result = train_tools.list_menu_categories()
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return

    categories = result.get("categories") or []
    sess = get_session(to)
    menu = train_tools.browse_menu()
    item_counts: dict[str, int] = {}
    for p in (menu.get("portions") or []):
        cid = p.get("category_id") or ""
        item_counts[cid] = item_counts.get(cid, 0) + 1

    rows = []
    for c in categories[:10]:
        cid = str(c.get("id", ""))
        n = item_counts.get(cid, 0)
        desc = (c.get("description") or "").strip()
        if not desc and n > 0:
            desc = f"{n} items"
        elif not desc:
            desc = "Menu"
        rows.append(
            {
                "id": f"{CAT_PREFIX}{cid}",
                "title": _clip(c.get("name", "Category"), 24),
                "description": _clip(desc, 72),
            }
        )

    seat = f" · Coach {sess.coach}/{sess.berth}" if sess.coach and sess.berth else ""
    await wa.send_text(
        to,
        f"🍽️ *Step 4 — Category*\n\n"
        f"Train *{sess.train_number}* · {sess.vendor_name}{seat}\n\n"
        "Kya order karna hai? Category choose karein:",
    )
    await wa.send_list(
        to,
        "Neeche list se category tap karein:",
        "Select category",
        [{"title": "Categories", "rows": rows}],
        header="Food category",
    )


async def confirm_seat_and_categories(wa: WhatsAppClient, to: str, coach: str, berth: str) -> None:
    result = train_tools.set_delivery_seat(coach, berth)
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return
    await wa.send_text(
        to,
        f"✅ Seat *Coach {coach} · {berth}* confirm ho gaya.\n",
    )
    await send_smart_order_prompt(wa, to)
    await send_category_list(wa, to)


def _veg_badge(is_veg: bool) -> str:
    return "🟢 Veg" if is_veg else "🔴 Non-veg"


def _portion_body(p: dict) -> str:
    lines = [
        f"*{_veg_badge(p.get('is_veg', True))} {p['item_name']}*",
        f"_{p.get('portion_label', '')}_ · ₹{p['price_inr']:.0f}",
        f"Stock: {p.get('stock', 0)}",
    ]
    desc = (p.get("item_description") or "").strip()
    if desc:
        lines.append(f"\n{desc[:200]}")
    lines.append("\nKitne add karein?")
    return "\n".join(lines)


async def _send_portion_card(wa: WhatsAppClient, to: str, p: dict) -> None:
    """One menu item card: optional image + Add buttons."""
    pid = p["menu_portion_id"]
    image_url = resolve_menu_image_url(p.get("image_url"))
    body = _portion_body(p)
    buttons = [
        (f"{TRAIN_ADD1_PREFIX}{pid}", "Add 1"),
        (f"{TRAIN_ADD2_PREFIX}{pid}", "Add 2"),
        (MENU_TRAIN_CART, "Cart"),
    ]
    footer = p.get("portion_label", "")[:60]
    try:
        if image_url:
            await wa.send_buttons(
                to, body, buttons, header_image_url=image_url, footer=footer
            )
        else:
            await wa.send_buttons(
                to, body, buttons, header=_clip(p["item_name"], 60), footer=footer
            )
    except Exception:
        logger.exception("Image card failed for %s", pid)
        if image_url:
            await wa.send_image(to, image_url, caption=_portion_body(p)[:1024])
        await wa.send_buttons(to, "Add to cart:", buttons)


async def send_menu_list(
    wa: WhatsAppClient,
    to: str,
    *,
    page: int = 0,
    portions: list[dict] | None = None,
    list_intro: str | None = None,
) -> None:
    """Compact list of all items (default) — scales to 50–100+ with pages."""
    _set_user(to)

    if portions is None:
        result = train_tools.browse_menu()
        if result.get("status") == "error":
            await wa.send_text(to, f"❌ {result.get('message')}")
            return
        portions = result.get("portions") or []
    if not portions:
        await wa.send_text(to, "Is category mein abhi koi item nahi hai.")
        await send_category_list(wa, to)
        return

    total = len(portions)
    per = MENU_LIST_ITEMS_PER_PAGE
    start = page * per
    end = min(start + per, total)
    if start >= total:
        page = 0
        start = 0
        end = min(per, total)

    chunk = portions[start:end]
    sess = get_session(to)
    sess.menu_list_page = page
    seat = f" · Coach {sess.coach}/{sess.berth}" if sess.coach and sess.berth else ""

    rows = [
        {
            "id": f"{PORTION_PREFIX}{p['menu_portion_id']}",
            "title": _clip(f"{p['item_name']}", 24),
            "description": _clip(
                f"{_veg_badge(p.get('is_veg', True))} {p['portion_label']} · ₹{p['price_inr']:.0f}",
                72,
            ),
        }
        for p in chunk
    ]

    if page == 0:
        intro = list_intro or (
            f"🍽️ *Step 5 — Menu*\n\n"
            f"*{sess.category_name or 'Menu'}* · Train {sess.train_number}\n"
            f"🏪 {sess.vendor_name}{seat}\n\n"
            f"*{total}* items — neeche list se choose karein.\n"
            f"_Photos: *show image*_ · _Search: type item name_"
        )
        await wa.send_text(to, intro)

    await wa.send_list(
        to,
        f"Items {start + 1}–{end} of {total} — tap to add:",
        "Select item",
        [{"title": _clip(sess.category_name or "Menu", 24), "rows": rows}],
        header="Menu",
        footer=f"Page {page + 1}",
    )

    nav: list[tuple[str, str]] = []
    if end < total:
        nav.append((f"{MENU_LIST_PAGE_PREFIX}{page + 1}", "More items"))
    if page > 0:
        nav.append((f"{MENU_LIST_PAGE_PREFIX}{page - 1}", "Previous"))
    nav.append((MENU_SHOW_IMAGES, "Show images"))

    await wa.send_buttons(
        to,
        f"List: {start + 1}–{end} / {total}",
        nav[:3],
        footer="Photos optional",
    )
    await wa.send_buttons(
        to,
        "Cart / order:",
        [(MENU_TRAIN_CART, "View cart"), (MENU_TRAIN_CHECKOUT, "Place order")],
    )


async def send_menu_images(wa: WhatsAppClient, to: str, *, page: int | None = None) -> None:
    """Photo cards — only when user taps Show images or types show image."""
    _set_user(to)
    result = train_tools.browse_menu()
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message')}")
        return

    portions = result.get("portions") or []
    if not portions:
        await wa.send_text(to, "Is category mein koi item nahi.")
        return

    if page is None:
        page = get_session(to).menu_list_page

    total = len(portions)
    per = MENU_IMAGE_ITEMS_PER_PAGE
    start = page * per
    end = min(start + per, total)
    if start >= total:
        page = 0
        start = 0
        end = min(per, total)

    chunk = portions[start:end]
    sess = get_session(to)

    await wa.send_text(
        to,
        f"📸 *Photos* ({start + 1}–{end} of {total})\n"
        f"_Wapas list ke liye koi bhi item list se choose karein._",
    )

    for p in chunk:
        await _send_portion_card(wa, to, p)

    nav: list[tuple[str, str]] = []
    if end < total:
        nav.append((f"{MENU_IMAGE_PAGE_PREFIX}{page + 1}", "More photos"))
    if page > 0:
        nav.append((f"{MENU_IMAGE_PAGE_PREFIX}{page - 1}", "Back"))
    nav.append((f"{MENU_LIST_PAGE_PREFIX}{page}", "Back to list"))

    await wa.send_buttons(to, "Photos:", nav[:3])


async def send_portion_actions(
    wa: WhatsAppClient, to: str, portion_id: str, *, with_image: bool = False
) -> None:
    _set_user(to)
    result = train_tools.browse_menu()
    portion = next(
        (p for p in (result.get("portions") or []) if p["menu_portion_id"] == portion_id),
        None,
    )
    if not portion:
        await wa.send_text(to, "Item not found.")
        return

    if with_image:
        await _send_portion_card(wa, to, portion)
        return

    pid = portion["menu_portion_id"]
    await wa.send_buttons(
        to,
        _portion_body(portion),
        [
            (f"{TRAIN_ADD1_PREFIX}{pid}", "Add 1"),
            (f"{TRAIN_ADD2_PREFIX}{pid}", "Add 2"),
            (f"{ITEM_PHOTO_PREFIX}{pid}", "Show photo"),
        ],
        header=_clip(portion["item_name"], 60),
        footer=f"₹{portion['price_inr']:.0f}",
    )


async def send_train_cart(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)

    if not sess.train_number:
        await send_train_prompt(wa, to)
        return

    cart = train_tools.view_train_cart()
    if cart.get("empty"):
        more_btn = (
            (MENU_ADD_MORE, "Browse menu")
            if sess.has_active_journey()
            else (MENU_TRAIN_ORDER, "Order food")
        )
        await wa.send_buttons(
            to,
            "Cart khali hai 🛒\n\nPehle menu se items add karein.",
            [more_btn, (MENU_HOME, "Main menu")],
        )
        if sess.flow_step == FlowStep.ORDERING:
            await send_menu_list(wa, to)
        elif sess.flow_step == FlowStep.AWAITING_CATEGORY:
            await send_category_list(wa, to)
        return

    header = f"Train {sess.train_number}"
    if sess.passenger_name:
        header += f" · {sess.passenger_name}"
    body = cart.get("cart_summary", "")
    await wa.send_buttons(
        to,
        f"🛒 *Your cart*\n*{header}*\n\n{body}",
        [
            (MENU_TRAIN_CHECKOUT, "Place order"),
            (MENU_ADD_MORE, "Add more"),
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
            + (f" · Train {o.get('pnr')}" if o.get("pnr") else "")
        )
    await wa.send_text(to, "*Recent orders*\n\n" + "\n".join(lines))
    await wa.send_buttons(to, "Options:", [(MENU_HOME, "Main menu")])


async def handle_checkout(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    sess = get_session(to)

    if not sess.train_number or not sess.vendor_id:
        await wa.send_text(to, "❌ Pehle train → naam → seat → category → cart complete karein.")
        await send_train_prompt(wa, to)
        return
    if not sess.passenger_name:
        await send_name_prompt(wa, to)
        return
    if not sess.coach or not sess.berth:
        await wa.send_text(to, "🪑 Pehle coach aur seat bhejein.")
        return
    if not sess.category_id:
        await send_category_list(wa, to)
        return

    if not api_client.api_enabled():
        await wa.send_text(to, "⚠️ Backend API not configured. Set API_BASE_URL in agent/.env")
        return

    result = train_tools.place_train_order()
    if result.get("status") == "error":
        await wa.send_text(to, f"❌ {result.get('message', 'Checkout failed')}")
        await send_train_cart(wa, to)
        return

    msg = (
        f"✅ *Order placed!*\n"
        f"ID: `{result.get('order_id')}`\n"
        f"Total: ₹{result.get('total_inr', 0):.2f}\n"
        f"Train: {sess.train_number}\n"
        f"Pantry: {result.get('vendor')}\n"
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
    get_session(to).touch_journey()
    logger.info("Train interactive from %s: %s", to, reply_id)

    if reply_id == MENU_HOME:
        await send_main_menu(wa, to)
        return
    if reply_id == MENU_TRAIN_ORDER:
        await send_train_prompt(wa, to)
        return
    if reply_id == MENU_ADD_MORE:
        await resume_ordering(wa, to)
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

    if reply_id == MENU_SHOW_IMAGES:
        await send_menu_images(wa, to)
        return

    if reply_id.startswith(MENU_LIST_PAGE_PREFIX):
        try:
            page = int(reply_id[len(MENU_LIST_PAGE_PREFIX) :])
        except ValueError:
            page = 0
        await send_menu_list(wa, to, page=page)
        return

    if reply_id.startswith(MENU_IMAGE_PAGE_PREFIX):
        try:
            page = int(reply_id[len(MENU_IMAGE_PAGE_PREFIX) :])
        except ValueError:
            page = 0
        await send_menu_images(wa, to, page=page)
        return

    if reply_id.startswith(ITEM_PHOTO_PREFIX):
        await send_portion_actions(wa, to, reply_id[len(ITEM_PHOTO_PREFIX) :], with_image=True)
        return

    if reply_id.startswith(CAT_PREFIX):
        category_id = reply_id[len(CAT_PREFIX) :]
        result = train_tools.select_category(category_id)
        if result.get("status") == "error":
            await wa.send_text(to, f"❌ {result.get('message')}")
            return
        await wa.send_text(to, f"✅ *{result.get('category_name')}* selected")
        await send_menu_list(wa, to)
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


def wants_show_images(text: str) -> bool:
    t = _normalize(text)
    return t in SHOW_IMAGE_TRIGGERS or (
        ("image" in t or "photo" in t or "pic" in t) and ("dikha" in t or "show" in t)
    )


def extract_train_number(text: str) -> str | None:
    m = TRAIN_NUMBER_PATTERN.search(text.replace(" ", ""))
    return m.group(1) if m else None


def _wants_ai_help(text: str) -> bool:
    t = _normalize(text)
    return "?" in text or any(
        t.startswith(p)
        for p in ("what", "how", "why", "kya", "kaise", "kab", "explain")
    )


async def _prompt_for_current_step(wa: WhatsAppClient, to: str) -> None:
    sess = get_session(to)
    if sess.flow_step == FlowStep.AWAITING_TRAIN_NUMBER:
        await wa.send_text(to, "🚂 Pehle apna *train number* bhejein.")
        return
    if sess.flow_step == FlowStep.AWAITING_NAME:
        await send_name_prompt(wa, to)
        return
    if sess.flow_step == FlowStep.AWAITING_SEAT:
        await wa.send_text(
            to,
            "🪑 Apna *coach aur seat* bhejein.\n\nExample: `A1 12` ya `B2/45`",
        )
        return
    if sess.flow_step == FlowStep.AWAITING_CATEGORY:
        await send_smart_order_prompt(wa, to)
        await send_category_list(wa, to)
        return
    if sess.flow_step == FlowStep.ORDERING:
        await send_smart_order_prompt(wa, to)
        await send_menu_list(wa, to)
        return
    await send_train_prompt(wa, to)


async def handle_text(wa: WhatsAppClient, to: str, text: str, runner) -> None:
    """Guided ordering: train → name → seat → category → menu."""
    _set_user(to)
    sess = get_session(to)
    if sess.has_active_journey():
        sess.touch_journey()

    if wants_menu(text):
        await send_main_menu(wa, to)
        return

    if wants_order(text):
        if sess.has_active_journey():
            await resume_ordering(wa, to)
        else:
            await send_train_prompt(wa, to)
        return

    if _normalize(text) in {"cart", "my cart", "mera cart"}:
        await send_train_cart(wa, to)
        return

    if _normalize(text) in {"orders", "my orders", "mera order"}:
        await send_orders_summary(wa, to)
        return

    if sess.flow_step == FlowStep.AWAITING_TRAIN_NUMBER:
        num = extract_train_number(text)
        if num:
            await handle_train_lookup(wa, to, num)
        else:
            await wa.send_text(
                to,
                "❌ Valid train number nahi mila.\n\nExample: `12951`",
            )
        return

    if sess.flow_step == FlowStep.AWAITING_NAME:
        if len(text.strip()) >= 2:
            await handle_name(wa, to, text.strip())
        else:
            await wa.send_text(to, "❌ Apna naam bhejein (kam se kam 2 letters).")
        return

    if sess.flow_step == FlowStep.AWAITING_SEAT:
        seat = parse_seat(text)
        if not seat:
            await wa.send_text(
                to,
                "❌ Format samajh nahi aaya.\n\nBhejein jaise: `A1 12` ya `B2/45`",
            )
            return
        await confirm_seat_and_categories(wa, to, seat[0], seat[1])
        return

    if sess.flow_step == FlowStep.ORDERING and wants_show_images(text):
        await send_menu_images(wa, to)
        return

    # Natural language: "paneer thali chahiye", "2 chai", etc.
    if sess.has_active_journey() and looks_like_food_order(text):
        handled = await handle_food_message(
            wa,
            to,
            text,
            send_menu_list_fn=send_menu_list,
            send_portion_actions_fn=send_portion_actions,
            send_category_list_fn=send_category_list,
        )
        if handled:
            return

    # Mid-flow: keep user on guided path
    if sess.flow_step != FlowStep.IDLE:
        await _prompt_for_current_step(wa, to)
        return

    # Idle but active journey — ask what they want, don't replay old category list only
    if sess.has_active_journey():
        if looks_like_food_order(text):
            await handle_food_message(
                wa,
                to,
                text,
                send_menu_list_fn=send_menu_list,
                send_portion_actions_fn=send_portion_actions,
                send_category_list_fn=send_category_list,
            )
        else:
            await resume_ordering(wa, to)
        return

    train_num = extract_train_number(text)
    if train_num:
        await handle_train_lookup(wa, to, train_num)
        return

    if runner and _wants_ai_help(text):
        try:
            reply = await runner.chat(to, text)
            await wa.send_text(to, reply[:4000])
        except Exception:
            logger.exception("Agent failed for %s", to)
            await wa.send_text(to, "Sorry, samajh nahi aaya. Order ke liye train number bhejein.")
        await wa.send_buttons(
            to,
            "Order shuru karein:",
            [(MENU_TRAIN_ORDER, "Order food"), (MENU_HOME, "Menu")],
        )
        return

    await wa.send_text(
        to,
        "Namaste! 🚂 *Train Food* mein aapka swagat hai.\n\n"
        "Order ke liye sabse pehle apna *train number* bhejein.\n\n"
        "Example: `12951`",
    )
    await send_train_prompt(wa, to)


def parse_interactive_reply(message: dict) -> str | None:
    interactive = message.get("interactive") or {}
    if interactive.get("type") == "list_reply":
        return interactive.get("list_reply", {}).get("id")
    if interactive.get("type") == "button_reply":
        return interactive.get("button_reply", {}).get("id")
    return None
