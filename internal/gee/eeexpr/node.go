// Package eeexpr builds Earth Engine expression-graph JSON by hand.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §5.
//
// There is no Go Earth Engine SDK — Google ships JavaScript and Python clients
// only. The Python `ee` package is not an HTTP client but a lazy
// computation-graph builder: every `ee.ImageCollection(...).filterBounds(...)`
// call constructs an in-memory DAG, and only `.getInfo()` / `.getMapId()`
// serialises it and POSTs it. This package is the Go equivalent of that
// serialiser.
//
// Every function name and argument name emitted here was captured from the real
// Python client (see testdata/, produced by legacy-python/tools/dump_graphs.py)
// rather than guessed. A guessed name produces a runtime 400 from Google, not a
// compile error, which is why the fixture tests are the acceptance gate for the
// whole package (§5.4).
package eeexpr

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Node is a lazily-built Earth Engine value. Nodes are immutable: every helper
// returns a new Node. Nothing is serialised until Compile is called.
type Node interface{ isNode() }

type constNode struct{ v any }

// argRefNode is a reference to a function-definition parameter. Unlike every
// other node it must NOT be inlined or interned — it only has meaning inside
// the body of the functionDefinitionValue that declares it.
type argRefNode struct{ name string }

type invokeNode struct {
	fn   string
	args map[string]Node
}

type funcDefNode struct {
	argNames []string
	body     Node
}

// arrayNode is a JSON array whose elements are themselves nodes. A list of pure
// constants is emitted as a single constantValue instead, matching the Python
// serialiser.
type arrayNode struct{ items []Node }

type dictNode struct{ items map[string]Node }

func (constNode) isNode()   {}
func (argRefNode) isNode()  {}
func (invokeNode) isNode()  {}
func (funcDefNode) isNode() {}
func (arrayNode) isNode()   {}
func (dictNode) isNode()    {}

// Const wraps any JSON-encodable value as a constantValue.
func Const(v any) Node { return constNode{v} }

// Array builds an arrayValue from element nodes.
func Array(items ...Node) Node { return arrayNode{items: items} }

// Dict builds a dictionaryValue.
func Dict(m map[string]Node) Node { return dictNode{items: m} }

// ArgRef references a function-definition parameter by name.
func ArgRef(name string) Node { return argRefNode{name} }

// FuncDef builds a functionDefinitionValue. Its body is always hoisted into the
// top-level `values` map at compile time, because the wire format stores the
// body as a string reference rather than an inline node.
func FuncDef(argNames []string, body Node) Node {
	return funcDefNode{argNames: argNames, body: body}
}

// Invoke builds a functionInvocationValue. Nil arguments are dropped, matching
// the Python client, which omits unset optional parameters entirely rather than
// sending explicit nulls.
func Invoke(fn string, args map[string]Node) Node {
	clean := make(map[string]Node, len(args))
	for k, v := range args {
		if v != nil {
			clean[k] = v
		}
	}
	return invokeNode{fn: fn, args: clean}
}

// Expression is the wire format posted to Earth Engine.
type Expression struct {
	Values map[string]json.RawMessage `json:"values"`
	Result string                     `json:"result"`
}

// Compile walks the DAG and produces the wire format.
//
// Three kinds of node are hoisted into `values` and referred to by
// valueReference; everything else is inlined:
//
//   - the root, because `result` must name a key;
//   - every functionDefinitionValue body, because the format requires the body
//     to be a string reference;
//   - any subtree that appears more than once, which is what the Python
//     serialiser does and which keeps large graphs (a 2000-cell grid, a 21-stop
//     palette) from ballooning.
//
// Reference keys are assigned "0", "1", … deterministically. The exact key
// numbering is NOT part of the contract — Earth Engine only requires the
// references to resolve — so the fixture tests compare canonicalised trees
// rather than raw key assignments (§5.4).
func Compile(root Node) (*Expression, error) {
	if root == nil {
		return nil, fmt.Errorf("eeexpr: cannot compile a nil root")
	}

	c := &compiler{
		counts: map[string]int{},
		keyOf:  map[string]string{},
		values: map[string]json.RawMessage{},
	}
	c.count(root)

	// The root always gets a key, whatever its reference count.
	c.mustHoist = map[string]bool{c.fingerprint(root): true}
	c.collectFuncBodies(root)

	rootKey, err := c.hoist(root)
	if err != nil {
		return nil, err
	}
	return &Expression{Values: c.values, Result: rootKey}, nil
}

type compiler struct {
	counts    map[string]int
	keyOf     map[string]string
	values    map[string]json.RawMessage
	mustHoist map[string]bool
	next      int
}

func (c *compiler) count(n Node) {
	switch t := n.(type) {
	case argRefNode:
		return // never interned: only meaningful inside its own funcdef
	case invokeNode:
		c.counts[c.fingerprint(n)]++
		for _, k := range sortedKeys(t.args) {
			c.count(t.args[k])
		}
	case funcDefNode:
		c.counts[c.fingerprint(n)]++
		c.count(t.body)
	case arrayNode:
		c.counts[c.fingerprint(n)]++
		for _, it := range t.items {
			c.count(it)
		}
	case dictNode:
		c.counts[c.fingerprint(n)]++
		for _, k := range sortedKeys(t.items) {
			c.count(t.items[k])
		}
	default:
		c.counts[c.fingerprint(n)]++
	}
}

func (c *compiler) collectFuncBodies(n Node) {
	switch t := n.(type) {
	case invokeNode:
		for _, k := range sortedKeys(t.args) {
			c.collectFuncBodies(t.args[k])
		}
	case funcDefNode:
		c.mustHoist[c.fingerprint(t.body)] = true
		c.collectFuncBodies(t.body)
	case arrayNode:
		for _, it := range t.items {
			c.collectFuncBodies(it)
		}
	case dictNode:
		for _, k := range sortedKeys(t.items) {
			c.collectFuncBodies(t.items[k])
		}
	}
}

func (c *compiler) shouldHoist(n Node) bool {
	if _, isArg := n.(argRefNode); isArg {
		return false
	}
	fp := c.fingerprint(n)
	return c.mustHoist[fp] || c.counts[fp] > 1
}

// hoist emits n into values (if not already there) and returns its key.
func (c *compiler) hoist(n Node) (string, error) {
	fp := c.fingerprint(n)
	if k, ok := c.keyOf[fp]; ok {
		return k, nil
	}
	key := strconv.Itoa(c.next)
	c.next++
	c.keyOf[fp] = key
	// Reserve the slot before rendering so a self-referential walk terminates.
	c.values[key] = nil

	raw, err := c.render(n, true)
	if err != nil {
		return "", err
	}
	c.values[key] = raw
	return key, nil
}

// render produces the JSON for a node. When atTop is true the node is being
// written into the values map, so it is rendered in full even if it would
// otherwise be hoisted.
func (c *compiler) render(n Node, atTop bool) (json.RawMessage, error) {
	if !atTop && c.shouldHoist(n) {
		key, err := c.hoist(n)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{"valueReference": key})
	}

	switch t := n.(type) {
	case constNode:
		return json.Marshal(map[string]any{"constantValue": t.v})

	case argRefNode:
		return json.Marshal(map[string]any{"argumentReference": t.name})

	case invokeNode:
		args := map[string]json.RawMessage{}
		for _, k := range sortedKeys(t.args) {
			raw, err := c.render(t.args[k], false)
			if err != nil {
				return nil, err
			}
			args[k] = raw
		}
		return json.Marshal(map[string]any{
			"functionInvocationValue": map[string]any{
				"functionName": t.fn,
				"arguments":    args,
			},
		})

	case funcDefNode:
		bodyKey, err := c.hoist(t.body)
		if err != nil {
			return nil, err
		}
		return json.Marshal(map[string]any{
			"functionDefinitionValue": map[string]any{
				"argumentNames": t.argNames,
				"body":          bodyKey,
			},
		})

	case arrayNode:
		items := make([]json.RawMessage, len(t.items))
		for i, it := range t.items {
			raw, err := c.render(it, false)
			if err != nil {
				return nil, err
			}
			items[i] = raw
		}
		return json.Marshal(map[string]any{
			"arrayValue": map[string]any{"values": items},
		})

	case dictNode:
		items := map[string]json.RawMessage{}
		for _, k := range sortedKeys(t.items) {
			raw, err := c.render(t.items[k], false)
			if err != nil {
				return nil, err
			}
			items[k] = raw
		}
		return json.Marshal(map[string]any{
			"dictionaryValue": map[string]any{"values": items},
		})

	default:
		return nil, fmt.Errorf("eeexpr: unknown node type %T", n)
	}
}

// fingerprint is a structural identity for a subtree: two nodes with the same
// fingerprint are interchangeable and may share a reference.
func (c *compiler) fingerprint(n Node) string {
	var sb strings.Builder
	writeFingerprint(&sb, n)
	return sb.String()
}

func writeFingerprint(sb *strings.Builder, n Node) {
	switch t := n.(type) {
	case constNode:
		b, _ := json.Marshal(t.v)
		sb.WriteString("c(")
		sb.Write(b)
		sb.WriteString(")")
	case argRefNode:
		sb.WriteString("a(" + t.name + ")")
	case invokeNode:
		sb.WriteString("i(" + t.fn)
		for _, k := range sortedKeys(t.args) {
			sb.WriteString("," + k + ":")
			writeFingerprint(sb, t.args[k])
		}
		sb.WriteString(")")
	case funcDefNode:
		sb.WriteString("f(" + strings.Join(t.argNames, "|") + ":")
		writeFingerprint(sb, t.body)
		sb.WriteString(")")
	case arrayNode:
		sb.WriteString("[")
		for _, it := range t.items {
			writeFingerprint(sb, it)
			sb.WriteString(",")
		}
		sb.WriteString("]")
	case dictNode:
		sb.WriteString("{")
		for _, k := range sortedKeys(t.items) {
			sb.WriteString(k + ":")
			writeFingerprint(sb, t.items[k])
			sb.WriteString(",")
		}
		sb.WriteString("}")
	default:
		sb.WriteString("?")
	}
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// MarshalJSON renders the expression with its values map intact.
func (e *Expression) MarshalJSON() ([]byte, error) {
	type alias Expression
	return json.Marshal((*alias)(e))
}
