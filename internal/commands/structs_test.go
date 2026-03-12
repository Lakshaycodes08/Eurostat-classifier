package commands

import (
	"strings"
	"testing"
)

func TestResolveStruct_NestedStructUsesPropertiesNotSchema(t *testing.T) {
	// Outer has field "inner" of type STRUCT(Inner). Inner has "name": STRING.
	// Resolved output must have properties.inner.type="object", properties.inner.properties.name.type="string"
	// with no "schema" key at any level.
	wreken := map[string]interface{}{
		"STRUCTS": map[string]interface{}{
			"Inner": []interface{}{
				map[string]interface{}{"NAME": "name", "TYPE": "STRING", "REQUIRED": true},
			},
			"Outer": []interface{}{
				map[string]interface{}{"name": "inner", "type": "STRUCT(Inner)", "REQUIRED": false},
			},
		},
	}
	visited := make(map[string]bool)
	out, err := resolveStruct(wreken, "Outer", visited)
	if err != nil {
		t.Fatalf("resolveStruct: %v", err)
	}
	props, _ := out["properties"].(map[string]interface{})
	if props == nil {
		t.Fatal("expected properties")
	}
	innerNode, _ := props["inner"].(map[string]interface{})
	if innerNode == nil {
		t.Fatal("expected properties.inner")
	}
	if innerNode["type"] != "object" {
		t.Errorf("inner type: got %v", innerNode["type"])
	}
	// Must have "properties" for nested object, not "schema"
	if _, hasSchema := innerNode["schema"]; hasSchema {
		t.Error("nested object must not have 'schema' key; use 'properties' only")
	}
	innerProps, _ := innerNode["properties"].(map[string]interface{})
	if innerProps == nil {
		t.Fatal("expected properties.inner.properties")
	}
	nameNode, _ := innerProps["name"].(map[string]interface{})
	if nameNode == nil {
		t.Fatal("expected properties.inner.properties.name")
	}
	if nameNode["type"] != "string" {
		t.Errorf("name type: got %v", nameNode["type"])
	}
}

func TestResolveStruct_ArrayOfStruct(t *testing.T) {
	wreken := map[string]interface{}{
		"STRUCTS": map[string]interface{}{
			"Item": []interface{}{
				map[string]interface{}{"NAME": "id", "TYPE": "STRING", "REQUIRED": true},
			},
			"Container": []interface{}{
				map[string]interface{}{"NAME": "items", "TYPE": "[]STRUCT(Item)", "REQUIRED": false},
			},
		},
	}
	visited := make(map[string]bool)
	out, err := resolveStruct(wreken, "Container", visited)
	if err != nil {
		t.Fatalf("resolveStruct: %v", err)
	}
	props, _ := out["properties"].(map[string]interface{})
	if props == nil {
		t.Fatal("expected properties")
	}
	itemsNode, _ := props["items"].(map[string]interface{})
	if itemsNode == nil {
		t.Fatal("expected properties.items")
	}
	if itemsNode["type"] != "array" {
		t.Errorf("items type: got %v", itemsNode["type"])
	}
	itemsSchema, _ := itemsNode["items"].(map[string]interface{})
	if itemsSchema == nil {
		t.Fatal("expected properties.items.items")
	}
	if itemsSchema["type"] != "object" {
		t.Errorf("items.items type: got %v", itemsSchema["type"])
	}
	itemProps, _ := itemsSchema["properties"].(map[string]interface{})
	if itemProps == nil {
		t.Fatal("expected properties.items.items.properties")
	}
	idNode, _ := itemProps["id"].(map[string]interface{})
	if idNode == nil {
		t.Fatal("expected properties.items.items.properties.id")
	}
	if idNode["type"] != "string" {
		t.Errorf("id type: got %v", idNode["type"])
	}
}

func TestResolveStruct_CycleDetection(t *testing.T) {
	// A has field b: STRUCT(B), B has field a: STRUCT(A)
	wreken := map[string]interface{}{
		"STRUCTS": map[string]interface{}{
			"A": []interface{}{
				map[string]interface{}{"NAME": "b", "TYPE": "STRUCT(B)", "REQUIRED": false},
			},
			"B": []interface{}{
				map[string]interface{}{"NAME": "a", "TYPE": "STRUCT(A)", "REQUIRED": false},
			},
		},
	}
	visited := make(map[string]bool)
	_, err := resolveStruct(wreken, "A", visited)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error should mention cycle: %v", err)
	}
}

func TestParseStructType(t *testing.T) {
	name, ok := parseStructType("STRUCT(Foo)")
	if !ok || name != "Foo" {
		t.Errorf("parseStructType(STRUCT(Foo)): name=%q ok=%v", name, ok)
	}
	name, ok = parseStructType(`"STRUCT(Bar)"`)
	if !ok || name != "Bar" {
		t.Errorf("parseStructType quoted: name=%q ok=%v", name, ok)
	}
	_, ok = parseStructType("[]STRUCT(Foo)")
	if ok {
		t.Error("parseStructType should not match []STRUCT")
	}
}

func TestParseArrayStructType(t *testing.T) {
	name, ok := parseArrayStructType("[]STRUCT(Item)")
	if !ok || name != "Item" {
		t.Errorf("parseArrayStructType: name=%q ok=%v", name, ok)
	}
	name, ok = parseArrayStructType(`"[]STRUCT(ListItem)"`)
	if !ok || name != "ListItem" {
		t.Errorf("parseArrayStructType quoted: name=%q ok=%v", name, ok)
	}
	_, ok = parseArrayStructType("STRUCT(Foo)")
	if ok {
		t.Error("parseArrayStructType should not match STRUCT")
	}
}

func TestNormalizeType(t *testing.T) {
	if g := normalizeType("INT"); g != "integer" {
		t.Errorf("INT: got %q", g)
	}
	if g := normalizeType("INT64"); g != "integer" {
		t.Errorf("INT64: got %q", g)
	}
	if g := normalizeType("STRING"); g != "string" {
		t.Errorf("STRING: got %q", g)
	}
	if g := normalizeType("ANY"); g != "any" {
		t.Errorf("ANY: got %q", g)
	}
}

func TestResolveReturnTypeToSchema_ArrayOfStruct(t *testing.T) {
	wreken := map[string]interface{}{
		"STRUCTS": map[string]interface{}{
			"Item": []interface{}{
				map[string]interface{}{"NAME": "x", "TYPE": "STRING", "REQUIRED": false},
			},
		},
	}
	visited := make(map[string]bool)
	out, err := resolveReturnTypeToSchema(wreken, "[]STRUCT(Item)", visited)
	if err != nil {
		t.Fatalf("resolveReturnTypeToSchema: %v", err)
	}
	if out["type"] != "array" {
		t.Errorf("type: got %v", out["type"])
	}
	items, _ := out["items"].(map[string]interface{})
	if items == nil || items["type"] != "object" {
		t.Errorf("items: got %v", out["items"])
	}
	props, _ := items["properties"].(map[string]interface{})
	if props == nil || props["x"] == nil {
		t.Errorf("items.properties: got %v", items["properties"])
	}
}

