"""
WhatsApp interactive UI: reply buttons + list (dropdown).
Handles button/list taps; free-text still uses Gemini agent when needed.
"""

from __future__ import annotations

import logging
import re

from config import settings
from data.products import CATALOG, get_product
from services.whatsapp import WhatsAppClient, _clip
from tools import shop_tools
from tools.shop_tools import add_to_cart, confirm_payment, get_my_orders, place_order, view_cart

logger = logging.getLogger(__name__)

# --- Interactive IDs ---
MENU_BROWSE = "menu_browse"
MENU_CART = "menu_cart"
MENU_ORDERS = "menu_orders"
MENU_CHECKOUT = "menu_checkout"
MENU_HOME = "menu_home"
MENU_SEARCH = "menu_search"

CAT_PREFIX = "cat_"
PROD_PREFIX = "prod_"
PAGE_PREFIX = "page_"
ADD1_PREFIX = "add1_"
ADD2_PREFIX = "add2_"
ADD3_PREFIX = "add3_"
PAY_PREFIX = "pay_"
MENU_WEBSHOP = "menu_webshop"

PRODUCTS_PER_PAGE = 10

MENU_TRIGGERS = {
    "hi", "hello", "hey", "start", "menu", "help", "namaste", "hii", "hola",
}
BROWSE_TRIGGERS = {
    "catalog", "products", "browse", "shop", "list", "sab dikhao", "sab dikha do",
    "show products", "all products", "available", "kya kya milta hai",
}


def _set_user(user_id: str) -> None:
    shop_tools.set_current_user(user_id)


def _normalize(text: str) -> str:
    return re.sub(r"\s+", " ", text.strip().lower())


def wants_menu(text: str) -> bool:
    t = _normalize(text)
    return t in MENU_TRIGGERS or t.startswith(("menu", "help"))


def wants_browse(text: str) -> bool:
    t = _normalize(text)
    return any(p in t for p in BROWSE_TRIGGERS) or "show me all" in t or "things that are available" in t


def shop_url(user_id: str) -> str:
    base = settings.public_base_url.rstrip("/")
    return f"{base}/shop?u={user_id}"


async def send_shop_website(wa: WhatsAppClient, to: str) -> None:
    """CTA URL opens in WhatsApp in-app browser (do not paste link in Chrome)."""
    url = shop_url(to)
    await wa.send_cta_url(
        to,
        "Tap the button below to open our shop *inside WhatsApp*.\n\n"
        "⚠️ Do not copy the link to Chrome — use only this button.\n"
        "After checkout, tap *Back to WhatsApp* to return to chat.",
        "Open Shop",
        url,
        header="Mini Shop",
        footer="Opens in WhatsApp",
    )


def _categories() -> list[str]:
    seen: list[str] = []
    for p in CATALOG:
        if p.category not in seen:
            seen.append(p.category)
    return seen


async def send_main_menu(wa: WhatsAppClient, to: str) -> None:
    await wa.send_buttons(
        to,
        "Welcome to our grocery shop! 👋\n\nPlease choose an option below:",
        [
            (MENU_BROWSE, "Browse products"),
            (MENU_CART, "My cart"),
            (MENU_WEBSHOP, "Open mini shop"),
        ],
        header="Shop Menu",
        footer="Cart is shared with the website",
    )
    # Shop opens via CTA button only (in-app browser) — not duplicated as plain URL


async def send_category_list(wa: WhatsAppClient, to: str) -> None:
    rows = [
        {
            "id": f"{CAT_PREFIX}{cat}",
            "title": _clip(cat.capitalize(), 24),
            "description": _clip(
                f"{sum(1 for p in CATALOG if p.category == cat)} items",
                72,
            ),
        }
        for cat in _categories()
    ]
    await wa.send_list(
        to,
        "Select a category to see products:",
        "View categories",
        [{"title": "Categories", "rows": rows[:10]}],
        header="Browse",
        footer="Single tap to open",
    )


async def send_product_list(
    wa: WhatsAppClient, to: str, *, category: str | None = None, page: int = 0
) -> None:
    items = [p for p in CATALOG if not category or p.category == category]
    start = page * PRODUCTS_PER_PAGE
    chunk = items[start : start + PRODUCTS_PER_PAGE]

    if not chunk:
        await wa.send_text(to, "No more products on this page.")
        await send_main_menu(wa, to)
        return

    title = category.capitalize() if category else "All products"
    rows = [
        {
            "id": f"{PROD_PREFIX}{p.id}",
            "title": _clip(p.name, 24),
            "description": _clip(f"₹{p.price_inr:.0f} · {p.unit} · stock {p.quantity}", 72),
        }
        for p in chunk
    ]
    sections = [{"title": _clip(title, 24), "rows": rows}]

    footer = f"Page {page + 1}"
    if start + PRODUCTS_PER_PAGE < len(items):
        footer += " — more available"

    await wa.send_list(
        to,
        f"Tap a product to add options ({len(chunk)} items):",
        "Select product",
        sections,
        header=_clip(title, 60),
        footer=_clip(footer, 60),
    )

    nav_buttons: list[tuple[str, str]] = [(MENU_HOME, "Main menu")]
    if start + PRODUCTS_PER_PAGE < len(items):
        nav_buttons.insert(0, (f"{PAGE_PREFIX}{page + 1}_{category or 'all'}", "More products"))
    if page > 0:
        nav_buttons.insert(
            0,
            (f"{PAGE_PREFIX}{page - 1}_{category or 'all'}", "Previous"),
        )
    await wa.send_buttons(to, "Navigation:", nav_buttons[:3])


async def send_product_actions(wa: WhatsAppClient, to: str, product_id: str) -> None:
    p = get_product(product_id)
    if not p:
        await wa.send_text(to, "Product not found.")
        await send_main_menu(wa, to)
        return

    await wa.send_buttons(
        to,
        f"*{p.name}*\n{p.unit} — ₹{p.price_inr:.2f}\nStock: {p.quantity}\n\nHow many to add?",
        [
            (f"{ADD1_PREFIX}{p.id}", "Add 1"),
            (f"{ADD2_PREFIX}{p.id}", "Add 2"),
            (f"{ADD3_PREFIX}{p.id}", "Add 3"),
        ],
        header="Add to cart",
    )


async def send_cart_summary(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    cart = view_cart()
    if cart.get("empty"):
        await wa.send_buttons(
            to,
            "Your cart is empty. 🛒",
            [(MENU_BROWSE, "Browse products"), (MENU_HOME, "Main menu")],
        )
        return

    body = cart.get("cart_summary", "")
    await wa.send_buttons(
        to,
        f"*Your cart*\n\n{body}",
        [
            (MENU_CHECKOUT, "Place order"),
            (MENU_BROWSE, "Add more"),
            (MENU_HOME, "Main menu"),
        ],
        header="Cart",
    )


async def send_orders_summary(wa: WhatsAppClient, to: str) -> None:
    _set_user(to)
    data = get_my_orders()
    if not data.get("orders"):
        await wa.send_buttons(
            to,
            "No orders yet.",
            [(MENU_BROWSE, "Start shopping"), (MENU_HOME, "Main menu")],
        )
        return

    lines = []
    for o in data["orders"]:
        lines.append(
            f"#{o['order_id']} — ₹{o['total_inr']:.2f} ({o['payment_status']})"
        )
    await wa.send_text(to, "*Your orders*\n\n" + "\n".join(lines))
    await wa.send_buttons(to, "Options:", [(MENU_HOME, "Main menu")])


async def handle_interactive(wa: WhatsAppClient, to: str, reply_id: str) -> None:
    _set_user(to)
    logger.info("Interactive tap from %s: %s", to, reply_id)

    if reply_id == MENU_HOME:
        await send_main_menu(wa, to)
        return
    if reply_id == MENU_WEBSHOP:
        await send_shop_website(wa, to)
        return
    if reply_id == MENU_BROWSE:
        await send_category_list(wa, to)
        return
    if reply_id == MENU_CART:
        await send_cart_summary(wa, to)
        return
    if reply_id == MENU_ORDERS:
        await send_orders_summary(wa, to)
        return
    if reply_id == MENU_CHECKOUT:
        result = place_order()
        if result.get("status") == "error":
            await wa.send_text(to, result.get("message", "Checkout failed."))
            await send_cart_summary(wa, to)
            return
        await wa.send_text(
            to,
            f"✅ Order #{result['order_id']}\nTotal: ₹{result['total_inr']:.2f}\n\n"
            f"Pay: {result['payment_link']}",
        )
        await wa.send_buttons(
            to,
            "After payment, confirm below:",
            [
                (f"{PAY_PREFIX}{result['order_id']}", "I paid"),
                (MENU_ORDERS, "My orders"),
                (MENU_HOME, "Menu"),
            ],
        )
        return
    if reply_id == MENU_SEARCH:
        await wa.send_text(to, "Type what you need, e.g. *1 litre oil* or *basmati rice*")
        return

    if reply_id.startswith(CAT_PREFIX):
        cat = reply_id[len(CAT_PREFIX) :]
        await send_product_list(wa, to, category=cat, page=0)
        return

    if reply_id.startswith(PAGE_PREFIX):
        # page_1_all or page_1_oil
        rest = reply_id[len(PAGE_PREFIX) :]
        page_str, _, cat = rest.partition("_")
        try:
            page = int(page_str)
        except ValueError:
            page = 0
        category = None if cat == "all" else cat
        await send_product_list(wa, to, category=category, page=page)
        return

    if reply_id.startswith(PROD_PREFIX):
        await send_product_actions(wa, to, reply_id[len(PROD_PREFIX) :])
        return

    if reply_id.startswith(ADD1_PREFIX):
        pid = reply_id[len(ADD1_PREFIX) :]
        r = add_to_cart(pid, 1)
        await wa.send_text(to, r.get("message", "Done"))
        await send_cart_summary(wa, to)
        return

    if reply_id.startswith(ADD2_PREFIX):
        pid = reply_id[len(ADD2_PREFIX) :]
        r = add_to_cart(pid, 2)
        await wa.send_text(to, r.get("message", "Done"))
        await send_cart_summary(wa, to)
        return

    if reply_id.startswith(ADD3_PREFIX):
        pid = reply_id[len(ADD3_PREFIX) :]
        r = add_to_cart(pid, 3)
        await wa.send_text(to, r.get("message", "Done"))
        await send_cart_summary(wa, to)
        return

    if reply_id.startswith(PAY_PREFIX):
        oid = reply_id[len(PAY_PREFIX) :]
        r = confirm_payment(oid)
        await wa.send_text(to, r.get("message", "Updated"))
        await send_main_menu(wa, to)
        return

    await wa.send_text(to, "Unknown option. Opening menu…")
    await send_main_menu(wa, to)


async def handle_text(wa: WhatsAppClient, to: str, text: str, runner) -> None:
    _set_user(to)

    if wants_menu(text):
        await send_main_menu(wa, to)
        return

    if wants_browse(text):
        await send_category_list(wa, to)
        return

    if _normalize(text) in {"shop", "website", "web", "online", "mini shop", "open shop"}:
        await send_shop_website(wa, to)
        return

    if _normalize(text) in {"cart", "my cart", "mera cart"}:
        await send_cart_summary(wa, to)
        return

    if _normalize(text) in {"orders", "my orders", "mera order"}:
        await send_orders_summary(wa, to)
        return

    # Free-text → AI agent, then show menu buttons again
    try:
        reply = await runner.chat(to, text)
        await wa.send_text(to, reply[:4000])
    except Exception:
        logger.exception("Agent failed for %s", to)
        await wa.send_text(to, "Sorry, could not process that. Try the menu below.")
    await wa.send_buttons(
        to,
        "Quick actions:",
        [
            (MENU_BROWSE, "Browse products"),
            (MENU_CART, "My cart"),
            (MENU_HOME, "Menu"),
        ],
    )


def parse_interactive_reply(message: dict) -> str | None:
    interactive = message.get("interactive") or {}
    if interactive.get("type") == "list_reply":
        return interactive.get("list_reply", {}).get("id")
    if interactive.get("type") == "button_reply":
        return interactive.get("button_reply", {}).get("id")
    return None
