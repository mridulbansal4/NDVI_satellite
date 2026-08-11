// onboarding_handler.go — E8-E24.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §10.8-§10.11.
//
// In this phase each handler runs its real validation — which is what the
// Phase 1 exit gate checks — and then answers 501 where the service layer does
// not exist yet. Validation runs BEFORE any service call in the Python code
// too, so the ordering is already correct and Phase 5/6 only replaces the tail.

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ── E8 — POST /api/auth/verify-token (§8.4) ─────────────────────────────────

func (s *Server) verifyToken(c *gin.Context) {
	body := readBody(c)
	if _, ok := body["idToken"]; body == nil || !ok {
		Err(c, http.StatusBadRequest, "Missing 'idToken' field")
		return
	}
	s.notImplemented(c, "Firebase ID token verification (Phase 5)")
}

// ── E9 — POST /api/auth/send-otp (§8.5) ─────────────────────────────────────

func (s *Server) sendOTP(c *gin.Context) {
	phone := normalisePhoneInput(c)
	if (len(phone) != 10 && len(phone) != 12) || !allDigits(phone) {
		Err(c, http.StatusBadRequest, "Provide a valid 10-digit Indian mobile number.")
		return
	}
	s.notImplemented(c, "SMS OTP delivery (Phase 5)")
}

// ── E10 — POST /api/auth/verify-otp ─────────────────────────────────────────

func (s *Server) verifyOTP(c *gin.Context) {
	body := readBody(c)
	phone := normalisePhone(jsonString(body, "phone"))
	otp := strings.TrimSpace(jsonString(body, "otp"))
	if phone == "" || otp == "" {
		Err(c, http.StatusBadRequest, "phone and otp are required.")
		return
	}
	// No OTP store yet, so nothing can ever match. That is the same answer a
	// cold Python process gives, and it is the branch the corpus pins.
	Err(c, http.StatusUnauthorized, "Incorrect or expired OTP.")
}

// ── E11/E12 — signup / login. Note 422, not 400 (§6.2). ─────────────────────

func (s *Server) signup(c *gin.Context) {
	var req SignupRequest
	if !BindAndValidate(c, &req, http.StatusUnprocessableEntity) {
		return
	}
	s.notImplemented(c, "signup (Phase 5)")
}

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if !BindAndValidate(c, &req, http.StatusUnprocessableEntity) {
		return
	}
	s.notImplemented(c, "login (Phase 5)")
}

// ── E13-E15 — farmer ────────────────────────────────────────────────────────

func (s *Server) basicDetails(c *gin.Context) {
	var req BasicDetailsRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "farmer basic details (Phase 5)")
}

func (s *Server) location(c *gin.Context) {
	var req LocationRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "farmer location (Phase 5)")
}

func (s *Server) pincode(c *gin.Context) {
	s.notImplemented(c, "India Post PIN lookup (Phase 5)")
}

// ── E16-E20 — farm / crop / irrigation / soil / consent ─────────────────────

func (s *Server) createFarm(c *gin.Context) {
	var req FarmRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "farm creation (Phase 5)")
}

func (s *Server) addCrop(c *gin.Context) {
	var req CropRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "crop (Phase 5)")
}

func (s *Server) addIrrigation(c *gin.Context) {
	var req IrrigationRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "irrigation (Phase 5)")
}

func (s *Server) addSoil(c *gin.Context) {
	var req SoilRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "soil (Phase 5)")
}

func (s *Server) submitConsent(c *gin.Context) {
	var req ConsentRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	s.notImplemented(c, "consent (Phase 5)")
}

// ── E21 — dashboard ─────────────────────────────────────────────────────────

func (s *Server) dashboard(c *gin.Context) {
	s.notImplemented(c, "dashboard (Phase 5)")
}

// ── E22-E24 — chatbot (§10.11) ──────────────────────────────────────────────

func (s *Server) chat(c *gin.Context) {
	body := readBody(c)
	message := strings.TrimSpace(jsonString(body, "message"))
	if message == "" {
		Err(c, http.StatusBadRequest, "message field is required and must not be empty.")
		return
	}
	s.notImplemented(c, "chatbot (Phase 6)")
}

func (s *Server) chatReset(c *gin.Context) {
	body := readBody(c)
	sessionID := strings.TrimSpace(jsonString(body, "session_id"))
	if sessionID == "" {
		Err(c, http.StatusBadRequest, "session_id is required.")
		return
	}
	// Clearing a session that does not exist is a no-op in Python too, so this
	// is already the final behaviour rather than a stub.
	c.JSON(http.StatusOK, gin.H{"ok": true, "session_id": sessionID})
}

func (s *Server) chatHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"model":    s.deps.Cfg.OllamaModel,
		"base_url": s.deps.Cfg.OllamaBaseURL,
	})
}

// ── helpers ─────────────────────────────────────────────────────────────────

// jsonString mirrors Python's str(body.get(key, "")): a missing key, a null,
// or a non-string value all collapse to "".
func jsonString(body map[string]json.RawMessage, key string) string {
	raw, ok := body[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// normalisePhoneInput reproduces app.py's
// str(body.get("phone", "")).strip().replace(" ", "").
func normalisePhoneInput(c *gin.Context) string {
	return normalisePhone(jsonString(readBody(c), "phone"))
}

func normalisePhone(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "")
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
