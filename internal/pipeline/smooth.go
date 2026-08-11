package pipeline

import (
	"encoding/json"
	"math"
)

// GridFeature is one output cell. Properties are ordered-insensitive but their
// VALUES are load-bearing: a nil means "masked / no data" and must serialise as
// JSON null, never 0 (§13.1).
type GridFeature struct {
	Type       string          `json:"type"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties map[string]any  `json:"properties"`
}

// GridCollection is the FeatureCollection returned by the analysis endpoints.
type GridCollection struct {
	Type     string        `json:"type"`
	Features []GridFeature `json:"features"`
}

// SmoothGridValues is a direct port of grid_service._smooth_grid_values (§7.6).
//
// For each cell i and band b:
//
//	smoothed[i] = Σⱼ wᵢⱼ·valueⱼ / Σⱼ wᵢⱼ ,  wᵢⱼ = exp(−d²ᵢⱼ / 2σ²)
//
// with σ = sigmaFactor × average nearest-neighbour centroid spacing, and a 3σ
// cutoff. Several details are load-bearing for numeric parity and are called
// out inline; changing any of them changes the last bits of every cell value.
//
// Complexity is O(n²) with n ≤ MaxGridCells (2000), i.e. ≤ 4M distance
// evaluations per request, which is well under 50 ms. PRD §7.6 explicitly says
// not to introduce a k-d tree in the initial port.
func SmoothGridValues(features []GridFeature, bands []string, sigmaFactor float64) []GridFeature {
	n := len(features)
	if n < 2 {
		return features
	}

	// Centroid = unweighted mean of the ring's coordinates INCLUDING the
	// duplicated closing vertex. That double-counts one corner and shifts the
	// centroid slightly; it is a quirk of the Python, and reproducing it is
	// required for parity.
	cx := make([]float64, n)
	cy := make([]float64, n)
	for i := range features {
		ring, ok := outerRing(features[i].Geometry)
		if !ok || len(ring) == 0 {
			continue
		}
		var sx, sy float64
		for _, c := range ring {
			sx += c[0]
			sy += c[1]
		}
		cx[i] = sx / float64(len(ring))
		cy[i] = sy / float64(len(ring))
	}

	// Estimate spacing from a sample of cells, not all of them.
	step := n / 20
	if step < 1 {
		step = 1
	}
	var totalNN float64
	var cnt int
	for i := 0; i < n; i += step {
		minD := math.Inf(1)
		for j := 0; j < n; j++ {
			if j == i {
				continue
			}
			dx, dy := cx[i]-cx[j], cy[i]-cy[j]
			if d := math.Sqrt(dx*dx + dy*dy); d < minD {
				minD = d
			}
		}
		if !math.IsInf(minD, 1) {
			totalNN += minD
			cnt++
		}
	}
	if cnt == 0 {
		return features
	}

	sigma := (totalNN / float64(cnt)) * sigmaFactor
	inv2sig2 := 1.0 / (2.0 * sigma * sigma)
	searchR2 := (sigma * 3.0) * (sigma * 3.0)

	// Read every neighbour value up front. Note these are the ALREADY-ROUNDED
	// 4 dp values: reduce_grid_values rounds before smoothing, so smoothing
	// operates on rounded inputs. Smoothing the unrounded values would drift.
	vals := make([]map[string]*float64, n)
	for i := range features {
		m := make(map[string]*float64, len(bands))
		for _, b := range bands {
			m[b] = floatOrNil(features[i].Properties[b])
		}
		vals[i] = m
	}

	out := make([]GridFeature, n)
	for i := range features {
		props := make(map[string]any, len(features[i].Properties))
		for k, v := range features[i].Properties {
			props[k] = v
		}

		for _, band := range bands {
			if vals[i][band] == nil {
				continue // a null cell stays null for that band
			}
			var wSum, vSum float64
			// Iterate j in slice order, matching Python's range(n): floating
			// point addition is not associative, so summation order determines
			// the last bits (§7.6, parity tolerance 1e-9).
			for j := 0; j < n; j++ {
				vj := vals[j][band]
				if vj == nil {
					continue
				}
				dx, dy := cx[i]-cx[j], cy[i]-cy[j]
				d2 := dx*dx + dy*dy
				if d2 > searchR2 {
					continue
				}
				// j == i IS included, with weight exp(0) = 1.
				w := math.Exp(-d2 * inv2sig2)
				wSum += w
				vSum += w * (*vj)
			}
			if wSum > 0 {
				props[band] = Round4(vSum / wSum)
			}
			// If wSum == 0 the original value is left untouched.
		}

		out[i] = GridFeature{
			Type:       "Feature",
			Geometry:   features[i].Geometry,
			Properties: props,
		}
	}
	return out
}

// outerRing extracts coordinates[0] from a GeoJSON polygon geometry.
func outerRing(raw json.RawMessage) ([][2]float64, bool) {
	var g struct {
		Coordinates [][][]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &g); err != nil || len(g.Coordinates) == 0 {
		return nil, false
	}
	ring := make([][2]float64, 0, len(g.Coordinates[0]))
	for _, p := range g.Coordinates[0] {
		if len(p) < 2 {
			continue
		}
		ring = append(ring, [2]float64{p[0], p[1]})
	}
	return ring, true
}

// floatOrNil reads a property that may legitimately be null.
func floatOrNil(v any) *float64 {
	switch t := v.(type) {
	case nil:
		return nil
	case float64:
		return &t
	case *float64:
		return t
	case int:
		f := float64(t)
		return &f
	case json.Number:
		f, err := t.Float64()
		if err != nil {
			return nil
		}
		return &f
	}
	return nil
}
