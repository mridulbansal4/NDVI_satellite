"""
repositories/irrigation.py — SQL operations for the irrigation table.
"""

import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def create_irrigation(farm_id: str, irrigation_type: str,
                       water_source: str | None) -> dict:
    """Inserts an irrigation record for a farm (Step 7)."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO irrigation (farm_id, irrigation_type, water_source)
                VALUES (%s, %s, %s)
                RETURNING id, farm_id, irrigation_type, water_source
                """,
                (farm_id, irrigation_type, water_source)
            )
            return dict(cur.fetchone())


def get_irrigation_by_farm(farm_id: str) -> dict | None:
    """Returns irrigation record for a farm."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "SELECT id, irrigation_type, water_source FROM irrigation WHERE farm_id = %s",
                (farm_id,)
            )
            row = cur.fetchone()
            return dict(row) if row else None
