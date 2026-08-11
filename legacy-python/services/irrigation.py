"""
services/irrigation.py — Business logic for irrigation step (Step 7).
"""

import logging
from repositories import irrigation as irrigation_repo
from repositories import farm as farm_repo
from firestore.session import write_session

logger = logging.getLogger(__name__)


def add_irrigation(farmer_id: str, farm_id: str, irrigation_type: str,
                   water_source: str | None) -> dict:
    """
    Step 7: Validates farm ownership and inserts irrigation record.
    Updates Firestore session to current_step=7.
    """
    farm = farm_repo.get_farm_by_id(farm_id)
    if not farm or str(farm["farmer_id"]) != farmer_id:
        raise ValueError(f"Farm {farm_id} not found or does not belong to this farmer.")

    irrigation = irrigation_repo.create_irrigation(farm_id, irrigation_type, water_source)
    write_session(farmer_id, current_step=7, partial_data={"irrigation_type": irrigation_type})
    logger.info("[Irrigation] %s added to farm %s", irrigation_type, farm_id)
    return irrigation
