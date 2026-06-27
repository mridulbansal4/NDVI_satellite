"""
services/dashboard.py — Business logic for the farmer dashboard.
Assembles farmer profile + farms + crops + latest VI report.
"""

import logging
from repositories import farmer as farmer_repo
from repositories import farm as farm_repo
from repositories import crop as crop_repo
from repositories import vi_report as vi_repo

logger = logging.getLogger(__name__)


def get_dashboard(farmer_id: str) -> dict:
    """
    GET /dashboard logic:
      1. Fetch farmer profile from PostgreSQL.
      2. Fetch all farms for this farmer.
      3. For each farm, fetch crops and the latest VI report.
      4. Assemble and return the combined dashboard payload.
    """
    farmer = farmer_repo.find_farmer_by_id(farmer_id)
    if not farmer:
        raise ValueError(f"Farmer {farmer_id} not found.")

    farms = farm_repo.get_farms_by_farmer(farmer_id)
    farm_ids = [str(f["id"]) for f in farms]

    # Batch fetch latest VI report for all farms in one query
    vi_map = vi_repo.get_latest_vi_report_per_farm(farm_ids)

    assembled_farms = []
    for farm in farms:
        farm_id = str(farm["id"])
        crops = crop_repo.get_crops_by_farm(farm_id)
        latest_vi = vi_map.get(farm_id)

        farm_data = {
            "id": farm_id,
            "farm_name": farm["farm_name"],
            "total_area": float(farm["total_area"]),
            "area_unit": farm["area_unit"],
            "latitude": farm["latitude"],
            "longitude": farm["longitude"],
            "boundary_geom": farm.get("boundary_geom"),
            "crops": [
                {
                    "crop_name": c["crop_name"],
                    "season": c["season"],
                    "sowing_date": str(c["sowing_date"]),
                }
                for c in crops
            ],
            "latest_vi_report": None,
        }

        if latest_vi:
            farm_data["latest_vi_report"] = {
                "cvi_mean": round(float(latest_vi["cvi_mean"]), 4),
                "ndvi": round(float(latest_vi["ndvi"]), 4),
                "confidence_score": round(float(latest_vi["confidence_score"]), 2),
                "period_start": str(latest_vi["period_start"]),
                "period_end": str(latest_vi["period_end"]),
            }

        assembled_farms.append(farm_data)

    logger.info("[Dashboard] Assembled dashboard for farmer %s (%d farms)", farmer_id, len(farms))
    return {
        "farmer": {
            "id": str(farmer["id"]),
            "name": farmer["name"],
            "mobile_number": farmer["mobile_number"],
            "preferred_language": farmer["preferred_language"],
        },
        "farms": assembled_farms,
    }
