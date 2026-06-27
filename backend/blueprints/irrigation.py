"""
blueprints/irrigation.py — POST /irrigation (Step 7)
"""

from flask import Blueprint, request, jsonify, g
from marshmallow import Schema, fields, ValidationError, validate
from middlewares.auth import jwt_required_middleware
from services import irrigation as irrigation_service

bp = Blueprint("irrigation", __name__, url_prefix="/irrigation")

VALID_IRRIGATION_TYPES = ["rainfed", "borewell", "canal", "drip_irrigation", "sprinkler"]


class IrrigationSchema(Schema):
    farm_id = fields.UUID(required=True)
    irrigation_type = fields.Str(
        required=True, validate=validate.OneOf(VALID_IRRIGATION_TYPES)
    )
    water_source = fields.Str(load_default=None)


@bp.route("", methods=["POST"])
@jwt_required_middleware
def add_irrigation():
    """
    POST /irrigation (Step 7)
    Body: { farm_id, irrigation_type, water_source? }
    """
    schema = IrrigationSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    try:
        result = irrigation_service.add_irrigation(
            g.farmer_id,
            str(data["farm_id"]),
            data["irrigation_type"],
            data.get("water_source"),
        )
    except ValueError as err:
        return jsonify({"error": str(err)}), 403

    return jsonify(result), 201
