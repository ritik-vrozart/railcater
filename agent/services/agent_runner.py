from __future__ import annotations

import logging
import os

from google.adk.runners import Runner
from google.adk.sessions import InMemorySessionService
from google.genai import types

from agents.shop_agent import root_agent
from config import settings
from tools import train_tools

logger = logging.getLogger(__name__)

APP_NAME = settings.app_name


class ShopAgentRunner:
    """Runs the ADK shop agent per WhatsApp user (in-memory sessions, no DB)."""

    def __init__(self) -> None:
        if settings.google_api_key:
            os.environ["GOOGLE_API_KEY"] = settings.google_api_key
        os.environ.setdefault("GOOGLE_GENAI_USE_VERTEXAI", str(settings.google_genai_use_vertexai).lower())

        self._session_service = InMemorySessionService()
        self._runner = Runner(
            app_name=APP_NAME,
            agent=root_agent,
            session_service=self._session_service,
            auto_create_session=True,
        )

    def _session_id(self, user_id: str) -> str:
        return f"wa_{user_id}"

    async def chat(self, user_id: str, message: str) -> str:
        train_tools.set_current_user(user_id)
        session_id = self._session_id(user_id)

        content = types.Content(role="user", parts=[types.Part(text=message)])
        final_text: str | None = None

        try:
            async for event in self._runner.run_async(
                user_id=user_id,
                session_id=session_id,
                new_message=content,
            ):
                if event.is_final_response():
                    if event.content and event.content.parts:
                        part = event.content.parts[0]
                        if hasattr(part, "text") and part.text:
                            final_text = part.text
                    break
        except Exception:
            logger.exception("Agent run failed for user %s", user_id)
            raise

        if not final_text:
            return "Sorry, I could not process that. Please try again."
        return final_text.strip()


_runner: ShopAgentRunner | None = None


def get_agent_runner() -> ShopAgentRunner:
    global _runner
    if _runner is None:
        _runner = ShopAgentRunner()
    return _runner
