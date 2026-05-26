"""Per-user ordering session (train, pantry/vendor, cart)."""

from __future__ import annotations

import time
from dataclasses import dataclass, field
from typing import Any

# Keep train / seat / pantry context while user browses cart (minutes)
JOURNEY_TTL_SECONDS = 15 * 60


@dataclass
class TrainCartLine:
    menu_portion_id: str
    item_name: str
    portion_label: str
    quantity: int
    unit_price_cents: int

    @property
    def line_total_cents(self) -> int:
        return self.unit_price_cents * self.quantity


class FlowStep:
    IDLE = "idle"
    AWAITING_TRAIN_NUMBER = "awaiting_train_number"
    AWAITING_NAME = "awaiting_name"
    AWAITING_SEAT = "awaiting_seat"
    AWAITING_CATEGORY = "awaiting_category"
    AWAITING_VEG = "awaiting_veg"
    ORDERING = "ordering"  # menu + cart
    # Legacy PNR flow (unused by WhatsApp guided UI)
    AWAITING_PNR = "awaiting_pnr"
    AWAITING_STATION = "awaiting_station"
    AWAITING_VENDOR = "awaiting_vendor"
    AWAITING_SEAT_TEXT = "awaiting_seat_text"


@dataclass
class UserSession:
    user_id: str
    customer_id: str | None = None
    flow_step: str = FlowStep.IDLE
    awaiting_pnr: bool = False
    train_number: str | None = None
    train_id: str | None = None
    train_name: str | None = None
    passenger_name: str | None = None
    category_id: str | None = None
    category_name: str | None = None
    menu_list_page: int = 0
    pending_food_query: str | None = None
    pending_quantity: int | None = None
    pnr: str | None = None
    pnr_lookup: dict[str, Any] | None = None
    station_id: str | None = None
    station_name: str | None = None
    vendor_id: str | None = None
    vendor_name: str | None = None
    coach: str | None = None
    berth: str | None = None
    delivery_window: dict[str, Any] | None = None
    cart_lines: list[TrainCartLine] = field(default_factory=list)
    last_order_id: str | None = None
    journey_updated_at: float | None = None

    @property
    def cart_total_cents(self) -> int:
        return sum(line.line_total_cents for line in self.cart_lines)

    def clear_cart(self) -> None:
        self.cart_lines = []

    def touch_journey(self) -> None:
        """Refresh TTL so Add more / cart keeps same train context."""
        self.journey_updated_at = time.time()

    def journey_expired(self) -> bool:
        if self.journey_updated_at is None:
            return False
        return (time.time() - self.journey_updated_at) > JOURNEY_TTL_SECONDS

    def has_active_journey(self) -> bool:
        """Train + pantry locked in for ~15 min after first lookup."""
        if not self.train_number or not self.vendor_id:
            return False
        return not self.journey_expired()

    def reset_journey(self) -> None:
        """Keep user id but clear ordering journey state."""
        self.flow_step = FlowStep.IDLE
        self.awaiting_pnr = False
        self.train_number = None
        self.train_id = None
        self.train_name = None
        self.passenger_name = None
        self.category_id = None
        self.category_name = None
        self.menu_list_page = 0
        self.pending_food_query = None
        self.pending_quantity = None
        self.pnr = None
        self.pnr_lookup = None
        self.station_id = None
        self.station_name = None
        self.vendor_id = None
        self.vendor_name = None
        self.coach = None
        self.berth = None
        self.delivery_window = None
        self.clear_cart()
        self.journey_updated_at = None


_sessions: dict[str, UserSession] = {}


def get_session(user_id: str) -> UserSession:
    if user_id not in _sessions:
        _sessions[user_id] = UserSession(user_id=user_id)
    sess = _sessions[user_id]
    if sess.train_number and sess.journey_expired():
        sess.reset_journey()
    return sess
