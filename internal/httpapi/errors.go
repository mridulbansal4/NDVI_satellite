// errors.go — canonical error envelopes.
//
// PRD reference: PRAGYA_GO_MIGRATION_PRD.md §7.11, §6.2.
//
// Two envelope shapes exist in the frozen contract and must not be unified:
//
//	{"error": "<message>"}                     — ad-hoc route errors
//	{"errors": {"<field>": ["<message>"]}}     — marshmallow schema failures
//
// and a third from Flask-JWT-Extended:
//
//	{"msg": "<message>"}                       — auth middleware failures
//
// The status code that accompanies the marshmallow envelope differs by route:
// /auth/signup and /auth/login answer 422, every other onboarding route answers
// 400. That inconsistency is in the Python code and is deliberately preserved.

package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Err writes the {"error": ...} envelope.
func Err(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// Msg writes the {"msg": ...} envelope Flask-JWT-Extended uses.
func Msg(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"msg": msg})
}

// FieldErrors writes the marshmallow-shaped {"errors": {field: [msg]}} envelope.
func FieldErrors(c *gin.Context, status int, fields map[string][]string) {
	c.JSON(status, gin.H{"errors": fields})
}

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	// Report the JSON tag name, not the Go field name: the envelope keys are
	// mobile_number, preferred_language, … exactly as marshmallow emits them.
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	// mlen=<min>,<max> mirrors marshmallow validate.Length(min, max). It is a
	// single tag rather than validator's separate min/max so that one failure
	// yields marshmallow's single combined message.
	_ = v.RegisterValidation("mlen", func(fl validator.FieldLevel) bool {
		lo, hi, hasLo, hasHi := optionalTwoParams(fl.Param())
		f := deref(fl.Field())
		if f.Kind() != reflect.String {
			return true
		}
		n := float64(len([]rune(f.String())))
		if hasLo && n < lo {
			return false
		}
		if hasHi && n > hi {
			return false
		}
		return true
	})

	// mrange=<min>,<max> mirrors validate.Range(min, max). An empty bound means
	// unbounded on that side, e.g. mrange=0.01, for Range(min=0.01).
	_ = v.RegisterValidation("mrange", func(fl validator.FieldLevel) bool {
		lo, hi, hasLo, hasHi := optionalTwoParams(fl.Param())
		f := deref(fl.Field())
		var n float64
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n = float64(f.Int())
		case reflect.Float32, reflect.Float64:
			n = f.Float()
		default:
			return true
		}
		if hasLo && n < lo {
			return false
		}
		if hasHi && n > hi {
			return false
		}
		return true
	})

	return v
}

// BindAndValidate decodes JSON into dst, runs validation, and on failure writes
// the marshmallow-shaped envelope with the given status. It reports whether the
// request may proceed.
//
// Parity notes:
//   - A body that is absent or unparseable is treated as {}, matching Flask's
//     `request.get_json(silent=True) or {}` and `get_json(force=True) or {}`.
//     The resulting errors are therefore "Missing data for required field."
//     per required field, never a JSON-parse error.
//   - Pointer fields model marshmallow's load_default=None, so a field that is
//     absent stays nil and serialises back as JSON null.
func BindAndValidate(c *gin.Context, dst any, status int) bool {
	raw, _ := io.ReadAll(c.Request.Body)
	if len(strings.TrimSpace(string(raw))) > 0 {
		dec := json.NewDecoder(strings.NewReader(string(raw)))
		if err := dec.Decode(dst); err != nil {
			// A type mismatch (e.g. a string where a float is expected) is a
			// marshmallow "Not a valid …" error, keyed on the offending field.
			if fe := typeErrorToField(err); fe != nil {
				FieldErrors(c, status, fe)
				return false
			}
			// Anything else (malformed JSON, wrong top-level type) is treated
			// as an empty body, exactly as Flask's `... or {}` does. Validation
			// then reports the required fields as missing.
		}
	}

	if err := validate.Struct(dst); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			FieldErrors(c, status, translate(ve, dst))
			return false
		}
		FieldErrors(c, status, map[string][]string{"_schema": {err.Error()}})
		return false
	}
	return true
}

// translate converts validator failures into marshmallow's exact wording.
// Messages are registered per (field, tag); validator's own English is never
// emitted.
func translate(ve validator.ValidationErrors, dst any) map[string][]string {
	out := map[string][]string{}
	for _, fe := range ve {
		field := fe.Field()
		msg := messageFor(dst, field, fe.Tag(), fe.Param())
		out[field] = append(out[field], msg)
	}
	// Deterministic ordering when a field somehow collects several messages.
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

func messageFor(dst any, field, tag, param string) string {
	// A schema may override any (field, tag) pair with marshmallow's custom
	// `error=` text, e.g. "Mobile number must be exactly 10 digits."
	if m, ok := dst.(interface {
		Message(field, tag string) string
	}); ok {
		if s := m.Message(field, tag); s != "" {
			return s
		}
	}

	switch tag {
	case "required":
		return "Missing data for required field."
	case "oneof":
		return "Must be one of: " + strings.Join(strings.Fields(param), ", ") + "."
	case "mlen":
		lo, hi, ok := twoParams(param)
		if !ok {
			return "Invalid length."
		}
		return fmt.Sprintf("Length must be between %s and %s.", num(lo), num(hi))
	case "mrange":
		lo, hi, hasLo, hasHi := optionalTwoParams(param)
		switch {
		case hasLo && hasHi:
			return fmt.Sprintf(
				"Must be greater than or equal to %s and less than or equal to %s.",
				num(lo), num(hi))
		case hasLo:
			return fmt.Sprintf("Must be greater than or equal to %s.", num(lo))
		case hasHi:
			return fmt.Sprintf("Must be less than or equal to %s.", num(hi))
		}
		return "Invalid value."
	case "uuid", "uuid4":
		return "Not a valid UUID."
	case "datetime":
		return "Not a valid date."
	default:
		return "Invalid value."
	}
}

// num renders a bound the way marshmallow interpolates it: 255, 0.01, -90.
func num(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// deref unwraps a pointer field so mlen/mrange see the underlying value.
// Required fields are modelled as pointers (nil == JSON key absent) so that
// "present but empty" produces marshmallow's length/format error rather than
// "Missing data for required field."; the built-in validations dereference on
// their own, but custom ones do not.
func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return v
		}
		v = v.Elem()
	}
	return v
}

// Bounds inside mlen/mrange params are separated by ':' rather than ','.
// validator splits the whole tag string on commas before a custom validation
// ever sees it, so "mlen=1,255" would be parsed as the tag `mlen=1` followed by
// an undefined tag `255` — which panics at struct-cache build time.
const boundSep = ":"

// twoParams parses a required "<lo>:<hi>" pair, as used by mlen.
func twoParams(p string) (lo, hi float64, ok bool) {
	lo, hi, hasLo, hasHi := optionalTwoParams(p)
	if !hasLo || !hasHi {
		return 0, 0, false
	}
	return lo, hi, true
}

// optionalTwoParams parses "<lo>:<hi>" where either side may be empty, as used
// by mrange to express marshmallow's one-sided Range(min=0.01).
func optionalTwoParams(p string) (lo, hi float64, hasLo, hasHi bool) {
	parts := strings.SplitN(p, boundSep, 2)
	if s := strings.TrimSpace(parts[0]); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			lo, hasLo = v, true
		}
	}
	if len(parts) == 2 {
		if s := strings.TrimSpace(parts[1]); s != "" {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				hi, hasHi = v, true
			}
		}
	}
	return lo, hi, hasLo, hasHi
}

// typeErrorToField maps a json.UnmarshalTypeError onto the marshmallow message
// for a field of the wrong type.
func typeErrorToField(err error) map[string][]string {
	var ute *json.UnmarshalTypeError
	if !errors.As(err, &ute) || ute.Field == "" {
		return nil
	}
	field := ute.Field
	if i := strings.LastIndexByte(field, '.'); i >= 0 {
		field = field[i+1:]
	}
	var msg string
	switch ute.Type.Kind() {
	case reflect.Float32, reflect.Float64:
		msg = "Not a valid number."
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		msg = "Not a valid integer."
	case reflect.Bool:
		msg = "Not a valid boolean."
	case reflect.String:
		msg = "Not a valid string."
	default:
		msg = "Invalid value."
	}
	return map[string][]string{field: {msg}}
}
