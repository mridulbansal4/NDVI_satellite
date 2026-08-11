package eeexpr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// assertGraphEqualsGolden compares a Go-built graph against a fixture captured
// from the real Python client, SEMANTICALLY rather than textually (§5.4).
//
// Both sides are inlined into a nested tree first, because reference keys are
// assigned in traversal order and Go's order legitimately differs from
// Python's. Two graphs are equal iff their inlined trees are equal.
func assertGraphEqualsGolden(t *testing.T, name string, got Node) {
	t.Helper()

	expr, err := Compile(got)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	gotTree, err := CanonicaliseExpression(expr)
	if err != nil {
		t.Fatalf("canonicalise Go graph: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Skipf("fixture %s not present: %v", name, err)
	}
	wantTree, err := Canonicalise(raw)
	if err != nil {
		t.Fatalf("canonicalise fixture %s: %v", name, err)
	}

	if diff := cmp.Diff(wantTree, gotTree); diff != "" {
		t.Errorf("graph does not match fixture %s (-python +go):\n%s", name, diff)
	}
}

// fixtureGeom is the pinned polygon dump_graphs.py serialised, byte-identical
// to testdata/polygons/nashik_vineyard.json.
func fixtureGeom() Geometry {
	return Polygon([][][]float64{{
		{73.79, 20.011},
		{73.7916, 20.011},
		{73.7916, 20.0122},
		{73.79, 20.0122},
		{73.79, 20.011},
	}})
}

func TestGeometryGraph(t *testing.T) {
	assertGraphEqualsGolden(t, "geometry", fixtureGeom().N)
}

func TestCompileHoistsFunctionBodies(t *testing.T) {
	b := NewBuilder()
	coll := ImageCollectionLoad("X").Map(b, func(img Image) Image {
		return img.DivideNum(2)
	})
	expr, err := Compile(coll.N)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(expr)
	var decoded struct {
		Values map[string]json.RawMessage `json:"values"`
	}
	_ = json.Unmarshal(raw, &decoded)

	// The body must be a reference key, never an inline node — the wire format
	// requires it, and Earth Engine rejects an inlined body.
	found := false
	for _, v := range decoded.Values {
		var m map[string]json.RawMessage
		_ = json.Unmarshal(v, &m)
		fiv, ok := m["functionInvocationValue"]
		if !ok {
			continue
		}
		var inv struct {
			Arguments map[string]json.RawMessage `json:"arguments"`
		}
		_ = json.Unmarshal(fiv, &inv)
		ba, ok := inv.Arguments["baseAlgorithm"]
		if !ok {
			continue
		}
		var fd struct {
			FunctionDefinitionValue struct {
				Body json.RawMessage `json:"body"`
			} `json:"functionDefinitionValue"`
		}
		_ = json.Unmarshal(ba, &fd)
		var s string
		if err := json.Unmarshal(fd.FunctionDefinitionValue.Body, &s); err != nil {
			t.Errorf("functionDefinitionValue body is not a string reference: %s",
				fd.FunctionDefinitionValue.Body)
		}
		if _, present := decoded.Values[s]; !present {
			t.Errorf("body reference %q does not resolve", s)
		}
		found = true
	}
	if !found {
		t.Error("no Collection.map baseAlgorithm found in the compiled graph")
	}
}

func TestCompileInternsSharedSubtrees(t *testing.T) {
	shared := ImageLoad("A").Select("B1")
	root := shared.Add(shared)

	expr, err := Compile(root.N)
	if err != nil {
		t.Fatal(err)
	}
	// The shared Select subtree must be emitted once and referenced twice.
	if len(expr.Values) < 2 {
		t.Errorf("expected the shared subtree to be hoisted, got %d values", len(expr.Values))
	}
	tree, err := CanonicaliseExpression(expr)
	if err != nil {
		t.Fatal(err)
	}
	// Canonicalisation must still reproduce the full tree on both branches.
	inv := tree.(map[string]any)["invoke"].(map[string]any)
	args := inv["args"].(map[string]any)
	if diff := cmp.Diff(args["image1"], args["image2"]); diff != "" {
		t.Errorf("interned branches differ after canonicalisation:\n%s", diff)
	}
}

func TestCanonicaliseRejectsDanglingReference(t *testing.T) {
	_, err := Canonicalise(json.RawMessage(
		`{"values":{"0":{"valueReference":"9"}},"result":"0"}`))
	if err == nil {
		t.Error("expected an error for a dangling reference")
	}
}

func TestCanonicaliseCollapsesConstantArrays(t *testing.T) {
	// arrayValue-of-constants and a single constantValue array mean the same
	// thing to Earth Engine; the Python client emits whichever falls out of its
	// interning pass, so they must compare equal.
	a, err := Canonicalise(json.RawMessage(
		`{"values":{"0":{"arrayValue":{"values":[{"constantValue":1},{"constantValue":2}]}}},"result":"0"}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Canonicalise(json.RawMessage(
		`{"values":{"0":{"constantValue":[1,2]}},"result":"0"}`))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(a, b); diff != "" {
		t.Errorf("constant array forms should canonicalise identically:\n%s", diff)
	}
}
