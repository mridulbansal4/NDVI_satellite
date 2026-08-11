"""
blueprints/soil.py — POST /soil (Step 8 — optional)
"""

from flask import Blueprint, request, jsonify, g
from marshmallow import Schema, fields, ValidationError, validate
from middlewares.auth import jwt_required_middleware
from services import soil as soil_service

bp = Blueprint("soil", __name__, url_prefix="/soil")

VALID_SOIL_TYPES = ["black", "red", "sandy", "mixed", "unknown"]


class SoilSchema(Schema):
    farm_id = fields.UUID(required=True)
    soil_type = fields.Str(required=True, validate=validate.OneOf(VALID_SOIL_TYPES))


@bp.route("", methods=["POST"])
@jwt_required_middleware
def add_soil():
    """
    POST /soil (Step 8 — optional, farmer may skip this step)
    Body: { farm_id, soil_type }
    """
    schema = SoilSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    try:
        result = soil_service.add_soil_info(
            g.farmer_id,
            str(data["farm_id"]),
            data["soil_type"],
        )
    except ValueError as err:
        return jsonify({"error": str(err)}), 403

    return jsonify(result), 201
