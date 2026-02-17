// structs.go resolves STRUCT(typeName) references in wrekenfile INPUTS using the STRUCTS section.
// When writing to tooling.json, composite types like STRUCT(app.invite.createRequest) are expanded
// to their scalar (and nested object) schema so tooling.json contains the full resolved shape.
package commands

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadWrekenfile reads and parses a wrekenfile.yaml into a map.
func LoadWrekenfile(wrekenPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(wrekenPath)
	if err != nil {
		return nil, fmt.Errorf("read wrekenfile: %w", err)
	}
	var wreken map[string]interface{}
	if err := yaml.Unmarshal(data, &wreken); err != nil {
		return nil, fmt.Errorf("parse wrekenfile: %w", err)
	}
	return wreken, nil
}

// ResolveInputs resolves all STRUCT(...) types in method INPUTS using the wrekenfile's STRUCTS section.
// Returns the same structure as inputs but with STRUCT types expanded to OBJECT + schema (resolved to scalars).
func ResolveInputs(wreken map[string]interface{}, inputs interface{}) (interface{}, error) {
	if inputs == nil {
		return nil, nil
	}
	list, ok := inputs.([]interface{})
	if !ok {
		return inputs, nil
	}
	resolved := make([]interface{}, 0, len(list))
	for i, item := range list {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			resolved = append(resolved, item)
			continue
		}
		// Each list element is a single key (param name) -> param spec (TYPE, LOCATION, REQUIRED, ...)
		newItem := make(map[string]interface{})
		for paramName, paramSpec := range itemMap {
			specMap, ok := paramSpec.(map[string]interface{})
			if !ok {
				newItem[paramName] = paramSpec
				continue
			}
			typeStr := getString(specMap, "TYPE")
			if typeStr == "" {
				typeStr = getString(specMap, "type")
			}
			structName, isStruct := parseStructType(typeStr)
			if !isStruct {
				newItem[paramName] = paramSpec
				continue
			}
			visited := make(map[string]bool)
			schema, err := resolveStruct(wreken, structName, visited)
			if err != nil {
				return nil, fmt.Errorf("input[%d].%s: %w", i, paramName, err)
			}
			// Copy param spec and replace with resolved OBJECT + schema
			copied := copyMap(specMap)
			copied["TYPE"] = "OBJECT"
			copied["schema"] = schema
			newItem[paramName] = copied
		}
		resolved = append(resolved, newItem)
	}
	return resolved, nil
}

// ResolveReturns resolves RETURNS from the wrekenfile (same STRUCT/scalar pattern as inputs).
// Picks the first success (STATUS 200) or first entry, resolves RETURNTYPE (STRUCT → schema, scalars preserved),
// and returns an output object for tooling.json: { "status": 200, "schema": resolvedSchema } or nil if no RETURNS.
func ResolveReturns(wreken map[string]interface{}, returns interface{}) (interface{}, error) {
	if returns == nil {
		return nil, nil
	}
	list, ok := returns.([]interface{})
	if !ok || len(list) == 0 {
		return nil, nil
	}
	// Prefer first entry with STATUS 200; otherwise first entry
	var chosen map[string]interface{}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		status := getStatus(m)
		if status == 200 {
			chosen = m
			break
		}
		if chosen == nil {
			chosen = m
		}
	}
	if chosen == nil {
		return nil, nil
	}
	typeStr := getString(chosen, "RETURNTYPE")
	if typeStr == "" {
		typeStr = getString(chosen, "returntype")
	}
	if typeStr == "" {
		return nil, nil
	}
	visited := make(map[string]bool)
	schema, err := resolveReturnTypeToSchema(wreken, typeStr, visited)
	if err != nil {
		return nil, err
	}
	status := getStatus(chosen)
	return map[string]interface{}{
		"status": status,
		"schema": schema,
	},
		nil
}

func getStatus(m map[string]interface{}) int {
	if v, ok := m["STATUS"]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	if v, ok := m["status"]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

// resolveReturnTypeToSchema turns a type string (STRUCT(name), []ANY, STRING, etc.) into a schema map.
func resolveReturnTypeToSchema(wreken map[string]interface{}, typeStr string, visited map[string]bool) (map[string]interface{}, error) {
	typeStr = strings.TrimSpace(typeStr)
	structName, isStruct := parseStructType(typeStr)
	if isStruct {
		return resolveStruct(wreken, structName, visited)
	}
	// Scalar or array
	switch {
	case typeStr == "[]ANY" || strings.Trim(typeStr, "\"") == "[]ANY":
		return map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "any"}}, nil
	case typeStr == "[]STRING" || strings.Trim(typeStr, "\"") == "[]STRING":
		return map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}, nil
	default:
		return map[string]interface{}{"type": normalizeType(typeStr)}, nil
	}
}

var structTypeRe = regexp.MustCompile(`^STRUCT\((.+)\)$`)

func parseStructType(s string) (name string, ok bool) {
	s = strings.TrimSpace(s)
	m := structTypeRe.FindStringSubmatch(s)
	if len(m) != 2 {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

// resolveStruct returns a schema object for the struct: { properties: { name: { type, required } }, required: [...] }.
// Recursively resolves nested STRUCT(...) fields. Detects cycles.
func resolveStruct(wreken map[string]interface{}, structName string, visited map[string]bool) (map[string]interface{}, error) {
	if visited[structName] {
		return nil, fmt.Errorf("struct %q: cycle detected", structName)
	}
	visited[structName] = true
	defer func() { delete(visited, structName) }()

	structsSection, ok := wreken["STRUCTS"]
	if !ok {
		return nil, fmt.Errorf("struct %q: STRUCTS section not found in wrekenfile", structName)
	}
	structsMap, ok := structsSection.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("STRUCTS must be a map")
	}
	fieldsList, ok := structsMap[structName]
	if !ok {
		return nil, fmt.Errorf("struct %q not found in STRUCTS", structName)
	}
	fields, ok := fieldsList.([]interface{})
	if !ok {
		return nil, fmt.Errorf("struct %q: fields must be a list", structName)
	}

	properties := make(map[string]interface{})
	var required []string
	for _, f := range fields {
		fieldMap, ok := f.(map[string]interface{})
		if !ok {
			continue
		}
		fieldName := getString(fieldMap, "name")
		if fieldName == "" {
			continue
		}
		fieldType := getString(fieldMap, "type")
		if fieldType == "" {
			fieldType = getString(fieldMap, "TYPE")
		}
		req := getBool(fieldMap, "REQUIRED") || getBool(fieldMap, "required")

		if req {
			required = append(required, fieldName)
		}

		innerStructName, isStruct := parseStructType(fieldType)
		if isStruct {
			nested, err := resolveStruct(wreken, innerStructName, visited)
			if err != nil {
				return nil, fmt.Errorf("struct %q field %q: %w", structName, fieldName, err)
			}
			properties[fieldName] = map[string]interface{}{
				"type":     "object",
				"schema":  nested,
				"required": req,
			}
			continue
		}

		// Scalar or non-struct composite (OBJECT, []STRING, []ANY, etc.)
		prop := map[string]interface{}{
			"type":     normalizeType(fieldType),
			"required": req,
		}
		properties[fieldName] = prop
	}

	out := map[string]interface{}{
		"properties": properties,
	}
	if len(required) > 0 {
		out["required"] = required
	}
	return out, nil
}

func getString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getBool(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true")
	}
	return false
}

func normalizeType(t string) string {
	t = strings.TrimSpace(t)
	switch {
	case strings.EqualFold(t, "STRING"):
		return "string"
	case strings.EqualFold(t, "BOOL"):
		return "boolean"
	case strings.EqualFold(t, "ANY"):
		return "any"
	case strings.EqualFold(t, "OBJECT"):
		return "object"
	default:
		return t
	}
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
