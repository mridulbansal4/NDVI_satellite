"""
blueprints/consent.py — POST /consent (Step 9 — final onboarding step)
"""

from flask import Blueprint, request, jsonify, g
from marshmallow import Schema, fields, ValidationError
from middlewares.auth import jwt_required_middleware
from services import consent as consent_service

bp = Blueprint("consent", __name__, url_prefix="/consent")


class ConsentSchema(Schema):
    satellite_monitoring = fields.Bool(required=True)


@bp.route("", methods=["POST"])
@jwt_required_middleware
def submit_consent():
    """
    POST /consent (Step 9 — final onboarding step)
    Body: { "satellite_monitoring": true }
    Logic:
      1. Save consent to PostgreSQL
      2. Delete Firestore onboarding session
      3. Mock-trigger VI Engine (logs farm_id)
    """
    schema = ConsentSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    result = consent_service.submit_consent(
        g.farmer_id,
        data["satellite_monitoring"],
    )
    return jsonify(result), 200
