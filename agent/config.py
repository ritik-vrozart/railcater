from pathlib import Path

from dotenv import load_dotenv
from pydantic_settings import BaseSettings, SettingsConfigDict

# Always load agent/.env before Settings() — avoids stale shell env / wrong cwd
_ENV_PATH = Path(__file__).resolve().parent / ".env"
load_dotenv(_ENV_PATH, override=True)


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=str(_ENV_PATH),
        env_file_encoding="utf-8",
        extra="ignore",
    )

    google_api_key: str = ""
    google_genai_use_vertexai: bool = False

    whatsapp_access_token: str = ""
    whatsapp_phone_number_id: str = ""
    whatsapp_verify_token: str = "whatsapp_bot_verify"
    whatsapp_api_version: str = "v21.0"
    # Business number for wa.me "back to chat" (digits only, e.g. 15551919669)
    whatsapp_wa_me_number: str = "15551919669"

    app_name: str = "whatsapp_shop_agent"
    host: str = "0.0.0.0"
    port: int = 8000
    debug: bool = False

    # Public URL for mini shop link in WhatsApp (your ngrok URL, no trailing slash)
    public_base_url: str = "http://localhost:8000"

    # Go backend API (train food orders, PNR, menu)
    api_base_url: str = "http://localhost:8080"

    # Shared secret for Go API → agent delivery notifications
    agent_notify_secret: str = "dev-notify-secret"


settings = Settings()
