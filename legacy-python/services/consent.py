"""
services/consent.py — Business logic for the consent + onboarding completion step (Step 9).
"""

import logging
from repositories import consent as consent_repo
from repositories import farm as farm_repo
from firestore.session import delete_session, write_farm_alert

logger = logging.getLogger(__name__)


def submit_consent(farmer_id: str, satellite_monitoring: bool) -> dict:
    """
    Step 9 (final):
      1. Save consent record to PostgreSQL.
      2. Delete Firestore onboarding session (farmer is now committed to DB).
      3. Mock-trigger VI Engine for all farms belonging to this farmer.
    Returns success confirmation.
    """
    # Save consent
    consent = consent_repo.create_consent(farmer_id, satellite_monitoring)

    # Delete Firestore session — onboarding complete
    delete_session(farmer_id)

    # Get all farms and mock-trigger VI Engine
    farms = farm_repo.get_farms_by_farmer(farmer_id)
    for farm in farms:
        farm_id = str(farm["id"])
        logger.info("[VI Engine] VI Engine triggered for farm_id: %s", farm_id)
        # Write a placeholder farm alert to Firestore (will be updated by real VI engine)
        write_farm_alert(farm_id, {
            "cvi_mean": 0.0,
            "ndvi": 0.0,
            "ndmi": 0.0,
        })

    logger.info("[Consent] Onboarding complete for farmer %s", farmer_id)
    return {
        "success": True,
        "message": "Onboarding complete",
        "consent_id": str(consent["id"]),
    }
