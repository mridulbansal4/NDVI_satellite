"""
repositories/consent.py — SQL operations for the consents table.
"""

import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def create_consent(farmer_id: str, satellite_monitoring: bool) -> dict:
    """
    Inserts a consent record for a farmer.
    Uses ON CONFLICT to handle idempotent submissions.
    """
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO consents (farmer_id, satellite_monitoring)
                VALUES (%s, %s)
                ON CONFLICT (farmer_id) DO UPDATE
                  SET satellite_monitoring = EXCLUDED.satellite_monitoring,
                      consented_at = NOW()
                RETURNING id, farmer_id, satellite_monitoring, consented_at
                """,
                (farmer_id, satellite_monitoring)
            )
            return dict(cur.fetchone())


def get_consent_by_farmer(farmer_id: str) -> dict | None:
    """Returns consent record for a farmer."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "SELECT id, satellite_monitoring, consented_at FROM consents WHERE farmer_id = %s",
                (farmer_id,)
            )
            row = cur.fetchone()
            return dict(row) if row else None
