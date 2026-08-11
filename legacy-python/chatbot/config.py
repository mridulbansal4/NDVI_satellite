"""
chatbot/config.py — Configuration Loader
"""

import os
from pathlib import Path
from dotenv import load_dotenv

_env_path = Path(__file__).parent.parent / ".env"
load_dotenv(dotenv_path=_env_path, override=False)

# ── Ollama ────────────────────────────────────────────────────────────────────
OLLAMA_BASE_URL: str  = os.getenv("OLLAMA_BASE_URL")
OLLAMA_MODEL: str     = os.getenv("OLLAMA_MODEL")

OLLAMA_TEMPERATURE: float = float(os.getenv("OLLAMA_TEMPERATURE", "0.7"))
OLLAMA_MAX_TOKENS: int    = int(os.getenv("OLLAMA_MAX_TOKENS",    "512"))

# ── Session management ────────────────────────────────────────────────────────
CHATBOT_MAX_HISTORY: int = int(os.getenv("CHATBOT_MAX_HISTORY", "20"))

# ── Logging ───────────────────────────────────────────────────────────────────
CHATBOT_LOG_LEVEL: str = os.getenv("CHATBOT_LOG_LEVEL", "INFO")
