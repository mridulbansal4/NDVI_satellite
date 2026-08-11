// Package repo holds one file's worth of hand-written SQL per table, mirroring
// backend/repositories/.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §2.3, §4.
//
// No ORM: the existing SQL uses PostGIS functions (ST_GeomFromGeoJSON,
// ST_AsGeoJSON), DISTINCT ON, ON CONFLICT … RETURNING and = ANY($1::uuid[]),
// all of which are a direct, verifiable port with pgx and a fight with GORM.
package repo

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/SanTiwari07/NDVI_satellite/internal/db"
)

// ErrNotFound is returned when a lookup finds no row. Callers map it onto the
// specific message and status the contract requires.
var ErrNotFound = errors.New("row not found")

// Store bundles every repository over one pool.
type Store struct {
	pool *db.Pool
}

// New returns a Store backed by pool.
func New(pool *db.Pool) *Store { return &Store{pool: pool} }

// numericToFloat converts a NUMERIC column to float64.
//
// §10.10: total_area is NUMERIC(10,2) and the Python casts it with float(), so
// the JSON must carry 2.5, not the string "2.50". pgtype.Numeric is the only
// scan target that round-trips the value exactly.
func numericToFloat(n pgtype.Numeric) (float64, error) {
	if !n.Valid {
		return 0, nil
	}
	f, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	return f.Float64, nil
}

// numericToString renders a NUMERIC column the way Flask's jsonify renders the
// psycopg2 Decimal it maps to: as a JSON STRING preserving the column's scale,
// so NUMERIC(10,2) 2.5 becomes "2.50".
//
// This is NOT the same as the dashboard's representation of the same column.
// services/dashboard.py explicitly calls float(farm["total_area"]), so
// /dashboard emits the number 2.5 while POST /farm emits the string "2.50".
// Both are in the frozen contract (verified against testdata/golden/
// e16_farm_success.json and e21_dashboard_success.json) and must not be
// unified, however odd the asymmetry looks.
func numericToString(n pgtype.Numeric) string {
	if !n.Valid || n.Int == nil {
		return ""
	}
	digits := n.Int.String()
	neg := strings.HasPrefix(digits, "-")
	if neg {
		digits = digits[1:]
	}
	scale := int(-n.Exp)
	if scale <= 0 {
		// No fractional part; a positive exponent means trailing zeros.
		out := digits + strings.Repeat("0", int(n.Exp))
		if neg {
			return "-" + out
		}
		return out
	}
	for len(digits) <= scale {
		digits = "0" + digits
	}
	out := digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	if neg {
		return "-" + out
	}
	return out
}

// rfc1123Date renders a DATE the way Flask's jsonify renders a Python date:
// "Sun, 15 Jun 2025 00:00:00 GMT".
//
// Used by the POST /crop response. The DASHBOARD instead uses dateString,
// because services/dashboard.py calls str(c["sowing_date"]) explicitly.
func rfc1123Date(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
}

// dateString renders a DATE column as YYYY-MM-DD.
//
// §10.10: the Python uses str(date), so these must NOT become RFC3339
// timestamps — the frontend parses them as plain dates.
func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}

func wrap(op string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

// round mirrors Python's round(x, n) closely enough for the values involved
// here (§10.10 rounds cvi_mean/ndvi to 4 dp and confidence_score to 2 dp).
func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
