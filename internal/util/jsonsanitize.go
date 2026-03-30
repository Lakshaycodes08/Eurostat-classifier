// jsonsanitize.go replaces JSON-incompatible float values in structures loaded from YAML/JSON.
package util

import (
	"math"
	"reflect"
)

// SanitizeForJSON returns a deep copy of v with float64 NaN and Inf replaced by nil so
// encoding/json can marshal the result.
func SanitizeForJSON(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return nil
		}
		return x
	case float32:
		f := float64(x)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
		return x
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, el := range x {
			out[i] = SanitizeForJSON(el)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, el := range x {
			out[k] = SanitizeForJSON(el)
		}
		return out
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice:
			if rv.Type().Elem().Kind() == reflect.Uint8 {
				return v
			}
			n := rv.Len()
			out := make([]interface{}, n)
			for i := 0; i < n; i++ {
				out[i] = SanitizeForJSON(rv.Index(i).Interface())
			}
			return out
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String {
				return v
			}
			out := make(map[string]interface{})
			for _, key := range rv.MapKeys() {
				out[key.String()] = SanitizeForJSON(rv.MapIndex(key).Interface())
			}
			return out
		default:
			return v
		}
	}
}
