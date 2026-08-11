"""
services/farm.py — Business logic for farm creation (Steps 4 + 5).
"""

import logging
from repositories import farm as farm_repo
from firestore.session import write_session

logger = logging.getLogger(__name__)


def create_farm(farmer_id: str, farm_name: str, total_area: float, area_unit: str,
                land_ownership: str, latitude: float, longitude: float,
                boundary_geojson: dict | None, location_photo_url: str | None) -> dict:
    """
    Steps 4+5: Saves farm record to PostgreSQL.
    GeoJSON polygon boundary is converted to PostGIS GEOMETRY via ST_GeomFromGeoJSON.
    Updates Firestore session to current_step=5.
    """
    farm = farm_repo.create_farm(
        farmer_id, farm_name, total_area, area_unit, land_ownership,
        latitude, longitude, boundary_geojson, location_photo_url
    )
    write_session(farmer_id, current_step=5, partial_data={
        "farm_id": str(farm["id"]),
        "farm_name": farm_name,
        "latitude": latitude,
        "longitude": longitude,
    })
    logger.info("[Farm] Farm '%s' created for farmer %s (farm_id=%s)",
                farm_name, farmer_id, farm["id"])
    return farm
