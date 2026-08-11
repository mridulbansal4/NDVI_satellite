// Package geo mirrors backend/utils/geo_utils.py.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §6.1.
//
// The error strings produced here are surfaced verbatim to the frontend and are
// part of the frozen contract. Check ordering, wording and number formatting
// are all load-bearing; see geo_test.go, which asserts them against the
// responses captured from the running Flask app.
package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrNonNumericCoordinate reports a coordinate whose lon/lat entries are
// present and of the right arity but are not numbers (e.g. ["a","b"]).
//
// Parity note: Python does not handle this. validate_polygon evaluates
// `-180 <= lon <= 180` against a str and raises TypeError, and app.py calls
// validate_polygon OUTSIDE its try block, so Flask turns it into a 500. The
// handler maps this sentinel to the same 500 rather than inventing a 400.
var ErrNonNumericCoordinate = errors.New("coordinate value is not numeric")

// ValidatePolygon mirrors utils/geo_utils.validate_polygon exactly, including
// message wording and check ordering. It returns nil when the polygon is valid.
//
// Only the OUTER ring is validated; inner rings (holes) are passed through
// unchecked, and MultiPolygon is rejected. Both behaviours are preserved.
func ValidatePolygon(raw json.RawMessage) error {
	if len(raw) == 0 {
		return errors.New("Geometry must be a JSON object.")
	}

	// Check 1: must be a JSON object.
	//
	// Decoded with UseNumber so that number literals keep their original text.
	// pyNum needs that to tell a Python int from a Python float.
	probe, err := Decode(raw)
	if err != nil {
		return errors.New("Geometry must be a JSON object.")
	}
	obj, ok := probe.(map[string]any)
	if !ok {
		return errors.New("Geometry must be a JSON object.")
	}

	// Check 2: type must be exactly "Polygon".
	//
	// Python formats a missing key as the string "None" because f"{None}"
	// renders that way, so a geometry with no "type" yields
	//     Geometry type must be 'Polygon', got 'None'.
	geoType := "None"
	if v, present := obj["type"]; present {
		geoType = pyRepr(v)
	}
	if geoType != "Polygon" {
		return fmt.Errorf("Geometry type must be 'Polygon', got '%s'.", geoType)
	}

	// Check 3: coordinates must be a non-empty array.
	coordsAny, present := obj["coordinates"]
	if !present {
		return errors.New("Geometry 'coordinates' must be a non-empty array.")
	}
	coords, ok := coordsAny.([]any)
	if !ok || len(coords) == 0 {
		return errors.New("Geometry 'coordinates' must be a non-empty array.")
	}

	// Check 4: the outer ring needs at least 4 positions.
	ring, ok := coords[0].([]any)
	if !ok || len(ring) < 4 {
		return errors.New("Outer ring must have at least 4 coordinate pairs (3 unique + 1 closing).")
	}

	// Checks 5-7, per position, in order.
	for i, ptAny := range ring {
		pt, ok := ptAny.([]any)
		if !ok || len(pt) < 2 {
			return fmt.Errorf("Coordinate at index %d is not a valid [lon, lat] pair.", i)
		}
		lon, lonOK := pt[0].(json.Number)
		lat, latOK := pt[1].(json.Number)
		if !lonOK || !latOK {
			return ErrNonNumericCoordinate
		}
		lonF, err := lon.Float64()
		if err != nil {
			return ErrNonNumericCoordinate
		}
		latF, err := lat.Float64()
		if err != nil {
			return ErrNonNumericCoordinate
		}
		if !(lonF >= -180 && lonF <= 180) {
			return fmt.Errorf("Longitude %s at index %d is out of range [-180, 180].",
				pyNum(lon), i)
		}
		if !(latF >= -90 && latF <= 90) {
			return fmt.Errorf("Latitude %s at index %d is out of range [-90, 90].",
				pyNum(lat), i)
		}
	}
	return nil
}

// Decode parses a request geometry with json.Number semantics, which is what
// lets pyNum tell a JSON int literal apart from a float literal.
func Decode(raw json.RawMessage) (any, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// pyRepr renders a decoded JSON value the way a Python f-string would, which is
// only needed for the "got '<type>'" message.
func pyRepr(v any) string {
	switch t := v.(type) {
	case nil:
		return "None"
	case bool:
		if t {
			return "True"
		}
		return "False"
	case string:
		return t
	case json.Number:
		return pyNum(t)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// pyNum formats a JSON number exactly as Python's str() would.
//
// This cannot be strconv.FormatFloat(v, 'g', -1, 64) as PRD §6.1 suggests:
// Python prints 181.0, not 181. Two rules are being reproduced.
//
//  1. json.loads keeps int literals as Python ints, so "200" prints as "200"
//     (arbitrary precision, no decimal point) while "200.0" prints "200.0".
//     json.Number preserves the original token, which is how we tell them apart.
//  2. Python's float repr is shortest-round-trip, switches to exponent notation
//     when the decimal exponent is < -4 or >= 16, always keeps at least one
//     fractional digit otherwise, and renders non-finite values as
//     inf / -inf / nan.
func pyNum(n json.Number) string {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		// A Python int: printed verbatim, however many digits it has.
		return s
	}
	f, err := n.Float64()
	if err != nil {
		// Overflow: Python's json.loads yields inf/-inf rather than failing.
		if strings.HasPrefix(s, "-") {
			return "-inf"
		}
		return "inf"
	}
	return pyFloat(f)
}

func pyFloat(f float64) string {
	if math.IsNaN(f) {
		return "nan"
	}
	if math.IsInf(f, 1) {
		return "inf"
	}
	if math.IsInf(f, -1) {
		return "-inf"
	}

	// Shortest round-trip digits, so we can inspect the decimal exponent.
	sci := strconv.FormatFloat(f, 'e', -1, 64)
	exp := 0
	if i := strings.IndexByte(sci, 'e'); i >= 0 {
		exp, _ = strconv.Atoi(sci[i+1:])
	}

	if exp < -4 || exp >= 16 {
		// Python renders 1e16 as "1e+16" and 1e-5 as "1e-05": no trailing ".0"
		// on the mantissa, exponent sign always present, at least two digits.
		// Go's 'e' verb with precision -1 already matches on all three counts.
		return sci
	}

	out := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.ContainsAny(out, ".") {
		out += ".0"
	}
	return out
}
