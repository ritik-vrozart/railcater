#!/usr/bin/env python3
"""Naya (abhi-abhi generate) short-lived token lo → long-lived (~60 din) save karo.

Graph API Explorer se token copy karte hi turant chalao (expire hone se pehle).

  cd agent
  python scripts/renew_whatsapp_token.py "EAAB..."

Permanent token (kabhi expire nahi):
  python scripts/save_whatsapp_token.py "SYSTEM_USER_TOKEN"
"""

from __future__ import annotations

import asyncio
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from services.whatsapp_token import (
    WhatsAppAuthError,
    debug_token_info,
    exchange_long_lived,
    persist_token,
    save_token_file,
    token_file_path,
)


async def main() -> None:
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    raw = sys.argv[1].strip()
    print("Exchanging for long-lived token…")
    try:
        long_lived = await exchange_long_lived(raw)
    except WhatsAppAuthError as exc:
        print(f"FAILED: {exc}")
        sys.exit(1)

    save_token_file(long_lived)
    persist_token(long_lived)

  # Update hint for manual .env paste
    path = token_file_path()
    print(f"OK — saved to {path}")
    print(f"Token length: {len(long_lived)}")
    print("Optional: WHATSAPP_ACCESS_TOKEN=.env me bhi yahi paste karo, phir uvicorn restart.")

    info = await debug_token_info(long_lived)
    if info:
        exp = info.get("expires_at", 0)
        if exp == 0:
            print("Type: permanent (System User)")
        else:
            import time

            days = max(0, (int(exp) - int(time.time())) // 86400)
            print(f"Valid ~{days} more days")


if __name__ == "__main__":
    asyncio.run(main())
