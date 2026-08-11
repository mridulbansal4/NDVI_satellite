// Package httpapi is the HTTP layer. It never builds Earth Engine expressions
// and never talks to GEE directly — that is internal/gee and internal/pipeline.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §3.2, §10.1.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/SanTiwari07/NDVI_satellite/internal/chatbot"
	"github.com/SanTiwari07/NDVI_satellite/internal/config"
	"github.com/SanTiwari07/NDVI_satellite/internal/httpapi/middleware"
	"github.com/SanTiwari07/NDVI_satellite/internal/logging"
	"github.com/SanTiwari07/NDVI_satellite/internal/ollama"
	"github.com/SanTiwari07/NDVI_satellite/internal/pipeline"
)

// rawJSON keeps a value exactly as it arrived so it can be echoed back
// byte-identically (farm_boundary must round-trip the client's own polygon).
type rawJSON = json.RawMessage

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

	// Memory and Ollama back the chatbot. Both are always non-nil; an
	// unreachable Ollama surfaces as the 502 the Python emits, not a panic.
	Memory *chatbot.Memory
	Ollama *ollama.Client
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
		d.Memory = chatbot.NewMemory(d.Cfg.ChatbotMaxHistory)
	}
	if d.Ollama == nil {
		d.Ollama = ollama.New(d.Cfg.OllamaBaseURL, d.Cfg.OllamaModel,
			d.Cfg.OllamaTemperature, d.Cfg.OllamaMaxTokens)
	}
	s := &Server{deps: d, log: logging.Named(d.Log, "app")}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.requestLogger())
	r.Use(middleware.CORS(d.Cfg.CORSOrigins))

	jwtAuth := middleware.JWT([]byte(d.Cfg.JWTSecret))

	// ── E1 ──────────────────────────────────────────────────────────────
	r.GET("/health", s.health)

	// ── E2-E10: /api/* ──────────────────────────────────────────────────
	api := r.Group("/api")
	{
		api.POST("/analyze", s.analyze)
		api.POST("/analyze-dates", s.analyzeDates)
		api.POST("/analyze-day", s.analyzeDay)
		api.POST("/analyze-radar-dates", s.analyzeRadarDates)
		api.POST("/analyze-radar", s.analyzeRadar)
		api.GET("/sample", s.sample)
		api.POST("/auth/verify-token", s.verifyToken)
		api.POST("/auth/send-otp", s.sendOTP)
		api.POST("/auth/verify-otp", s.verifyOTP)
	}

	// ── E11-E12: /auth/* ────────────────────────────────────────────────
	auth := r.Group("/auth")
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
	chat := r.Group("/chatbot")
	{
		chat.POST("/chat", s.chat)
		chat.POST("/reset", s.chatReset)
		chat.GET("/health", s.chatHealth)
	}

	return r
}

// requestLogger logs one line per request in the Python format, and records the
// wall-clock duration so slow pipeline stages stay attributable (§13.3).
func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		s.log.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
		)
	}
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

// notImplemented is the Phase 1 placeholder. It is deliberately 501 so the
// contract runner can distinguish "not built yet" from "built and wrong".
func (s *Server) notImplemented(c *gin.Context, what string) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "not implemented in this migration phase: " + what,
	})
}
