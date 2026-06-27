"""
db/pool.py — psycopg2 connection pool for agri_platform PostgreSQL database.
Uses ThreadedConnectionPool for multi-threaded Flask requests.
"""

import os
import logging
from psycopg2 import pool as pg_pool
from psycopg2.extras import RealDictCursor

logger = logging.getLogger(__name__)

_pool: pg_pool.ThreadedConnectionPool | None = None


def init_pool() -> None:
    """Initialize the connection pool. Called once at app startup."""
    global _pool
    database_url = os.getenv("DATABASE_URL")
    if not database_url:
        raise RuntimeError("DATABASE_URL environment variable is not set.")
    _pool = pg_pool.ThreadedConnectionPool(
        minconn=2,
        maxconn=20,
        dsn=database_url
    )
    logger.info("[DB] Connection pool initialized (min=2, max=20).")


def get_connection():
    """Borrow a connection from the pool."""
    if _pool is None:
        raise RuntimeError("DB pool not initialized. Call init_pool() first.")
    return _pool.getconn()


def release_connection(conn) -> None:
    """Return a connection to the pool."""
    if _pool and conn:
        _pool.putconn(conn)


class DBConnection:
    """Context manager for safe connection borrow/release."""

    def __enter__(self):
        self.conn = get_connection()
        self.conn.autocommit = False
        return self.conn

    def __exit__(self, exc_type, exc_val, exc_tb):
        if exc_type:
            self.conn.rollback()
            logger.error("[DB] Transaction rolled back due to: %s", exc_val)
        else:
            self.conn.commit()
        release_connection(self.conn)
        return False  # re-raise exceptions


def execute_query(sql: str, params: tuple = (), fetch: str = "all"):
    """
    Utility for single-statement read queries.
    fetch: "all" | "one" | "none"
    Returns list of dicts (all), single dict (one), or None (none).
    """
    with DBConnection() as conn:
        with conn.cursor(cursor_factory=RealDictCursor) as cur:
            cur.execute(sql, params)
            if fetch == "all":
                return [dict(row) for row in cur.fetchall()]
            elif fetch == "one":
                row = cur.fetchone()
                return dict(row) if row else None
            return None
