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


def get_firestore_client():
    """
    Returns a Firestore client. Initializes Firebase Admin SDK on first call.
    Supports two auth modes:
      1. GOOGLE_APPLICATION_CREDENTIALS env var (production / CI)
      2. serviceAccountKey.json in project root (local development)
    """
    global _db
    if _db is not None:
        return _db

    if not firebase_admin._apps:
        # Try service account key file first (local dev)
        key_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.dirname(__file__))),
            "serviceAccountKey.json"
        )
        if os.path.exists(key_path):
            cred = credentials.Certificate(key_path)
            logger.info("[Firestore] Using serviceAccountKey.json")
        else:
            # Fall back to GOOGLE_APPLICATION_CREDENTIALS
            cred = credentials.ApplicationDefault()
            logger.info("[Firestore] Using Application Default Credentials")

        firebase_admin.initialize_app(cred, {
            "projectId": os.getenv("FIREBASE_PROJECT_ID")
        })

    _db = fs.client()
    logger.info("[Firestore] Client initialized.")
    return _db
