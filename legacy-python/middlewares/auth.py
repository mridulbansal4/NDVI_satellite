"""
middlewares/auth.py — Mock JWT authentication decorator for Flask.
Uses Flask-JWT-Extended to verify Bearer tokens on protected routes.
The token is issued by the /auth/login endpoint with a mock payload.
"""

import logging
from functools import wraps
from flask import g
from flask_jwt_extended import verify_jwt_in_request, get_jwt_identity

logger = logging.getLogger(__name__)


def jwt_required_middleware(fn):
    """
    Decorator that:
      1. Validates the Bearer JWT in the Authorization header.
      2. Extracts farmer_id from the JWT identity.
      3. Stores it in Flask's g object as g.farmer_id for downstream use.

    Usage:
        @bp.route("/protected")
        @jwt_required_middleware
        def protected():
            farmer_id = g.farmer_id
    """
    @wraps(fn)
    def wrapper(*args, **kwargs):
        verify_jwt_in_request()
        farmer_id = get_jwt_identity()
        g.farmer_id = farmer_id
        logger.debug("[Auth] JWT verified for farmer_id=%s", farmer_id)
        return fn(*args, **kwargs)
    return wrapper
