# 21 — Authentication UI Subsystem

## Overview
Authentication UI controls reside in `frontend/src/components/PremiumAuthFlow.jsx`, `frontend/src/components/AuthModal.jsx`, and `frontend/src/pages/Login.jsx`.

## User Authentication Flow
1. User enters 10-digit Indian mobile number in `Login.jsx` or `PremiumAuthFlow.jsx`.
2. Triggers `POST /api/auth/send-otp`.
3. Modal (`AuthModal.jsx`) opens prompting user to enter 6-digit SMS verification code.
4. Triggers `POST /api/auth/verify-otp`.
5. Upon verification, Firebase issues client ID token which is posted to `/api/auth/verify-token`.

## Related Documents
- [10_FIREBASE_AUTH.md](../architecture/10_FIREBASE_AUTH.md)
- [22_COMPONENTS.md](./22_COMPONENTS.md)
