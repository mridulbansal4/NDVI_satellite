"""
repositories/vi_report.py — SQL operations for the vi_reports table (APPEND-ONLY).
Never UPDATE this table — each row is an immutable historical snapshot.
"""

import logging
from psycopg2.extras import RealDictCursor
from db.pool import DBConnection

logger = logging.getLogger(__name__)


def get_latest_vi_report_per_farm(farm_ids: list[str]) -> dict[str, dict]:
    """
    Returns the latest vi_report for each farm_id in farm_ids.
    Uses DISTINCT ON (farm_id) ORDER BY created_at DESC for efficiency.
    Returns a dict keyed by farm_id.
    """
    if not farm_ids:
        return {}

    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                SELECT DISTINCT ON (farm_id)
                    id, farm_id, cvi_mean, cvi_median, cvi_std_dev,
                    ndvi, evi, savi, ndmi, ndwi, gndvi,
                    confidence_score, scenes_used, period_start, period_end, created_at
                FROM vi_reports
                WHERE farm_id = ANY(%s::uuid[])
                ORDER BY farm_id, created_at DESC
                """,
                (farm_ids,)
            )
            rows = cur.fetchall()
            return {str(row["farm_id"]): dict(row) for row in rows}


def get_vi_reports_by_farm(farm_id: str, limit: int = 12) -> list[dict]:
    """Returns recent VI reports for a single farm (for trend charts)."""
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                SELECT id, cvi_mean, ndvi, evi, savi, ndmi, ndwi, gndvi,
                       confidence_score, period_start, period_end, created_at
                FROM vi_reports
                WHERE farm_id = %s
                ORDER BY created_at DESC
                LIMIT %s
                """,
                (farm_id, limit)
            )
            return [dict(r) for r in cur.fetchall()]


def insert_vi_report(farm_id: str, cvi_mean: float, cvi_median: float,
                     cvi_std_dev: float, ndvi: float, evi: float, savi: float,
                     ndmi: float, ndwi: float, gndvi: float,
                     confidence_score: float, scenes_used: int,
                     period_start: str, period_end: str) -> dict:
    """
    Appends a new VI report row. NEVER call UPDATE after this.
    Called by the VI Engine after processing GEE imagery.
    """
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(
                """
                INSERT INTO vi_reports (
                    farm_id, cvi_mean, cvi_median, cvi_std_dev,
                    ndvi, evi, savi, ndmi, ndwi, gndvi,
                    confidence_score, scenes_used, period_start, period_end
                )
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                RETURNING id, farm_id, cvi_mean, ndvi, confidence_score,
                          period_start, period_end, created_at
                """,
                (farm_id, cvi_mean, cvi_median, cvi_std_dev,
                 ndvi, evi, savi, ndmi, ndwi, gndvi,
                 confidence_score, scenes_used, period_start, period_end)
            )
            return dict(cur.fetchone())
