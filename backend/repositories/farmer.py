"""
repositories/farmer.py — SQL operations for the farmers and farmer_locations tables.
Password-based auth: stores bcrypt hash in password_hash column.
"""

import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def find_farmer_by_mobile(mobile_number: str) -> dict | None:
    """Returns farmer row including password_hash, or None."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "SELECT id, mobile_number, password_hash, name, preferred_language "
                "FROM farmers WHERE mobile_number = %s",
                (mobile_number,)
            )
            row = cur.fetchone()
            return dict(row) if row else None


def find_farmer_by_id(farmer_id: str) -> dict | None:
    """Returns farmer row or None."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "SELECT id, mobile_number, name, age, gender, preferred_language "
                "FROM farmers WHERE id = %s",
                (farmer_id,)
            )
            row = cur.fetchone()
            return dict(row) if row else None


def create_farmer(mobile_number: str, password_hash: str, name: str = None) -> dict:
    """
    Inserts a new farmer with a hashed password.
    name is optional at signup (can be filled in Step 1 of onboarding).
    Returns the new farmer row.
    """
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO farmers (mobile_number, password_hash, name)
                VALUES (%s, %s, %s)
                RETURNING id, mobile_number, name, preferred_language, created_at
                """,
                (mobile_number, password_hash, name)
            )
            return dict(cur.fetchone())


def update_farmer_details(farmer_id: str, name: str, age: int | None,
                           gender: str | None, preferred_language: str) -> dict:
    """Updates farmer record with basic profile details (Step 1 of onboarding)."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                UPDATE farmers
                SET name = %s, age = %s, gender = %s,
                    preferred_language = %s, updated_at = NOW()
                WHERE id = %s
                RETURNING id, name, age, gender, preferred_language
                """,
                (name, age, gender, preferred_language, farmer_id)
            )
            row = cur.fetchone()
            if not row:
                raise ValueError(f"Farmer {farmer_id} not found.")
            return dict(row)


def create_farmer_location(farmer_id: str, pin_code: str, state: str,
                            district: str, taluka: str,
                            village_name: str, full_address: str | None) -> dict:
    """Inserts farmer_locations row (Step 2 of onboarding)."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO farmer_locations
                  (farmer_id, pin_code, state, district, taluka, village_name, full_address)
                VALUES (%s, %s, %s, %s, %s, %s, %s)
                RETURNING id, farmer_id, pin_code, state, district, taluka, village_name
                """,
                (farmer_id, pin_code, state, district, taluka, village_name, full_address)
            )
            return dict(cur.fetchone())
