// analyze_handler.go — E2, E3, E4, E7 and the radar routes E5, E6.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §10.3-§10.7.
//
// Two things here are easy to get wrong and are both deliberate:
//
//  1. Two DIFFERENT "GEE not initialised" wordings exist. E2 is the only route
//     using the long one. E3-E7 use the short one. They must not be unified.
//  2. /api/analyze, /api/analyze-day and /api/analyze-radar answer HTTP 200
//     with an {"error": …} body when no imagery is found (§13.2). The frontend
//     checks response.ok and then reads .error. Do not "correct" this to 4xx.

package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SanTiwari07/NDVI_satellite/internal/geo"
)

const (
	// E2 only.
	msgGEENotReadyLong = "Google Earth Engine is not initialised. Check server logs."
	// E3, E4, E5, E6, E7.
	msgGEENotReadyShort = "GEE not initialised"
)

// sampleBands is the valid set for /api/sample. The error message embeds
// Python's list repr verbatim, single quotes and all.
var sampleBands = []string{"NDVI", "EVI", "SAVI", "NDMI", "NDWI", "GNDVI", "CVI"}

func sampleBandsRepr() string {
	quoted := make([]string, len(sampleBands))
	for i, b := range sampleBands {
		quoted[i] = "'" + b + "'"
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// readBody returns the decoded top-level object, or nil when the body is
// absent or unparseable — matching Flask's request.get_json(silent=True).
func readBody(c *gin.Context) map[string]json.RawMessage {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// requireGeometry runs the shared prologue: GEE readiness, presence of the
// geometry key, then polygon validation — in that exact order.
func (s *Server) requireGeometry(c *gin.Context, notReadyMsg, missingMsg string,
) (json.RawMessage, bool) {
	if !s.deps.GEEReady.Load() {
		Err(c, http.StatusServiceUnavailable, notReadyMsg)
		return nil, false
	}
	body := readBody(c)
	geometry, present := body["geometry"]
	if body == nil || !present {
		Err(c, http.StatusBadRequest, missingMsg)
		return nil, false
	}
	if err := geo.ValidatePolygon(geometry); err != nil {
		// Parity: a non-numeric coordinate raises TypeError inside
		// validate_polygon, and app.py calls it outside its try block, so
		// Flask turns it into a 500.
		if errors.Is(err, geo.ErrNonNumericCoordinate) {
			Err(c, http.StatusInternalServerError, "Internal Server Error")
			return nil, false
		}
		s.log.Warn("Invalid polygon received: " + err.Error())
		Err(c, http.StatusBadRequest, err.Error())
		return nil, false
	}
	return geometry, true
}

// ── E2 — POST /api/analyze ──────────────────────────────────────────────────

func (s *Server) analyze(c *gin.Context) {
	_, ok := s.requireGeometry(c, msgGEENotReadyLong,
		"Request body must contain a 'geometry' key with a GeoJSON Polygon.")
	if !ok {
		return
	}
	s.notImplemented(c, "Sentinel-2 pipeline (Phase 3)")
}

// ── E3 — POST /api/analyze-dates ────────────────────────────────────────────

func (s *Server) analyzeDates(c *gin.Context) {
	_, ok := s.requireGeometry(c, msgGEENotReadyShort, "Missing geometry")
	if !ok {
		return
	}
	s.notImplemented(c, "available dates (Phase 3)")
}

// ── E4 — POST /api/analyze-day ──────────────────────────────────────────────

func (s *Server) analyzeDay(c *gin.Context) {
	if !s.deps.GEEReady.Load() {
		Err(c, http.StatusServiceUnavailable, msgGEENotReadyShort)
		return
	}
	body := readBody(c)
	geometry, hasGeom := body["geometry"]
	_, hasDate := body["date"]
	if body == nil || !hasGeom || !hasDate {
		Err(c, http.StatusBadRequest, "Missing geometry or date")
		return
	}
	if err := geo.ValidatePolygon(geometry); err != nil {
		if errors.Is(err, geo.ErrNonNumericCoordinate) {
			Err(c, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		Err(c, http.StatusBadRequest, err.Error())
		return
	}
	s.notImplemented(c, "single-day analysis (Phase 3)")
}

// ── E5 — POST /api/analyze-radar-dates ──────────────────────────────────────

func (s *Server) analyzeRadarDates(c *gin.Context) {
	_, ok := s.requireGeometry(c, msgGEENotReadyShort, "Missing geometry")
	if !ok {
		return
	}
	s.notImplemented(c, "Sentinel-1 dates (Phase 4)")
}

// ── E6 — POST /api/analyze-radar ────────────────────────────────────────────

func (s *Server) analyzeRadar(c *gin.Context) {
	_, ok := s.requireGeometry(c, msgGEENotReadyShort, "Missing geometry")
	if !ok {
		return
	}
	s.notImplemented(c, "Sentinel-1 pipeline (Phase 4)")
}

// ── E7 — GET /api/sample (§10.7) ────────────────────────────────────────────
//
// Check ordering is load-bearing: GEE readiness, THEN the cached-image check,
// THEN coordinate parsing, THEN band validation.
func (s *Server) sample(c *gin.Context) {
	if !s.deps.GEEReady.Load() {
		Err(c, http.StatusServiceUnavailable, msgGEENotReadyShort)
		return
	}

	// Phase 3 replaces this with the ExprCache "__last__" lookup (§7.7).
	// Until the pipeline exists nothing is ever cached, so this is the
	// permanent branch for now — and it is the correct one for a cold server.
	Err(c, http.StatusNotFound, "No analysis available. Run an analysis first.")
}
