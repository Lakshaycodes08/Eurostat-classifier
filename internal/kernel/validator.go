// validator.go validates input against tool schema.
package kernel

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// ValidateInput validates the provided args against the tool's input schema.
// Returns an error with a structured message if required fields are missing,
// types don't match, or enum constraints are violated.
func ValidateInput(tool *Tool, args map[string]interface{}) error {
	if tool.Inputs == nil {
		return nil
	}

	inputs, ok := tool.Inputs.([]interface{})
	if !ok {
		return nil // not array format, skip
	}

	var errs []string

	for _, raw := range inputs {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := item["name"].(string)
		if name == "" {
			continue
		}

		required, _ := item["required"].(bool)
		expectedType, _ := item["type"].(string)
		val, present := args[name]

		// Check required
		if required && (!present || val == nil) {
			errs = append(errs, fmt.Sprintf("missing required field %q", name))
			continue
		}

		if !present || val == nil {
			continue
		}

		// Check type
		if expectedType != "" {
			if typeErr := checkType(name, val, expectedType); typeErr != "" {
				errs = append(errs, typeErr)
				continue
			}
		}

		// Check enum
		if enumRaw, ok := item["enum"].([]interface{}); ok && len(enumRaw) > 0 {
			if enumErr := checkEnum(name, val, enumRaw); enumErr != "" {
				errs = append(errs, enumErr)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("input validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

func checkType(name string, val interface{}, expected string) string {
	switch expected {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Sprintf("field %q must be a string, got %T", name, val)
		}
	case "int", "integer":
		switch v := val.(type) {
		case int, int8, int16, int32, int64:
			// native integer types — fine
		case float64:
			if v != math.Trunc(v) {
				return fmt.Sprintf("field %q must be an integer, got fractional number %v", name, v)
			}
		default:
			return fmt.Sprintf("field %q must be an integer, got %T", name, val)
		}
	case "float", "number":
		switch val.(type) {
		case float32, float64, int, int8, int16, int32, int64:
		default:
			return fmt.Sprintf("field %q must be a number, got %T", name, val)
		}
	case "bool", "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Sprintf("field %q must be a boolean, got %T", name, val)
		}
	case "object":
		if _, ok := val.(map[string]interface{}); !ok {
			return fmt.Sprintf("field %q must be an object, got %T", name, val)
		}
	case "array":
		if _, ok := val.([]interface{}); !ok {
			return fmt.Sprintf("field %q must be an array, got %T", name, val)
		}
	}
	return ""
}

func checkEnum(name string, val interface{}, allowed []interface{}) string {
	for _, a := range allowed {
		if reflect.DeepEqual(val, a) {
			return ""
		}
	}
	var strs []string
	for _, a := range allowed {
		strs = append(strs, fmt.Sprintf("%q", a))
	}
	return fmt.Sprintf("field %q must be one of [%s], got %v", name, strings.Join(strs, ", "), val)
}
