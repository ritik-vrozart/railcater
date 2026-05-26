#!/usr/bin/env python3
"""Save a permanent WhatsApp token to agent/data/whatsapp_token.txt

Usage:
  python scripts/save_whatsapp_token.py "YOUR_SYSTEM_USER_OR_LONG_LIVED_TOKEN"

Get a permanent token: Meta Business Suite → Business settings → System users
→ Generate token with whatsapp_business_messaging permission.
"""

from __future__ import annotations

import asyncio
import sys
from pathlib import Path

# Allow running from agent/
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from services.whatsapp_token import debug_token_info, persist_token, token_file_path


async def main() -> None:
    if len(sys.argv) < 2:
        print("Usage: python scripts/save_whatsapp_token.py <ACCESS_TOKEN>")
        sys.exit(1)

    token = persist_token(sys.argv[1])
    path = token_file_path()
    print(f"Saved token ({len(token)} chars) to {path}")

    info = await debug_token_info(token)
    if info:
        exp = info.get("expires_at", 0)
        if exp == 0:
            print("Token type: non-expiring (permanent System User token)")
        else:
            print(f"Token expires_at (unix): {exp}")
        print(f"Valid: {info.get('is_valid')}")
    else:
        print("Set WHATSAPP_APP_ID and WHATSAPP_APP_SECRET in .env to inspect token expiry.")


if __name__ == "__main__":
    asyncio.run(main())
