"""
firestore/session.py — Firestore session helpers.

Collections managed:
  /otp_sessions/{mobile_number}   — OTP tokens with TTL
  /farmer_sessions/{farmer_id}    — Onboarding resume state
  /farm_alerts/{farm_id}          — Real-time VI dashboard alerts
"""

import logging
from datetime import datetime, timedelta, timezone
from google.cloud.firestore import SERVER_TIMESTAMP
from firestore.client import get_firestore_client

logger = logging.getLogger(__name__)

# ─────────────────────────────────────────────────────────────────────────────
# OTP Sessions   /otp_sessions/{mobile_number}
# ─────────────────────────────────────────────────────────────────────────────

def write_otp_session(mobile_number: str, otp: str) -> None:
    """
    Creates or overwrites an OTP session document.
    TTL: expires_at set to NOW + 10 minutes (Firestore TTL policy on this field).
    """
    db = get_firestore_client()
    expires_at = datetime.now(timezone.utc) + timedelta(minutes=10)
    db.collection("otp_sessions").document(mobile_number).set({
        "otp": otp,
        "expires_at": expires_at,
        "verified": False,
    })
    logger.info("[Firestore] OTP session written for %s", mobile_number)


def read_otp_session(mobile_number: str) -> dict | None:
    """Reads the OTP session document for a mobile number. Returns None if not found."""
    db = get_firestore_client()
    doc = db.collection("otp_sessions").document(mobile_number).get()
    return doc.to_dict() if doc.exists else None


def mark_otp_verified(mobile_number: str) -> None:
    """Marks the OTP session as verified."""
    db = get_firestore_client()
    db.collection("otp_sessions").document(mobile_number).update({"verified": True})


# ─────────────────────────────────────────────────────────────────────────────
# Farmer Sessions   /farmer_sessions/{farmer_id}
# ─────────────────────────────────────────────────────────────────────────────

def write_session(farmer_id: str, current_step: int, partial_data: dict) -> None:
    """
    Writes or updates the onboarding session for a farmer.
    Called after every step form submission to support low-connectivity resume.

    Document structure:
      {
        "current_step": 3,
        "partial_data": { "name": "Ramesh", "pin_code": "422001" },
        "last_active": <Firestore SERVER_TIMESTAMP>
      }
    """
    try:
        db = get_firestore_client()
        db.collection("farmer_sessions").document(farmer_id).set({
            "current_step": current_step,
            "partial_data": partial_data,
            "last_active": SERVER_TIMESTAMP,
        }, merge=True)
        logger.info("[Firestore] Session written for farmer %s at step %d", farmer_id, current_step)
    except Exception as e:
        logger.warning("[Firestore] Session write failed (non-fatal): %s", e)


def read_session(farmer_id: str) -> dict | None:
    """
    Reads the onboarding session for a farmer.
    Returns None if no session exists (farmer is new or already committed).
    """
    try:
        db = get_firestore_client()
        doc = db.collection("farmer_sessions").document(farmer_id).get()
        if doc.exists:
            logger.info("[Firestore] Session read for farmer %s", farmer_id)
            return doc.to_dict()
    except Exception as e:
        logger.warning("[Firestore] Session read failed (non-fatal): %s", e)
    return None


def delete_session(farmer_id: str) -> None:
    """
    Deletes the onboarding session after Step 9 (consent) is submitted.
    Signals that farmer data has been committed to PostgreSQL.
    """
    try:
        db = get_firestore_client()
        db.collection("farmer_sessions").document(farmer_id).delete()
        logger.info("[Firestore] Session deleted for farmer %s (onboarding complete)", farmer_id)
    except Exception as e:
        logger.warning("[Firestore] Session delete failed (non-fatal): %s", e)


# ─────────────────────────────────────────────────────────────────────────────
# Farm Alerts   /farm_alerts/{farm_id}
# ─────────────────────────────────────────────────────────────────────────────

def write_farm_alert(farm_id: str, vi_data: dict) -> None:
    """
    Writes a farm alert document after the VI Engine inserts a new vi_report row.
    This powers reactive dashboard updates without SQL polling.

    Document structure:
      {
        "crop_health": "Good",
        "vegetation_index": 0.74,
        "water_stress": "Low",
        "weather_alert": "None",
        "pest_risk": "Low",
        "last_updated": <SERVER_TIMESTAMP>
      }
    """
    db = get_firestore_client()

    cvi_mean = vi_data.get("cvi_mean", 0)
    if cvi_mean >= 0.6:
        crop_health = "Good"
    elif cvi_mean >= 0.4:
        crop_health = "Moderate"
    else:
        crop_health = "Poor"

    ndmi = vi_data.get("ndmi", 0)
    water_stress = "Low" if ndmi >= 0.3 else ("Moderate" if ndmi >= 0.1 else "High")

    try:
        db.collection("farm_alerts").document(farm_id).set({
            "crop_health": crop_health,
            "vegetation_index": round(vi_data.get("ndvi", 0), 4),
            "water_stress": water_stress,
            "weather_alert": "None",
            "pest_risk": "Low",
            "last_updated": SERVER_TIMESTAMP,
        })
        logger.info("[Firestore] Farm alert written for farm %s (health=%s)", farm_id, crop_health)
    except Exception as e:
        logger.warning("[Firestore] Farm alert write failed (non-fatal): %s", e)


def read_farm_alert(farm_id: str) -> dict | None:
    """Reads the latest farm alert for a farm."""
    try:
        db = get_firestore_client()
        doc = db.collection("farm_alerts").document(farm_id).get()
        return doc.to_dict() if doc.exists else None
    except Exception as e:
        logger.warning("[Firestore] Farm alert read failed (non-fatal): %s", e)
        return None
