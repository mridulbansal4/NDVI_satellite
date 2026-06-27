"""
repositories/crop.py — SQL operations for the crops table.
"""

import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def create_crop(farm_id: str, crop_name: str, crop_variety: str | None,
                sowing_date: str, season: str,
                expected_harvest_month: str | None) -> dict:
    """Inserts a crop record for a farm (Step 6)."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO crops
                  (farm_id, crop_name, crop_variety, sowing_date,
                   season, expected_harvest_month)
                VALUES (%s, %s, %s, %s, %s, %s)
                RETURNING id, farm_id, crop_name, crop_variety,
                          sowing_date, season, expected_harvest_month, created_at
                """,
                (farm_id, crop_name, crop_variety, sowing_date,
                 season, expected_harvest_month)
            )
            return dict(cur.fetchone())


def get_crops_by_farm(farm_id: str) -> list[dict]:
    """Returns all crop records for a farm."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                SELECT id, crop_name, crop_variety, sowing_date, season,
                       expected_harvest_month, created_at
                FROM crops WHERE farm_id = %s ORDER BY sowing_date DESC
                """,
                (farm_id,)
            )
            return [dict(r) for r in cur.fetchall()]
