"""
Natural-language food ordering — maps Hindi/English requests to menu items & categories.

Examples:
  "mujhe paneer thali chahiye" → Meals + veg + search "paneer thali"
  "2 chai" → Beverages + item chai
  "water bottle" → Beverages/water, not thali
"""

from __future__ import annotations

import re
from dataclasses import dataclass


@dataclass
class FoodIntent:
    item_query: str
    category_hints: list[str]
    veg_preference: str | None  # "veg" | "non_veg"
    quantity: int | None


_FILLER = re.compile(
    r"\b("
    r"i|want|need|please|mujhe|mere|ko|ke|liye|chahiye|chaiye|mangao|manga|do|de|dena|"
    r"order|kar|karo|dijiye|dijie|give|get|some|pls|ya|aur|bhi|ek|one|two|three|"
    r"|\d+\s*(x|piece|pieces|plate|plates)?"
    r")\b",
    re.I,
)

_QUANTITY = re.compile(
    r"(?:^|\s)(\d{1,2})\s*(?:x|plate|plates|piece|pieces|cup|cups|bottle|bottles)?\s*",
    re.I,
)

_CATEGORY_KEYWORDS: dict[str, list[str]] = {
    "meals": [
        "meal", "meals", "thali", "khana", "lunch", "dinner", "breakfast", "plate",
        "rice", "dal", "roti", "sabzi", "curry", "biryani", "paratha", "combo",
    ],
    "snacks": [
        "snack", "snacks", "sandwich", "samosa", "pakora", "biscuit", "chips",
        "maggi", "noodles",
    ],
    "beverages": [
        "chai", "tea", "coffee", "drink", "beverage", "juice", "cold drink",
        "pepsi", "coke", "soda", "lassi", "milk",
    ],
    "water": ["water", "bottle", "pani", "mineral", "aquafina", "bisleri"],
}

# Only explicit dietary labels — not dish names (paneer, dal, etc.)
_VEG_WORDS = {"veg", "vegetarian", "shakahari"}
_NONVEG_WORDS = {
    "non veg", "nonveg", "non-veg", "egg", "chicken", "mutton", "fish", "meat",
}


def _normalize(text: str) -> str:
    t = text.strip().lower()
    t = re.sub(r"[^\w\s]", " ", t)
    return re.sub(r"\s+", " ", t).strip()


def parse_food_intent(text: str) -> FoodIntent | None:
    """Return None if text does not look like a food order."""
    raw = text.strip()
    if len(raw) < 3:
        return None

    t = _normalize(raw)
    if not t:
        return None

    # Skip pure navigation / meta
    skip = {
        "hi", "hello", "menu", "cart", "help", "show image", "show images", "order food",
        "add more", "place order", "main menu", "yes", "no", "ok", "thanks",
    }
    if t in skip:
        return None

    quantity: int | None = None
    qm = _QUANTITY.search(t)
    if qm:
        quantity = int(qm.group(1))
        if 1 <= quantity <= 20:
            t = _QUANTITY.sub(" ", t, count=1).strip()

    veg: str | None = None
    if any(w in t for w in _NONVEG_WORDS):
        veg = "non_veg"
    elif any(w in t for w in _VEG_WORDS):
        veg = "veg"

    category_hints: list[str] = []
    for cat, words in _CATEGORY_KEYWORDS.items():
        if any(w in t for w in words):
            category_hints.append(cat)

    # Item query = user words minus fillers (keep paneer, thali, chai, etc.)
    item_t = _FILLER.sub(" ", t)
    item_t = re.sub(r"\s+", " ", item_t).strip()

    if not item_t and category_hints:
        item_t = category_hints[0]

    if not item_t and not category_hints:
        if not any(w in t for w in ("chahiye", "chaiye", "want", "need", "mangao", "order")):
            return None
        item_t = t

    if not category_hints and item_t:
        if any(w in item_t for w in ("thali", "paneer", "dal", "rice", "biryani", "meal")):
            category_hints.append("meals")
        if any(w in item_t for w in ("chai", "coffee", "tea", "lassi")):
            category_hints.append("beverages")
        if any(w in item_t for w in ("water", "bottle", "pani")):
            category_hints.append("water")

    return FoodIntent(
        item_query=item_t or raw.lower(),
        category_hints=category_hints,
        veg_preference=veg,
        quantity=quantity,
    )


def _tokens(s: str) -> set[str]:
    return {x for x in _normalize(s).split() if len(x) > 1}


def score_item_match(query: str, item_name: str, description: str = "") -> float:
    """0–1 relevance score."""
    q = _normalize(query)
    name = _normalize(item_name)
    if not q or not name:
        return 0.0
    if q in name or name in q:
        return 0.95
    qt, nt = _tokens(q), _tokens(name)
    if not qt:
        return 0.0
    overlap = len(qt & nt) / len(qt)
    if overlap >= 0.5:
        return 0.7 + overlap * 0.25
    # partial substring per token
    partial = sum(1 for tok in qt if any(tok in ntok or ntok in tok for ntok in nt))
    if partial:
        return 0.4 + 0.15 * partial
    desc = _normalize(description)
    if desc and any(tok in desc for tok in qt):
        return 0.35
    return 0.0


def category_name_matches(hints: list[str], category_name: str) -> bool:
    cn = _normalize(category_name)
    for hint in hints:
        if hint in cn or cn in hint:
            return True
        for kw in _CATEGORY_KEYWORDS.get(hint, []):
            if kw in cn:
                return True
    return False


def looks_like_food_order(text: str) -> bool:
    return parse_food_intent(text) is not None
