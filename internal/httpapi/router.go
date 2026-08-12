// Package httpapi is the HTTP layer. It never builds Earth Engine expressions
// and never talks to GEE directly — that is internal/gee and internal/pipeline.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §3.2, §10.1.
package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/SanTiwari07/NDVI_satellite/internal/chatbot"
	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/firebase"
	"github.com/SanTiwari07/NDVI_satellite/internal/gemini"
	"github.com/SanTiwari07/NDVI_satellite/internal/httpapi/middleware"
	"github.com/SanTiwari07/NDVI_satellite/internal/logging"
	"github.com/SanTiwari07/NDVI_satellite/internal/ollama"
	"github.com/SanTiwari07/NDVI_satellite/internal/pipeline"
	"github.com/SanTiwari07/NDVI_satellite/internal/service"
)

// Deps is everything the HTTP layer needs. Dependencies not yet built in this
// phase are nil, and the handlers that need them answer 501.
type Deps struct {
	Cfg *config.Config
	Log *slog.Logger

	// GEEReady is flipped by the startup probe goroutine (§5.5). It is read on
	// every analysis request, so it must be atomic rather than a plain bool.
	GEEReady      *atomic.Bool
	FirebaseReady *atomic.Bool

	// Analyzer runs the Sentinel-2 and Sentinel-1 pipelines. Nil until the EE
	// session is established, in which case GEEReady stays false and the
	// analysis routes answer 503 before ever dereferencing it.
	Analyzer *pipeline.Analyzer

	// Memory, Gemini and Ollama back the chatbot. All are non-nil; an
	// unreachable model surfaces as a 502, not a panic.
	//
	// Gemini is preferred when GEMINI_API_KEY is set, otherwise the local
	// Ollama server is used. Keeping both means a developer without a key still
	// has a working chatbot, and switching back is a config change.
	Memory *chatbot.Memory
	Gemini *gemini.Client
	Ollama *ollama.Client

	// Onboarding is nil when DATABASE_URL is unset, in which case every route
	// that needs it answers 500 with an actionable message rather than
	// panicking. That mirrors the Python, which logs "[DB] pool init skipped"
	// at startup and then fails per request.
	Onboarding *service.Onboarding

	// Firebase and SMS are always non-nil. Firebase being unconfigured is the
	// normal dev state and surfaces as firebase_ready=false (§13.7).
	Firebase *firebase.Admin
	SMS      *service.SMSService
	PinAPI   *service.PinCodeClient
}

// Server owns the route table.
type Server struct {
	deps Deps
	log  *slog.Logger
}

// New builds the Gin engine with all 24 routes from §10.1.
func New(d Deps) *gin.Engine {
	if d.GEEReady == nil {
		d.GEEReady = &atomic.Bool{}
	}
	if d.FirebaseReady == nil {
		d.FirebaseReady = &atomic.Bool{}
	}
	if d.Memory == nil {
		d.Memory = chatbot.NewMemoryWithLimits(d.Cfg.ChatbotMaxHistory,
			d.Cfg.ChatSessionTTL, d.Cfg.ChatMaxSessions)
	}
	if d.Ollama == nil {
		d.Ollama = ollama.New(d.Cfg.OllamaBaseURL, d.Cfg.OllamaModel,
			d.Cfg.OllamaTemperature, d.Cfg.OllamaMaxTokens)
	}
	if d.Gemini == nil {
		d.Gemini = gemini.New(d.Cfg.GeminiAPIKey, d.Cfg.GeminiModel,
			d.Cfg.GeminiBaseURL, d.Cfg.OllamaTemperature, d.Cfg.OllamaMaxTokens,
			d.Cfg.GeminiThinkingBudget)
	}
	if d.PinAPI == nil {
		d.PinAPI = service.NewPinCodeClient()
	}
	if d.SMS == nil {
		d.SMS = service.NewSMSService(d.Cfg, logging.Named(d.Log, "app.sms"))
	}
	if d.Firebase == nil {
		d.Firebase = firebase.NewAdmin(d.Cfg.ServiceAccountKey,
			d.Cfg.FirebaseProjectID, logging.Named(d.Log, "app.auth"))
	}
	s := &Server{deps: d, log: logging.Named(d.Log, "app")}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.requestLogger())
	r.Use(middleware.CORS(d.Cfg.CORSOrigins))
	r.Use(middleware.BodyLimit(d.Cfg.MaxRequestBytes))

	jwtAuth := middleware.JWT([]byte(d.Cfg.JWTSecret))

	// Rate limiting covers the routes that cost money per call and take no
	// credentials: the analysis endpoints (Earth Engine quota), the chatbot
	// (Gemini tokens), and /auth + /api/auth (an SMS message, or a 32 MB scrypt
	// on every signup). /health and /dashboard are left alone — /health is what
	// a load balancer polls, and /dashboard already requires a JWT.
	throttle := func(c *gin.Context) { c.Next() }
	if d.Cfg.RateLimitEnabled {
		throttle = middleware.NewRateLimiter(d.Cfg.RateLimitPerMin, d.Cfg.RateLimitBurst).Middleware()
	}
	// Separate from the rate limit: this bounds how many analyses run AT ONCE,
	// which is what actually protects the Earth Engine quota.
	analyzeGate := middleware.InFlight(d.Cfg.AnalyzeMaxInFlight)

	// ── E1 ──────────────────────────────────────────────────────────────
	r.GET("/health", s.health)

	// ── E2-E10: /api/* ──────────────────────────────────────────────────
	api := r.Group("/api", throttle)
	{
		api.POST("/analyze", analyzeGate, s.analyze)
		api.POST("/analyze-dates", analyzeGate, s.analyzeDates)
		api.POST("/analyze-day", analyzeGate, s.analyzeDay)
		api.POST("/analyze-radar-dates", analyzeGate, s.analyzeRadarDates)
		api.POST("/analyze-radar", analyzeGate, s.analyzeRadar)
		api.GET("/sample", s.sample)
		api.POST("/auth/verify-token", s.verifyToken)
		api.POST("/auth/send-otp", s.sendOTP)
		api.POST("/auth/verify-otp", s.verifyOTP)
	}

	// ── E11-E12: /auth/* ────────────────────────────────────────────────
	auth := r.Group("/auth", throttle)
	{
		auth.POST("/signup", s.signup)
		auth.POST("/login", s.login)
	}

	// ── E13-E15: /farmer/* ──────────────────────────────────────────────
	farmer := r.Group("/farmer")
	{
		farmer.POST("/basic-details", jwtAuth, s.basicDetails)
		farmer.POST("/location", jwtAuth, s.location)
		farmer.GET("/pincode/:pin_code", s.pincode) // no auth, matches Flask
	}

	// ── E16-E20 ─────────────────────────────────────────────────────────
	// Flask registers these blueprints with url_prefix="/farm" and route "",
	// which serves BOTH /farm and /farm/. Gin needs both registered explicitly
	// or it 301-redirects, which would break the axios client's POST.
	for path, h := range map[string]gin.HandlerFunc{
		"/farm":       s.createFarm,
		"/crop":       s.addCrop,
		"/irrigation": s.addIrrigation,
		"/soil":       s.addSoil,
		"/consent":    s.submitConsent,
	} {
		r.POST(path, jwtAuth, h)
		r.POST(path+"/", jwtAuth, h)
	}

	// ── E21 ─────────────────────────────────────────────────────────────
	r.GET("/dashboard", jwtAuth, s.dashboard)
	r.GET("/dashboard/", jwtAuth, s.dashboard)

	// ── E22-E24: /chatbot/* ─────────────────────────────────────────────
	chat := r.Group("/chatbot", throttle)
	{
		chat.POST("/chat", s.chat)
		chat.POST("/reset", s.chatReset)
		chat.GET("/health", s.chatHealth)
	}

	return r
}

// requestLogger logs one line per request in the Python format, and records the
// wall-clock duration so slow pipeline stages stay attributable (§13.3).
//
// The request id ties this line to the pipeline-stage lines a slow analysis
// emits; without it, concurrent analyses interleave in the log and there is no
// way to tell which stage timing belonged to which request.
func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		rid := requestID(c)
		c.Set(requestIDKey, rid)
		c.Header("X-Request-Id", rid)

		c.Next()

		s.log.Info("request",
			"id", rid,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ms", time.Since(started).Milliseconds(),
		)
	}
}

// requestIDKey is the Gin-context key holding the per-request correlation id.
const requestIDKey = "__request_id__"

// requestID honours an inbound X-Request-Id so a trace started at the proxy
// survives into these logs, and mints one otherwise.
func requestID(c *gin.Context) string {
	if v := strings.TrimSpace(c.GetHeader("X-Request-Id")); v != "" {
		// Bound it: the value is echoed into a response header and the log, and
		// an unbounded caller-supplied string is a log-injection vector.
		if len(v) > 64 {
			v = v[:64]
		}
		return sanitiseID(v)
	}
	return uuid.NewString()
}

// sanitiseID strips anything that could forge a log line or a header.
func sanitiseID(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return uuid.NewString()
	}
	return string(out)
}

// ── E1 — GET /health (§10.2) ────────────────────────────────────────────────
//
// Always 200. `project` is GEE_PROJECT_ID and may be empty. Python emits the
// raw value of the env var, so an unset project serialises as JSON null.
func (s *Server) health(c *gin.Context) {
	var project any
	if s.deps.Cfg.GEEProjectID != "" {
		project = s.deps.Cfg.GEEProjectID
	}
	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"gee_ready":      s.deps.GEEReady.Load(),
		"firebase_ready": s.deps.FirebaseReady.Load(),
		"project":        project,
	})
}
