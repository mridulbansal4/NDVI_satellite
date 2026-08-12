// onboarding_handler.go — E8-E24.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §10.8-§10.11.
//
// Validation runs before any service call, exactly as in the Python. The
// 422-vs-400 split between /auth/* and every other onboarding route is an
// inconsistency in the original and is deliberately preserved (§6.2).

package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/SanTiwari07/NDVI_satellite/internal/chatbot"
	"github.com/SanTiwari07/NDVI_satellite/internal/firebase"
	"github.com/SanTiwari07/NDVI_satellite/internal/gemini"
	"github.com/SanTiwari07/NDVI_satellite/internal/httpapi/middleware"
	"github.com/SanTiwari07/NDVI_satellite/internal/ollama"
	"github.com/SanTiwari07/NDVI_satellite/internal/service"
)

// dbUnavailable answers when the database was never configured.
//
// The Python 500s with a raw psycopg2 error in this situation; this is the same
// status with an actionable message, and it only fires in a deployment with no
// DATABASE_URL — where the Python logs "[DB] pool init skipped … (GEE-only mode)"
// at startup and then fails per request anyway.
func (s *Server) dbUnavailable(c *gin.Context) bool {
	if s.deps.Onboarding == nil {
		Err(c, http.StatusInternalServerError,
			"Database is not configured. Check DATABASE_URL and server logs.")
		return true
	}
	return false
}

// ── E8 — POST /api/auth/verify-token (§8.4) ─────────────────────────────────

func (s *Server) verifyToken(c *gin.Context) {
	body := readBody(c)
	raw, present := body["idToken"]
	if body == nil || !present {
		Err(c, http.StatusBadRequest, "Missing 'idToken' field")
		return
	}
	var idToken string
	_ = json.Unmarshal(raw, &idToken)

	user, err := s.deps.Firebase.VerifyIDToken(c.Request.Context(), idToken)
	if err != nil {
		// An invalid token and an unconfigured Firebase both raise ValueError
		// in the Python, which the route turns into 401.
		if errors.Is(err, firebase.ErrInvalidToken) || errors.Is(err, firebase.ErrNotConfigured) {
			Err(c, http.StatusUnauthorized, firebase.ErrInvalidToken.Error())
			return
		}
		s.log.Error("Server error verifying JWT token: " + err.Error())
		Err(c, http.StatusInternalServerError, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token verified successfully",
		"user":    user,
	})
}

// ── E9 — POST /api/auth/send-otp (§8.5) ─────────────────────────────────────

func (s *Server) sendOTP(c *gin.Context) {
	phone := normalisePhone(jsonString(readBody(c), "phone"))
	if (len(phone) != 10 && len(phone) != 12) || !allDigits(phone) {
		Err(c, http.StatusBadRequest, "Provide a valid 10-digit Indian mobile number.")
		return
	}
	if err := s.deps.SMS.SendOTP(c.Request.Context(), phone); err != nil {
		s.log.Warn("send_otp failed: " + err.Error())
		Err(c, http.StatusBadGateway, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "OTP sent."})
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
	if !s.deps.SMS.VerifyOTP(phone, otp) {
		Err(c, http.StatusUnauthorized, "Incorrect or expired OTP.")
		return
	}
	// The echoed number is the last 10 digits with a +91 prefix.
	last10 := phone
	if len(last10) > 10 {
		last10 = last10[len(last10)-10:]
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "phone": "+91" + last10})
}

// ── E11/E12 — signup / login. Note 422, not 400 (§6.2). ─────────────────────

func (s *Server) signup(c *gin.Context) {
	var req SignupRequest
	if !BindAndValidate(c, &req, http.StatusUnprocessableEntity) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.Signup(c.Request.Context(),
		*req.MobileNumber, *req.Password, req.Name)
	switch {
	case errors.Is(err, service.ErrMobileTaken):
		Err(c, http.StatusConflict, err.Error())
	case err != nil:
		Err(c, http.StatusInternalServerError, "Signup failed: "+err.Error())
	default:
		c.JSON(http.StatusCreated, result)
	}
}

func (s *Server) login(c *gin.Context) {
	var req LoginRequest
	if !BindAndValidate(c, &req, http.StatusUnprocessableEntity) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.Login(c.Request.Context(),
		*req.MobileNumber, *req.Password)
	switch {
	case errors.Is(err, service.ErrNoAccount),
		errors.Is(err, service.ErrNoPassword),
		errors.Is(err, service.ErrWrongPassword):
		// All three are 401 but carry distinct messages the frontend branches on.
		Err(c, http.StatusUnauthorized, err.Error())
	case err != nil:
		Err(c, http.StatusInternalServerError, "Login failed: "+err.Error())
	default:
		c.JSON(http.StatusOK, result)
	}
}

// ── E13-E15 — farmer ────────────────────────────────────────────────────────

func (s *Server) basicDetails(c *gin.Context) {
	var req BasicDetailsRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.UpdateBasicDetails(c.Request.Context(),
		middleware.FarmerID(c), *req.Name, req.Age, req.Gender, *req.PreferredLanguage)
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) location(c *gin.Context) {
	var req LocationRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.SaveLocation(c.Request.Context(),
		middleware.FarmerID(c), *req.PinCode, *req.VillageName, req.FullAddress)
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

// pincode proxies the India Post lookup so the frontend avoids a CORS problem.
func (s *Server) pincode(c *gin.Context) {
	result, err := s.deps.PinAPI.Resolve(c.Request.Context(), c.Param("pin_code"))
	if err != nil {
		var pe *service.PinCodeError
		if errors.As(err, &pe) && pe.NotFound {
			Err(c, http.StatusNotFound, pe.Msg)
			return
		}
		Err(c, http.StatusInternalServerError, "Failed to resolve PIN code")
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── E16-E20 — farm / crop / irrigation / soil / consent ─────────────────────

// ownership maps the shared 403 branch of steps 6-8.
func (s *Server) ownership(c *gin.Context, err error, farmID string) bool {
	if errors.Is(err, service.ErrOwnership) {
		Err(c, http.StatusForbidden, service.OwnershipMessage(farmID))
		return true
	}
	return false
}

func (s *Server) createFarm(c *gin.Context) {
	var req FarmRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	var boundary json.RawMessage
	if req.BoundaryGeom != nil {
		boundary, _ = json.Marshal(req.BoundaryGeom)
	}

	result, err := s.deps.Onboarding.CreateFarm(c.Request.Context(),
		middleware.FarmerID(c), *req.FarmName, *req.TotalArea, *req.AreaUnit,
		*req.LandOwnership, *req.Latitude, *req.Longitude, boundary, req.LocationPhotoURL)
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (s *Server) addCrop(c *gin.Context) {
	var req CropRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.AddCrop(c.Request.Context(),
		middleware.FarmerID(c), *req.FarmID, *req.CropName, req.CropVariety,
		*req.SowingDate, *req.Season, req.ExpectedHarvestMonth)
	if s.ownership(c, err, *req.FarmID) {
		return
	}
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (s *Server) addIrrigation(c *gin.Context) {
	var req IrrigationRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.AddIrrigation(c.Request.Context(),
		middleware.FarmerID(c), *req.FarmID, *req.IrrigationType, req.WaterSource)
	if s.ownership(c, err, *req.FarmID) {
		return
	}
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (s *Server) addSoil(c *gin.Context) {
	var req SoilRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.AddSoilInfo(c.Request.Context(),
		middleware.FarmerID(c), *req.FarmID, *req.SoilType)
	if s.ownership(c, err, *req.FarmID) {
		return
	}
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (s *Server) submitConsent(c *gin.Context) {
	var req ConsentRequest
	if !BindAndValidate(c, &req, http.StatusBadRequest) {
		return
	}
	if s.dbUnavailable(c) {
		return
	}

	result, err := s.deps.Onboarding.SubmitConsent(c.Request.Context(),
		middleware.FarmerID(c), *req.SatelliteMonitoring)
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── E21 — dashboard ─────────────────────────────────────────────────────────

func (s *Server) dashboard(c *gin.Context) {
	if s.dbUnavailable(c) {
		return
	}
	result, err := s.deps.Onboarding.GetDashboard(c.Request.Context(), middleware.FarmerID(c))
	if errors.Is(err, service.ErrFarmerNotFound) {
		// The wrapped message carries the id; strip the sentinel prefix so the
		// body is exactly "Farmer <id> not found." (§10.10).
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		Err(c, http.StatusNotFound, msg)
		return
	}
	if err != nil {
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── E22-E24 — chatbot (§10.11) ──────────────────────────────────────────────

func (s *Server) chat(c *gin.Context) {
	body := readBody(c)
	message := strings.TrimSpace(jsonString(body, "message"))
	sessionID := strings.TrimSpace(jsonString(body, "session_id"))
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	if message == "" {
		Err(c, http.StatusBadRequest, "message field is required and must not be empty.")
		return
	}

	// farmData / heatmapData are decoded with UseNumber so that 30 and 30.0
	// render as Python would (see internal/chatbot.pyFormat).
	fd := chatbot.FallbackFarm()
	if raw, present := body["farmData"]; present && string(raw) != "null" {
		_ = decodeWithNumbers(raw, &fd)
	}
	hd := chatbot.FallbackHeatmap()
	if raw, present := body["heatmapData"]; present && string(raw) != "null" {
		_ = decodeWithNumbers(raw, &hd)
	}

	systemPrompt, err := chatbot.BuildSystemPrompt(fd, hd)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "session_id": sessionID})
		return
	}

	// History is fetched BEFORE the new user message is appended (§10.11).
	history := s.deps.Memory.History(sessionID)

	reply, err := s.generateReply(c, systemPrompt, history, message)
	if err != nil {
		s.log.Warn("[session=" + short(sessionID) + "] Chain error: " + err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"error": unreachableMessage(s, err), "session_id": sessionID})
		return
	}

	// Both turns are appended ONLY after a successful reply.
	s.deps.Memory.Append(sessionID, "user", message)
	s.deps.Memory.Append(sessionID, "assistant", reply)

	c.JSON(http.StatusOK, gin.H{"reply": reply, "session_id": sessionID})
}

func (s *Server) chatReset(c *gin.Context) {
	body := readBody(c)
	sessionID := strings.TrimSpace(jsonString(body, "session_id"))
	if sessionID == "" {
		Err(c, http.StatusBadRequest, "session_id is required.")
		return
	}
	// Clearing a session that does not exist is a no-op in Python too.
	s.deps.Memory.Clear(sessionID)
	s.log.Info("[session=" + short(sessionID) + "] Session cleared.")
	c.JSON(http.StatusOK, gin.H{"ok": true, "session_id": sessionID})
}

// chatHealth reports the model backend actually in use.
//
// The response KEEPS its three keys — status, model, base_url — because the
// frontend reads them; only the values change when Gemini is configured.
func (s *Server) chatHealth(c *gin.Context) {
	model, baseURL := s.deps.Cfg.OllamaModel, s.deps.Cfg.OllamaBaseURL
	if s.deps.Gemini.Enabled() {
		model, baseURL = s.deps.Cfg.GeminiModel, s.deps.Cfg.GeminiBaseURL
	}
	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"model":    model,
		"base_url": baseURL,
	})
}

// ── helpers ─────────────────────────────────────────────────────────────────

// generateReply dispatches to whichever model backend is configured.
//
// Gemini wins when GEMINI_API_KEY is set; otherwise the local Ollama server is
// used. Both take the same arguments and return a trimmed reply, so the calling
// handler — and the response contract — is unchanged either way.
func (s *Server) generateReply(
	c *gin.Context, systemPrompt string, history []chatbot.Message, message string,
) (string, error) {
	if s.deps.Gemini.Enabled() {
		return s.deps.Gemini.Chat(c.Request.Context(), systemPrompt,
			toGeminiMessages(history), message)
	}
	return s.deps.Ollama.Chat(c.Request.Context(), systemPrompt,
		toOllamaMessages(history), message)
}

// unreachableMessage maps a backend failure onto the user-facing text.
//
// A transport failure becomes the generic "could not reach" wording; anything
// else (an empty generation, a safety refusal) is already user-facing and is
// passed through as-is.
func unreachableMessage(s *Server, err error) string {
	if s.deps.Gemini.Enabled() {
		if errors.Is(err, gemini.ErrUnreachable) {
			return gemini.UnreachableMessage
		}
		return err.Error()
	}
	if errors.Is(err, ollama.ErrUnreachable) {
		return ollama.UnreachableMessage
	}
	return err.Error()
}

func toGeminiMessages(in []chatbot.Message) []gemini.Message {
	out := make([]gemini.Message, len(in))
	for i, m := range in {
		out[i] = gemini.Message{Role: m.Role, Content: m.Content}
	}
	return out
}

// decodeWithNumbers unmarshals preserving numeric literals.
func decodeWithNumbers(raw json.RawMessage, dst any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	return dec.Decode(dst)
}

func toOllamaMessages(in []chatbot.Message) []ollama.Message {
	out := make([]ollama.Message, len(in))
	for i, m := range in {
		out[i] = ollama.Message{Role: m.Role, Content: m.Content}
	}
	return out
}

func short(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

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

// normalisePhone reproduces str(body.get("phone", "")).strip().replace(" ", "").
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
