"""Handle natural-language food orders in WhatsApp train flow."""

from __future__ import annotations

import logging

from services.food_intent import (
    looks_like_food_order,
    parse_food_intent,
)
from services.whatsapp import WhatsAppClient
from store.session import FlowStep, get_session
from tools import train_tools

logger = logging.getLogger(__name__)


async def send_smart_order_prompt(wa: WhatsAppClient, to: str) -> None:
    await wa.send_text(
        to,
        "✍️ *Kya order karna hai?* (likh kar bhejein)\n\n"
        "Examples:\n"
        "• mujhe paneer thali chahiye\n"
        "• veg meal\n"
        "• 2 chai\n"
        "• water bottle\n\n"
        "Ya neeche *category list* se choose karein.",
    )


async def handle_food_message(
    wa: WhatsAppClient,
    to: str,
    text: str,
    *,
    send_menu_list_fn,
    send_portion_actions_fn,
    send_category_list_fn,
) -> bool:
    """
    Parse & fulfil a natural-language food request.
    Returns True if handled.
    Veg/non-veg rules live in the agent prompt; guided flow does not ask Veg/Non-veg buttons.
    """
    if not looks_like_food_order(text):
        return False

    intent = parse_food_intent(text)
    if not intent:
        return False

    sess = get_session(to)
    if not sess.vendor_id:
        return False

    sess.flow_step = FlowStep.ORDERING
    sess.touch_journey()

    search = train_tools.search_menu(
        text,
        veg=intent.veg_preference,
        limit=10,
    )
    if search.get("status") == "error":
        await wa.send_text(to, f"❌ {search.get('message')}")
        return True

    portions = search.get("portions") or []

    if not portions:
        await wa.send_text(
            to,
            f"❌ *{intent.item_query}* menu par nahi mila.\n\n"
            "Dusra naam try karein ya category list se choose karein.",
        )
        sess.category_id = None
        sess.category_name = None
        await send_category_list_fn(wa, to)
        return True

    best = portions[0]
    if best.get("category_id"):
        train_tools.select_category(str(best["category_id"]))
        sess.category_name = best.get("category_name") or sess.category_name

    if len(portions) == 1:
        p = portions[0]
        qty = intent.quantity or 1
        if qty == 1:
            await wa.send_text(
                to,
                f"✅ Mil gaya: *{p['item_name']}* ({p['portion_label']}) — ₹{p['price_inr']:.0f}",
            )
            await send_portion_actions_fn(wa, to, p["menu_portion_id"])
            return True

        r = train_tools.add_meal_to_cart(p["menu_portion_id"], qty)
        await wa.send_text(to, r.get("message", "Added ✅"))
        return True

    if len(portions) <= 3:
        await wa.send_text(
            to,
            f"🔍 *{intent.item_query}* — {len(portions)} option(s):",
        )
        for p in portions:
            await send_portion_actions_fn(wa, to, p["menu_portion_id"])
        return True

    await wa.send_text(
        to,
        f"🔍 *{intent.item_query}* — {len(portions)} items mile:",
    )
    await send_menu_list_fn(wa, to, portions=portions, page=0)
    return True
