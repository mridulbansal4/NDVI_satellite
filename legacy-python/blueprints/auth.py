"""
blueprints/auth.py — Auth routes.
POST /auth/signup  → { token, is_new_user, farmer_id }
POST /auth/login   → { token, is_new_user, farmer_id }
"""

from flask import Blueprint, request, jsonify
from marshmallow import Schema, fields, ValidationError, validate
from services import auth as auth_service

bp = Blueprint("auth", __name__, url_prefix="/auth")


# ─── Schemas ────────────────────────────────────────────────────────────────────

class SignupSchema(Schema):
    mobile_number = fields.Str(
        required=True,
        validate=validate.Regexp(r"^\d{10}$", error="Mobile number must be exactly 10 digits.")
    )
    password = fields.Str(
        required=True,
        validate=validate.Length(min=6, error="Password must be at least 6 characters.")
    )
    name = fields.Str(load_default=None)


class LoginSchema(Schema):
    mobile_number = fields.Str(
        required=True,
        validate=validate.Regexp(r"^\d{10}$", error="Mobile number must be exactly 10 digits.")
    )
    password = fields.Str(
        required=True,
        validate=validate.Length(min=1, error="Password is required.")
    )


# ─── Routes ─────────────────────────────────────────────────────────────────────

@bp.route("/signup", methods=["POST"])
def signup():
    """POST /auth/signup"""
    body = request.get_json(silent=True) or {}
    try:
        data = SignupSchema().load(body)
    except ValidationError as ve:
        return jsonify({"errors": ve.messages}), 422

    try:
        result = auth_service.signup(
            mobile_number=data["mobile_number"],
            password=data["password"],
            name=data.get("name"),
        )
        return jsonify(result), 201
    except ValueError as ve:
        return jsonify({"error": str(ve)}), 409
    except Exception as exc:
        return jsonify({"error": f"Signup failed: {exc}"}), 500


@bp.route("/login", methods=["POST"])
def login():
    """POST /auth/login"""
    body = request.get_json(silent=True) or {}
    try:
        data = LoginSchema().load(body)
    except ValidationError as ve:
        return jsonify({"errors": ve.messages}), 422

    try:
        result = auth_service.login(
            mobile_number=data["mobile_number"],
            password=data["password"],
        )
        return jsonify(result), 200
    except ValueError as ve:
        return jsonify({"error": str(ve)}), 401
    except Exception as exc:
        return jsonify({"error": f"Login failed: {exc}"}), 500
