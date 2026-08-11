# 10 — Firebase Authentication & Security

## Architecture Overview

MindstriX implements Firebase Phone Authentication (OTP) for mobile login and token verification.

```
Client (React)                  Flask Backend                     Firebase Auth
     |                               |                                 |
     |---- 1. Enter Phone # -------->|                                 |
     |     POST /api/auth/send-otp   |---- 2. Request OTP ------------>|
     |                               |<--- 3. SMS Triggered -----------|
     |                               |                                 |
     |---- 4. Submit OTP ----------->|                                 |
     |     POST /api/auth/verify-otp |---- 5. Validate OTP ------------|
     |<--- 6. Issue Firebase JWT ----|                                 |
     |                               |                                 |
     |---- 7. Request API Route ----->|                                 |
     |     Headers: Auth (JWT)       |---- 8. verify_jwt_token() ------>|
     |<--- 9. Protected Data --------|                                 |
```

## Backend Verification (`services/auth_service.py`)
- **`init_firebase()`**: Initializes the `firebase_admin` SDK using `FIREBASE_SERVICE_ACCOUNT_PATH`.
- **`verify_jwt_token(id_token)`**: Decodes and verifies the ID token using `auth.verify_id_token()`, returning user `uid` and `phone_number`.

## Related Documents
- [07_API_ARCHITECTURE.md](./07_API_ARCHITECTURE.md)
- [21_AUTHENTICATION_UI.md](../dashboard/21_AUTHENTICATION_UI.md)
