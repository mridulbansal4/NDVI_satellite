"""
repositories/soil.py — SQL operations for the soil_info table.
"""

import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def upsert_soil_info(farm_id: str, soil_type: str) -> dict:
    """
    Inserts or updates soil_info for a farm (Step 8 — optional).
    Uses ON CONFLICT on the UNIQUE farm_id constraint.
    """
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO soil_info (farm_id, soil_type)
                VALUES (%s, %s)
                ON CONFLICT (farm_id) DO UPDATE SET soil_type = EXCLUDED.soil_type
                RETURNING id, farm_id, soil_type
                """,
                (farm_id, soil_type)
            )
            return dict(cur.fetchone())


def get_soil_by_farm(farm_id: str) -> dict | None:
    """Returns soil_info for a farm or None if skipped."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "SELECT id, soil_type FROM soil_info WHERE farm_id = %s",
                (farm_id,)
            )
            row = cur.fetchone()
            return dict(row) if row else None
