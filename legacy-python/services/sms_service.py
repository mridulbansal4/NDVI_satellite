"""
services/sms_service.py — SMS OTP via National Bulk SMS
=========================================================
Generates a 6-digit OTP, sends it via nationalbulksms.com,
and stores it in-memory with a 10-minute expiry for verification.
"""

import os
import random
import time
import logging
import requests

logger = logging.getLogger("app.sms")

_SMS_API_URL = "https://sms.nationalbulksms.com/fe/api/v1/send"
_OTP_EXPIRY_SECONDS = 600   # 10 minutes

# In-memory store: { "919876543210": {"otp": "123456", "expires_at": 1234567890} }
_otp_store: dict[str, dict] = {}


def _credentials():
    return {
        "username":     os.getenv("SMS_USERNAME", ""),
        "password":     os.getenv("SMS_PASSWORD", ""),
        "from":         os.getenv("SMS_FROM", ""),
        "dltContentId": os.getenv("SMS_DLT_CONTENT_ID", ""),
        "dltPeid":      os.getenv("SMS_DLT_PE_ID", ""),
    }


def _e164(phone: str) -> str:
    """Normalise a 10-digit Indian number to E.164 without the + prefix."""
    digits = phone.replace(" ", "").replace("+", "").replace("-", "")
    if digits.startswith("91") and len(digits) == 12:
        return digits
    if len(digits) == 10:
        return f"91{digits}"
    return digits


def send_otp(phone: str) -> str:
    """
    Generate a 6-digit OTP, SMS it to `phone`, store it, and return the OTP.
    Raises RuntimeError if the SMS gateway returns a failure.
    """
    otp = str(random.randint(100000, 999999))
    e164 = _e164(phone)

    creds = _credentials()
    text = (
        f"Dear Customer, your LaundryLy verification OTP is {otp}. "
        f"It is valid for 10 minutes. Do not share this OTP with anyone. - Regards LaundryLy"
    )

    params = {
        "username":     creds["username"],
        "password":     creds["password"],
        "unicode":      "false",
        "from":         creds["from"],
        "to":           e164,
        "text":         text,
        "dltContentId": creds["dltContentId"],
        "dltPeid":      creds["dltPeid"],
    }

    try:
        resp = requests.get(_SMS_API_URL, params=params, timeout=10)
        logger.info("SMS API response [%s]: %s", resp.status_code, resp.text[:120])
        if not resp.ok:
            raise RuntimeError(f"SMS gateway error {resp.status_code}: {resp.text[:120]}")
    except requests.RequestException as exc:
        logger.error("SMS send failed: %s", exc)
        raise RuntimeError("Could not reach SMS gateway. Try again.") from exc

    _otp_store[e164] = {
        "otp":        otp,
        "expires_at": time.time() + _OTP_EXPIRY_SECONDS,
    }
    logger.info("OTP stored for %s (expires in %ds)", e164, _OTP_EXPIRY_SECONDS)
    return otp


def verify_otp(phone: str, otp: str) -> bool:
    """
    Return True if `otp` matches the stored code for `phone` and hasn't expired.
    Deletes the entry on a successful match (single-use).
    """
    e164 = _e164(phone)
    record = _otp_store.get(e164)

    if not record:
        return False
    if time.time() > record["expires_at"]:
        _otp_store.pop(e164, None)
        return False
    if record["otp"] != otp.strip():
        return False

    _otp_store.pop(e164, None)
    return True
