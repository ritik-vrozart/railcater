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
    whatsapp_verify_token: str = ""
    whatsapp_api_version: str = "v21.0"
    whatsapp_graph_api_base: str = "https://graph.facebook.com"
    whatsapp_app_id: str = ""
    whatsapp_app_secret: str = ""
    whatsapp_token_file: str = ""
    whatsapp_wa_me_number: str = ""

    app_name: str = "whatsapp_train_food_agent"
    host: str = "0.0.0.0"
    port: int = 8000
    debug: bool = False

    # Public URL (HTTPS) — WhatsApp webhook, shop links (no trailing slash)
    public_base_url: str = ""

    # Go backend API (no trailing slash)
    api_base_url: str = ""

    # Must match api/.env AGENT_NOTIFY_SECRET
    agent_notify_secret: str = ""

    # Fallback menu photo when vendor item has no image_url (HTTPS)
    default_menu_image_url: str = ""

    # Legacy grocery shop payment link base (optional)
    payment_link_base_url: str = ""

    # Shown in bot help / demos (optional)
    demo_train_number: str = ""


settings = Settings()
