"""
services/soil.py — Business logic for the optional soil step (Step 8).
"""

import logging
from repositories import soil as soil_repo
from repositories import farm as farm_repo
from firestore.session import write_session

logger = logging.getLogger(__name__)


def add_soil_info(farmer_id: str, farm_id: str, soil_type: str) -> dict:
    """
    Step 8 (optional): Upserts soil_info for the farm.
    Updates Firestore session to current_step=8.
    """
    farm = farm_repo.get_farm_by_id(farm_id)
    if not farm or str(farm["farmer_id"]) != farmer_id:
        raise ValueError(f"Farm {farm_id} not found or does not belong to this farmer.")

    soil = soil_repo.upsert_soil_info(farm_id, soil_type)
    write_session(farmer_id, current_step=8, partial_data={"soil_type": soil_type})
    logger.info("[Soil] %s soil type recorded for farm %s", soil_type, farm_id)
    return soil
