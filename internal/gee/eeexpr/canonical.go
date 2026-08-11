package eeexpr

import (
	"encoding/json"
	"fmt"
)

// Canonicalise inlines an expression graph into a single nested tree by
// following every valueReference edge, so that two graphs can be compared for
// meaning rather than for byte layout.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §5.4.
//
// This is necessary because reference keys ("0", "1", …) are assigned by the
// serialiser in traversal order, and Go's traversal order legitimately differs
// from Python's. Two graphs are equal iff their inlined trees are equal.
//
// Two further normalisations are applied so that representations which mean the
// same thing to Earth Engine compare equal:
//
//   - An arrayValue whose elements are all constantValue collapses into a
//     single constantValue holding the array. The Python client emits whichever
//     form falls out of its interning pass, so `["B8","B4"]` may appear either
//     way depending on whether an element was shared.
//   - JSON numbers are normalised through float64, so 20 and 20.0 compare equal.
//
// functionDefinitionValue bodies are resolved from their string reference into
// the inlined subtree. argumentReference nodes are left alone — they are names,
// not edges.
func Canonicalise(raw json.RawMessage) (any, error) {
	var expr struct {
		Values map[string]json.RawMessage `json:"values"`
		Result string                     `json:"result"`
	}
	if err := json.Unmarshal(raw, &expr); err != nil {
		return nil, fmt.Errorf("canonicalise: %w", err)
	}
	root, ok := expr.Values[expr.Result]
	if !ok {
		return nil, fmt.Errorf("canonicalise: result key %q not present in values", expr.Result)
	}

	c := &canonicaliser{values: expr.Values, seen: map[string]int{}}
	return c.node(root)
}

// scope binds a function-definition parameter name to a positional placeholder
// so that graphs are compared up to ALPHA-EQUIVALENCE.
//
// The Python client names mapping variables _MAPPING_VAR_<n>_<i>, where <n>
// depends on how many CustomFunctions had been constructed earlier in the
// session. The same logical graph can therefore carry _MAPPING_VAR_0_0 in one
// capture and _MAPPING_VAR_2_0 in another, purely as an artefact of build
// order. Earth Engine does not care: the name only has to agree between a
// funcdef's argumentNames and the argumentReference nodes in its body.
//
// Renaming both sides to $<depth>_<index> makes two lambdas that differ only in
// their parameter names compare equal, while a body that references the WRONG
// variable still fails — which is the bug worth catching.
type scope struct {
	names map[string]string
	prev  *scope
}

func (s *scope) lookup(name string) (string, bool) {
	for cur := s; cur != nil; cur = cur.prev {
		if v, ok := cur.names[name]; ok {
			return v, true
		}
	}
	return "", false
}

// CanonicaliseExpression is the Compile-output equivalent of Canonicalise.
func CanonicaliseExpression(e *Expression) (any, error) {
	raw, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return Canonicalise(raw)
}

type canonicaliser struct {
	values map[string]json.RawMessage
	seen   map[string]int
	scope  *scope
	depth  int
}

const maxRefDepth = 64

func (c *canonicaliser) node(raw json.RawMessage) (any, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	switch {
	case has(m, "valueReference"):
		var key string
		if err := json.Unmarshal(m["valueReference"], &key); err != nil {
			return nil, err
		}
		c.seen[key]++
		if c.seen[key] > maxRefDepth {
			return nil, fmt.Errorf("canonicalise: reference cycle at %q", key)
		}
		target, ok := c.values[key]
		if !ok {
			return nil, fmt.Errorf("canonicalise: dangling reference %q", key)
		}
		out, err := c.node(target)
		c.seen[key]--
		return out, err

	case has(m, "constantValue"):
		var v any
		if err := json.Unmarshal(m["constantValue"], &v); err != nil {
			return nil, err
		}
		return map[string]any{"const": normaliseNumbers(v)}, nil

	case has(m, "integerValue"):
		// Integers travel as strings in this format.
		var s string
		if err := json.Unmarshal(m["integerValue"], &s); err != nil {
			return nil, err
		}
		var f float64
		if _, err := fmt.Sscanf(s, "%g", &f); err != nil {
			return nil, err
		}
		return map[string]any{"const": f}, nil

	case has(m, "argumentReference"):
		var name string
		if err := json.Unmarshal(m["argumentReference"], &name); err != nil {
			return nil, err
		}
		// Rewrite to the positional placeholder bound by the enclosing
		// funcdef. A reference with no binding in scope is left as-is so the
		// comparison still flags it.
		if placeholder, ok := c.scope.lookup(name); ok {
			return map[string]any{"arg": placeholder}, nil
		}
		return map[string]any{"arg": name}, nil

	case has(m, "arrayValue"):
		var av struct {
			Values []json.RawMessage `json:"values"`
		}
		if err := json.Unmarshal(m["arrayValue"], &av); err != nil {
			return nil, err
		}
		items := make([]any, len(av.Values))
		allConst := true
		consts := make([]any, len(av.Values))
		for i, it := range av.Values {
			v, err := c.node(it)
			if err != nil {
				return nil, err
			}
			items[i] = v
			if mm, ok := v.(map[string]any); ok && len(mm) == 1 {
				if cv, isConst := mm["const"]; isConst {
					consts[i] = cv
					continue
				}
			}
			allConst = false
		}
		if allConst {
			// Same meaning as a constantValue holding the whole array.
			return map[string]any{"const": consts}, nil
		}
		return map[string]any{"array": items}, nil

	case has(m, "dictionaryValue"):
		var dv struct {
			Values map[string]json.RawMessage `json:"values"`
		}
		if err := json.Unmarshal(m["dictionaryValue"], &dv); err != nil {
			return nil, err
		}
		out := map[string]any{}
		for k, v := range dv.Values {
			cv, err := c.node(v)
			if err != nil {
				return nil, err
			}
			out[k] = cv
		}
		return map[string]any{"dict": out}, nil

	case has(m, "functionDefinitionValue"):
		var fd struct {
			ArgumentNames []string        `json:"argumentNames"`
			Body          json.RawMessage `json:"body"`
		}
		if err := json.Unmarshal(m["functionDefinitionValue"], &fd); err != nil {
			return nil, err
		}
		// The body is a reference key, held as a bare JSON string.
		var key string
		if err := json.Unmarshal(fd.Body, &key); err != nil {
			return nil, fmt.Errorf("canonicalise: functionDefinitionValue body is not a string reference: %w", err)
		}
		target, ok := c.values[key]
		if !ok {
			return nil, fmt.Errorf("canonicalise: dangling function body reference %q", key)
		}
		// Bind the parameters to positional placeholders for the body walk.
		bindings := make(map[string]string, len(fd.ArgumentNames))
		placeholders := make([]string, len(fd.ArgumentNames))
		for i, n := range fd.ArgumentNames {
			ph := fmt.Sprintf("$%d_%d", c.depth, i)
			bindings[n] = ph
			placeholders[i] = ph
		}
		c.scope = &scope{names: bindings, prev: c.scope}
		c.depth++
		body, err := c.node(target)
		c.depth--
		c.scope = c.scope.prev
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"funcdef": map[string]any{"args": placeholders, "body": body},
		}, nil

	case has(m, "functionInvocationValue"):
		var fi struct {
			FunctionName      string                     `json:"functionName"`
			FunctionReference string                     `json:"functionReference"`
			Arguments         map[string]json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(m["functionInvocationValue"], &fi); err != nil {
			return nil, err
		}
		args := map[string]any{}
		for k, v := range fi.Arguments {
			cv, err := c.node(v)
			if err != nil {
				return nil, err
			}
			args[k] = cv
		}
		name := fi.FunctionName
		if name == "" && fi.FunctionReference != "" {
			target, ok := c.values[fi.FunctionReference]
			if !ok {
				return nil, fmt.Errorf("canonicalise: dangling functionReference %q", fi.FunctionReference)
			}
			ref, err := c.node(target)
			if err != nil {
				return nil, err
			}
			return map[string]any{"invokeRef": map[string]any{"fn": ref, "args": args}}, nil
		}
		return map[string]any{"invoke": map[string]any{"fn": name, "args": args}}, nil

	case has(m, "bytesValue"):
		var s string
		if err := json.Unmarshal(m["bytesValue"], &s); err != nil {
			return nil, err
		}
		return map[string]any{"bytes": s}, nil
	}

	return nil, fmt.Errorf("canonicalise: unrecognised value node with keys %v", keysOf(m))
}

func has(m map[string]json.RawMessage, k string) bool {
	_, ok := m[k]
	return ok
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// normaliseNumbers pushes every JSON number through float64 so that a value
// written as 20 and one written as 20.0 compare equal.
func normaliseNumbers(v any) any {
	switch t := v.(type) {
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = normaliseNumbers(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = normaliseNumbers(e)
		}
		return out
	default:
		return v
	}
}
