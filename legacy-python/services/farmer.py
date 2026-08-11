"""
services/farmer.py — Business logic for farmer profile and location steps.
"""

import logging
from repositories import farmer as farmer_repo
from firestore.session import write_session
from utils.pincode import resolve_pincode

logger = logging.getLogger(__name__)


def update_basic_details(farmer_id: str, name: str, age: int | None,
                          gender: str | None, preferred_language: str) -> dict:
    """
    Step 2: Save basic farmer details to PostgreSQL.
    Also updates Firestore session to current_step=2.
    """
    updated = farmer_repo.update_farmer_details(farmer_id, name, age, gender, preferred_language)
    write_session(farmer_id, current_step=2, partial_data={
        "name": name,
        "preferred_language": preferred_language,
    })
    logger.info("[Farmer] Basic details saved for farmer %s", farmer_id)
    return updated


def save_location(farmer_id: str, pin_code: str, village_name: str,
                  full_address: str | None) -> dict:
    """
    Step 3: Resolve PIN code via India Post API, then save farmer_locations row.
    Also updates Firestore session to current_step=3.
    Falls back to empty strings if PIN API is unreachable.
    """
    try:
        geo = resolve_pincode(pin_code)
        state = geo["state"]
        district = geo["district"]
        taluka = geo["taluka"]
    except Exception as exc:
        logger.warning("[Farmer] PIN resolve failed (%s) — saving with empty geo fields", exc)
        state = district = taluka = ""

    location = farmer_repo.create_farmer_location(
        farmer_id, pin_code, state, district, taluka, village_name, full_address
    )
    write_session(farmer_id, current_step=3, partial_data={
        "pin_code": pin_code,
        "village_name": village_name,
        "state": state,
        "district": district,
    })
    logger.info("[Farmer] Location saved for farmer %s (PIN=%s)", farmer_id, pin_code)
    return location
