"""
blueprints/farm.py — POST /farm (Steps 4 + 5)
"""

from flask import Blueprint, request, jsonify, g
from marshmallow import Schema, fields, ValidationError, validate
from middlewares.auth import jwt_required_middleware
from services import farm as farm_service

bp = Blueprint("farm", __name__, url_prefix="/farm")

VALID_AREA_UNITS = ["acres", "hectares"]
VALID_OWNERSHIP = ["own_land", "leased_land", "contract_farming"]


class FarmSchema(Schema):
    farm_name = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    total_area = fields.Float(required=True, validate=validate.Range(min=0.01))
    area_unit = fields.Str(required=True, validate=validate.OneOf(VALID_AREA_UNITS))
    land_ownership = fields.Str(required=True, validate=validate.OneOf(VALID_OWNERSHIP))
    latitude = fields.Float(required=True, validate=validate.Range(min=-90, max=90))
    longitude = fields.Float(required=True, validate=validate.Range(min=-180, max=180))
    boundary_geom = fields.Dict(load_default=None)   # GeoJSON Polygon object
    location_photo_url = fields.Str(load_default=None)


@bp.route("", methods=["POST"])
@jwt_required_middleware
def create_farm():
    """
    POST /farm (Steps 4 + 5)
    Body: { farm_name, total_area, area_unit, land_ownership,
            latitude, longitude, boundary_geom?, location_photo_url? }
    """
    schema = FarmSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    result = farm_service.create_farm(
        g.farmer_id,
        data["farm_name"],
        data["total_area"],
        data["area_unit"],
        data["land_ownership"],
        data["latitude"],
        data["longitude"],
        data.get("boundary_geom"),
        data.get("location_photo_url"),
    )
    return jsonify(result), 201
