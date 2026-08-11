"""
firestore/client.py — Firebase Admin SDK singleton initialization.
Uses GOOGLE_APPLICATION_CREDENTIALS env var or serviceAccountKey.json.
"""

import os
import logging
import firebase_admin
from firebase_admin import credentials, firestore as fs

logger = logging.getLogger(__name__)

_db = None
_init_failed = False  # cached once Firestore is found to be unconfigured (dev)


def get_firestore_client():
    """
    Returns a Firestore client. Initializes Firebase Admin SDK on first call.
    Supports two auth modes:
      1. serviceAccountKey.json in project root (local development)
      2. GOOGLE_APPLICATION_CREDENTIALS env var (production / CI)

    If neither is present (typical local dev), this raises immediately and
    caches that result, so callers fail fast instead of paying the multi-second
    Application Default Credentials metadata-server probe on every request.
    """
    global _db, _init_failed
    if _db is not None:
        return _db
    if _init_failed:
        raise RuntimeError("Firestore is not configured (no credentials).")

    try:
        if not firebase_admin._apps:
            # Try service account key file first (local dev)
            key_path = os.path.join(
                os.path.dirname(os.path.dirname(os.path.dirname(__file__))),
                "serviceAccountKey.json"
            )
            if os.path.exists(key_path):
                cred = credentials.Certificate(key_path)
                logger.info("[Firestore] Using serviceAccountKey.json")
            elif os.getenv("GOOGLE_APPLICATION_CREDENTIALS"):
                # Production / CI — explicit ADC. Avoid blind ADC discovery in dev.
                cred = credentials.ApplicationDefault()
                logger.info("[Firestore] Using Application Default Credentials")
            else:
                _init_failed = True
                raise RuntimeError(
                    "Firestore disabled: no serviceAccountKey.json and "
                    "GOOGLE_APPLICATION_CREDENTIALS not set."
                )

            firebase_admin.initialize_app(cred, {
                "projectId": os.getenv("FIREBASE_PROJECT_ID")
            })

        _db = fs.client()
        logger.info("[Firestore] Client initialized.")
        return _db
    except Exception:
        _init_failed = True
        raise
