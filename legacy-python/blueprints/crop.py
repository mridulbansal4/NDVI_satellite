"""
blueprints/crop.py — POST /crop (Step 6)
"""

from flask import Blueprint, request, jsonify, g
from marshmallow import Schema, fields, ValidationError, validate
from middlewares.auth import jwt_required_middleware
from services import crop as crop_service

bp = Blueprint("crop", __name__, url_prefix="/crop")

VALID_SEASONS = ["kharif", "rabi", "zaid"]


class CropSchema(Schema):
    farm_id = fields.UUID(required=True)
    crop_name = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    crop_variety = fields.Str(load_default=None)
    sowing_date = fields.Date(required=True, format="%Y-%m-%d")
    season = fields.Str(required=True, validate=validate.OneOf(VALID_SEASONS))
    expected_harvest_month = fields.Str(load_default=None)


@bp.route("", methods=["POST"])
@jwt_required_middleware
def add_crop():
    """
    POST /crop (Step 6)
    Body: { farm_id, crop_name, crop_variety?, sowing_date, season, expected_harvest_month? }
    """
    schema = CropSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    try:
        result = crop_service.add_crop(
            g.farmer_id,
            str(data["farm_id"]),
            data["crop_name"],
            data.get("crop_variety"),
            data["sowing_date"].isoformat(),
            data["season"],
            data.get("expected_harvest_month"),
        )
    except ValueError as err:
        return jsonify({"error": str(err)}), 403

    return jsonify(result), 201
