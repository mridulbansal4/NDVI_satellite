"""
repositories/farm.py — SQL operations for the farms table.
"""

import json
import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def create_farm(farmer_id: str, farm_name: str, total_area: float, area_unit: str,
                land_ownership: str, latitude: float, longitude: float,
                boundary_geojson: dict | None, location_photo_url: str | None) -> dict:
    """
    Inserts a new farm record.
    boundary_geojson is a GeoJSON Polygon dict; converted via ST_GeomFromGeoJSON().
    """
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            if boundary_geojson:
                geojson_str = json.dumps(boundary_geojson)
                cur.execute(
                    """
                    INSERT INTO farms
                      (farmer_id, farm_name, total_area, area_unit, land_ownership,
                       latitude, longitude, boundary_geom, location_photo_url)
                    VALUES (%s, %s, %s, %s, %s, %s, %s,
                            ST_GeomFromGeoJSON(%s), %s)
                    RETURNING id, farmer_id, farm_name, total_area, area_unit,
                              land_ownership, latitude, longitude, created_at
                    """,
                    (farmer_id, farm_name, total_area, area_unit, land_ownership,
                     latitude, longitude, geojson_str, location_photo_url)
                )
            else:
                cur.execute(
                    """
                    INSERT INTO farms
                      (farmer_id, farm_name, total_area, area_unit, land_ownership,
                       latitude, longitude, location_photo_url)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                    RETURNING id, farmer_id, farm_name, total_area, area_unit,
                              land_ownership, latitude, longitude, created_at
                    """,
                    (farmer_id, farm_name, total_area, area_unit, land_ownership,
                     latitude, longitude, location_photo_url)
                )
            return dict(cur.fetchone())


def get_farms_by_farmer(farmer_id: str) -> list[dict]:
    """Returns all farms for a farmer (used in dashboard)."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                SELECT id, farm_name, total_area, area_unit,
                       land_ownership, latitude, longitude, created_at,
                       ST_AsGeoJSON(boundary_geom)::json AS boundary_geom
                FROM farms
                WHERE farmer_id = %s
                ORDER BY created_at ASC
                """,
                (farmer_id,)
            )
            return [dict(r) for r in cur.fetchall()]


def get_farm_by_id(farm_id: str) -> dict | None:
    """Returns a single farm or None."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                "SELECT id, farmer_id, farm_name, total_area, area_unit, "
                "land_ownership, latitude, longitude, ST_AsGeoJSON(boundary_geom)::json AS boundary_geom FROM farms WHERE id = %s",
                (farm_id,)
            )
            row = cur.fetchone()
            return dict(row) if row else None
