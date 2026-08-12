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
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/SanTiwari07/NDVI_satellite/internal/geo"
	"github.com/SanTiwari07/NDVI_satellite/internal/pipeline"
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

// bodyCacheKey holds the decoded request body in the Gin context.
//
// The body stream can only be read once, but several handlers need it twice —
// requireGeometry pulls out "geometry" and the caller then wants "date". Flask
// has no such problem because request.get_json() memoises.
const bodyCacheKey = "__decoded_body__"

// readBody returns the decoded top-level object, or nil when the body is
// absent or unparseable — matching Flask's request.get_json(silent=True).
//
// The read is bounded by middleware.BodyLimit, which wraps Request.Body in a
// MaxBytesReader. Without that, this ReadAll is unbounded on routes that take
// no credentials: ReadTimeout caps how LONG a client may send, not how much,
// so a few concurrent large POSTs are an out-of-memory kill. An over-limit body
// surfaces here as a read error and is treated as an absent body, which is the
// same path an unparseable body already took.
func readBody(c *gin.Context) map[string]json.RawMessage {
	if cached, ok := c.Get(bodyCacheKey); ok {
		m, _ := cached.(map[string]json.RawMessage)
		return m
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		c.Set(bodyCacheKey, map[string]json.RawMessage(nil))
		return nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		c.Set(bodyCacheKey, map[string]json.RawMessage(nil))
		return nil
	}
	c.Set(bodyCacheKey, m)
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
	geometry, ok := s.requireGeometry(c, msgGEENotReadyLong,
		"Request body must contain a 'geometry' key with a GeoJSON Polygon.")
	if !ok {
		return
	}
	s.log.Info("Analysis request received. Converting geometry to EE…")

	result, err := s.deps.Analyzer.Analyze(c.Request.Context(), geometry)
	if err != nil {
		s.log.Error("Pipeline error: " + err.Error())
		Err(c, http.StatusInternalServerError, "Pipeline error: "+err.Error())
		return
	}
	if result == nil {
		// §13.2: HTTP 200 with an error body. The frontend checks response.ok
		// and then reads .error. Do NOT "correct" this to 404.
		Err(c, http.StatusOK,
			"No cloud-free Sentinel-2 imagery found for this area in the last 3 months.")
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── E3 — POST /api/analyze-dates ────────────────────────────────────────────

func (s *Server) analyzeDates(c *gin.Context) {
	geometry, ok := s.requireGeometry(c, msgGEENotReadyShort, "Missing geometry")
	if !ok {
		return
	}
	dates, err := s.deps.Analyzer.AvailableDates(c.Request.Context(), geometry)
	if err != nil {
		s.log.Error("Error fetching dates: " + err.Error())
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"dates": dates})
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
	var date string
	if raw, present := body["date"]; present {
		_ = json.Unmarshal(raw, &date)
	}

	result, err := s.deps.Analyzer.AnalyzeDay(c.Request.Context(), geometry, date)
	if err != nil {
		s.log.Error("Day analysis error: " + err.Error())
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	if result == nil {
		Err(c, http.StatusOK, "No imagery found for date "+date)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ── E5 — POST /api/analyze-radar-dates ──────────────────────────────────────

func (s *Server) analyzeRadarDates(c *gin.Context) {
	geometry, ok := s.requireGeometry(c, msgGEENotReadyShort, "Missing geometry")
	if !ok {
		return
	}
	dates, err := s.deps.Analyzer.RadarAvailableDates(c.Request.Context(), geometry)
	if err != nil {
		s.log.Error("Error fetching S1 dates: " + err.Error())
		Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"dates": dates})
}

// ── E6 — POST /api/analyze-radar ────────────────────────────────────────────

func (s *Server) analyzeRadar(c *gin.Context) {
	// requireGeometry consumed the body, so re-read the optional date from the
	// copy it stashed.
	geometry, ok := s.requireGeometry(c, msgGEENotReadyShort, "Missing geometry")
	if !ok {
		return
	}
	var date *string
	if raw, present := readBody(c)["date"]; present {
		var d string
		if err := json.Unmarshal(raw, &d); err == nil {
			date = &d
		}
	}

	result, err := s.deps.Analyzer.AnalyzeRadar(c.Request.Context(), geometry, date)
	if err != nil {
		s.log.Error("Radar pipeline error: " + err.Error())
		Err(c, http.StatusInternalServerError, "Radar pipeline error: "+err.Error())
		return
	}
	if result == nil {
		// Both halves of app.py's inline conditional (§10.6).
		who := "this area"
		if date != nil {
			who = *date
		}
		Err(c, http.StatusOK, "No Sentinel-1 imagery found for "+who+".")
		return
	}
	c.JSON(http.StatusOK, result)
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

	// §7.7: the cached-image check precedes coordinate parsing.
	if _, cached := s.deps.Analyzer.Cache.Get(pipeline.LastKey); !cached {
		Err(c, http.StatusNotFound, "No analysis available. Run an analysis first.")
		return
	}

	latStr, lngStr := c.Query("lat"), c.Query("lng")
	lat, errLat := strconv.ParseFloat(latStr, 64)
	lng, errLng := strconv.ParseFloat(lngStr, 64)
	if latStr == "" || lngStr == "" || errLat != nil || errLng != nil {
		Err(c, http.StatusBadRequest, "lat and lng are required numeric parameters.")
		return
	}

	band := strings.ToUpper(c.DefaultQuery("band", "NDVI"))
	valid := false
	for _, b := range sampleBands {
		if band == b {
			valid = true
			break
		}
	}
	if !valid {
		Err(c, http.StatusBadRequest, "Invalid band. Must be one of: "+sampleBandsRepr())
		return
	}

	value, _ := s.deps.Analyzer.Sample(c.Request.Context(), lat, lng, band)
	c.JSON(http.StatusOK, gin.H{"value": value, "band": band})
}
