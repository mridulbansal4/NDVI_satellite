"""
utils/pincode.py — India Post PIN code API caller.
Resolves state, district, and taluka from a 6-digit Indian PIN code.
API: https://api.postalpincode.in/pincode/{pincode}
"""

import logging
import requests

logger = logging.getLogger(__name__)

PINCODE_API_BASE = "https://api.postalpincode.in/pincode"
REQUEST_TIMEOUT_S = 8


def resolve_pincode(pin_code: str) -> dict:
    """
    Calls the India Post PIN code API and extracts state, district, and taluka.

    Returns:
        {
            "state": "Maharashtra",
            "district": "Nashik",
            "taluka": "Nashik"   (Block/Taluka from PostOffice record)
        }

    Raises:
        ValueError: If the PIN code is invalid or the API returns no results.
        RuntimeError: If the API call fails (network error, timeout).
    """
    if not pin_code or len(pin_code) != 6 or not pin_code.isdigit():
        raise ValueError(f"Invalid PIN code format: '{pin_code}'. Must be exactly 6 digits.")

    url = f"{PINCODE_API_BASE}/{pin_code}"
    try:
        resp = requests.get(url, timeout=REQUEST_TIMEOUT_S)
        resp.raise_for_status()
        data = resp.json()
    except requests.RequestException as exc:
        logger.error("[Pincode] API call failed for %s: %s", pin_code, exc)
        raise RuntimeError(f"PIN code API unreachable: {exc}") from exc

    # API response structure: [{ "Status": "Success", "PostOffice": [...] }]
    if not data or data[0].get("Status") != "Success":
        raise ValueError(f"PIN code {pin_code} not found or invalid.")

    post_offices = data[0].get("PostOffice") or []
    if not post_offices:
        raise ValueError(f"No post office data found for PIN code {pin_code}.")

    # Use the first post office record for state/district/taluka
    po = post_offices[0]
    result = {
        "state":    po.get("State", ""),
        "district": po.get("District", ""),
        "taluka":   po.get("Block", ""),   # "Block" maps to Taluka in India Post API
    }
    logger.info("[Pincode] Resolved %s → %s", pin_code, result)
    return result
