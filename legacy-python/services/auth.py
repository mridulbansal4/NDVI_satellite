"""
services/auth.py — Password-based authentication (no OTP).
Uses werkzeug.security for bcrypt hashing (ships with Flask, no extra dep).
Two flows:
  signup(mobile, password, name) → create farmer → JWT
  login(mobile, password) → verify hash → JWT
"""

import logging
from werkzeug.security import generate_password_hash, check_password_hash
from flask_jwt_extended import create_access_token
from repositories import farmer as farmer_repo
from firestore.session import write_session

logger = logging.getLogger(__name__)


def signup(mobile_number: str, password: str, name: str = None) -> dict:
    """
    POST /auth/signup
    1. Ensure mobile is not already registered.
    2. Hash password with werkzeug pbkdf2:sha256.
    3. Create farmer record.
    4. Write Firestore onboarding session (step=1).
    5. Return JWT.
    """
    existing = farmer_repo.find_farmer_by_mobile(mobile_number)
    if existing:
        raise ValueError("Mobile number already registered. Please log in.")

    pw_hash = generate_password_hash(password)
    new_farmer = farmer_repo.create_farmer(mobile_number, pw_hash, name)
    farmer_id = str(new_farmer["id"])
    token = create_access_token(identity=farmer_id)

    # Start Firestore onboarding session for resume on reconnect
    try:
        write_session(farmer_id, current_step=1, partial_data={"mobile_number": mobile_number})
    except Exception as e:
        logger.warning("[Firestore] Session write failed (non-fatal): %s", e)

    logger.info("[Auth] Signup complete for farmer: %s", farmer_id)
    return {
        "token": token,
        "is_new_user": True,
        "farmer_id": farmer_id,
    }


def login(mobile_number: str, password: str) -> dict:
    """
    POST /auth/login
    1. Find farmer by mobile.
    2. Verify password hash.
    3. Return JWT.
    """
    farmer = farmer_repo.find_farmer_by_mobile(mobile_number)

    if not farmer:
        raise ValueError("No account found with this mobile number. Please sign up.")

    if not farmer.get("password_hash"):
        raise ValueError("Account has no password set. Please sign up again.")

    if not check_password_hash(farmer["password_hash"], password):
        raise ValueError("Incorrect password. Please try again.")

    farmer_id = str(farmer["id"])
    token = create_access_token(identity=farmer_id)

    # Determine if onboarding is complete (name is set)
    is_new_user = not bool(farmer.get("name"))

    logger.info("[Auth] Login successful for farmer: %s", farmer_id)
    return {
        "token": token,
        "is_new_user": is_new_user,
        "farmer_id": farmer_id,
    }
