"""
blueprints/dashboard.py — GET /dashboard (requires JWT)
"""

from flask import Blueprint, jsonify, g
from middlewares.auth import jwt_required_middleware
from services import dashboard as dashboard_service

bp = Blueprint("dashboard", __name__, url_prefix="/dashboard")


@bp.route("", methods=["GET"])
@jwt_required_middleware
def get_dashboard():
    """
    GET /dashboard
    Auth: Bearer JWT required.
    Returns assembled farmer + farms + crops + latest VI report per farm.
    """
    try:
        data = dashboard_service.get_dashboard(g.farmer_id)
    except ValueError as err:
        return jsonify({"error": str(err)}), 404

    return jsonify(data), 200
