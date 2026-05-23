"""Per-user ordering session (PNR, station, vendor, train-food cart)."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


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
    AWAITING_PNR = "awaiting_pnr"
    AWAITING_STATION = "awaiting_station"
    AWAITING_VENDOR = "awaiting_vendor"
    AWAITING_SEAT = "awaiting_seat"
    AWAITING_SEAT_TEXT = "awaiting_seat_text"
    ORDERING = "ordering"  # menu + cart


@dataclass
class UserSession:
    user_id: str
    customer_id: str | None = None
    flow_step: str = FlowStep.IDLE
    awaiting_pnr: bool = False
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

    @property
    def cart_total_cents(self) -> int:
        return sum(line.line_total_cents for line in self.cart_lines)

    def clear_cart(self) -> None:
        self.cart_lines = []

    def reset_journey(self) -> None:
        """Keep user id but clear PNR journey state."""
        self.flow_step = FlowStep.IDLE
        self.awaiting_pnr = False
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


_sessions: dict[str, UserSession] = {}


def get_session(user_id: str) -> UserSession:
    if user_id not in _sessions:
        _sessions[user_id] = UserSession(user_id=user_id)
    return _sessions[user_id]
