"""
services/crop.py — Business logic for crop info (Step 6).
"""

import logging
from repositories import crop as crop_repo
from repositories import farm as farm_repo
from firestore.session import write_session

logger = logging.getLogger(__name__)


def add_crop(farmer_id: str, farm_id: str, crop_name: str,
             crop_variety: str | None, sowing_date: str,
             season: str, expected_harvest_month: str | None) -> dict:
    """
    Step 6: Validates farm belongs to farmer, then inserts crop record.
    Updates Firestore session to current_step=6.
    """
    farm = farm_repo.get_farm_by_id(farm_id)
    if not farm or str(farm["farmer_id"]) != farmer_id:
        raise ValueError(f"Farm {farm_id} not found or does not belong to this farmer.")

    crop = crop_repo.create_crop(
        farm_id, crop_name, crop_variety, sowing_date,
        season, expected_harvest_month
    )
    write_session(farmer_id, current_step=6, partial_data={"crop_name": crop_name})
    logger.info("[Crop] '%s' added to farm %s", crop_name, farm_id)
    return crop
