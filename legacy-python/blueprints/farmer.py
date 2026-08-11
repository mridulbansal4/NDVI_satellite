"""
blueprints/farmer.py — POST /farmer/basic-details (Step 2) + POST /farmer/location (Step 3)
"""

from flask import Blueprint, request, jsonify, g
from marshmallow import Schema, fields, ValidationError, validate
from middlewares.auth import jwt_required_middleware
from services import farmer as farmer_service

bp = Blueprint("farmer", __name__, url_prefix="/farmer")

VALID_GENDERS = ["male", "female", "other"]
VALID_LANGUAGES = ["english", "hindi", "marathi", "others"]


class BasicDetailsSchema(Schema):
    name = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    age = fields.Int(load_default=None, validate=validate.Range(min=1, max=120))
    gender = fields.Str(load_default=None, validate=validate.OneOf(VALID_GENDERS))
    preferred_language = fields.Str(
        required=True, validate=validate.OneOf(VALID_LANGUAGES)
    )


class LocationSchema(Schema):
    pin_code = fields.Str(
        required=True,
        validate=validate.Regexp(r"^\d{6}$", error="pin_code must be exactly 6 digits.")
    )
    village_name = fields.Str(required=True, validate=validate.Length(min=1, max=255))
    full_address = fields.Str(load_default=None)


@bp.route("/basic-details", methods=["POST"])
@jwt_required_middleware
def basic_details():
    """
    POST /farmer/basic-details (Step 2)
    Body: { name, age?, gender?, preferred_language }
    """
    schema = BasicDetailsSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    result = farmer_service.update_basic_details(
        g.farmer_id,
        data["name"],
        data.get("age"),
        data.get("gender"),
        data["preferred_language"],
    )
    return jsonify(result), 200


@bp.route("/location", methods=["POST"])
@jwt_required_middleware
def location():
    """
    POST /farmer/location (Step 3)
    Body: { pin_code, village_name, full_address? }
    Auto-resolves state/district/taluka via India Post PIN API.
    """
    schema = LocationSchema()
    try:
        data = schema.load(request.get_json(force=True) or {})
    except ValidationError as err:
        return jsonify({"errors": err.messages}), 400

    result = farmer_service.save_location(
        g.farmer_id,
        data["pin_code"],
        data["village_name"],
        data.get("full_address"),
    )
    return jsonify(result), 201


@bp.route("/pincode/<pin_code>", methods=["GET"])
def resolve_pincode_route(pin_code):
    """
    GET /farmer/pincode/<pin_code>
    Proxies the India Post API call through the backend to avoid frontend CORS issues.
    """
    from utils.pincode import resolve_pincode
    try:
        result = resolve_pincode(pin_code)
        return jsonify(result), 200
    except ValueError as ve:
        return jsonify({"error": str(ve)}), 404
    except Exception as exc:
        return jsonify({"error": "Failed to resolve PIN code"}), 500
