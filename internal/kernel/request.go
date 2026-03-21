// request.go builds HTTP requests from method definitions and input args.
package kernel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// inputLocation returns the LOCATION value for an arg key from method.InputLocations (case-insensitive key match).
// Returns empty string if not defined.
func inputLocation(method *Method, key string) string {
	if method.InputLocations == nil {
		return ""
	}
	return method.InputLocations[strings.ToLower(key)]
}

// queryParamReservedKeys are top-level args that must not be sent as URL query parameters.
var queryParamReservedKeys = map[string]bool{
	"body": true, "params": true, "Authorization": true, "headers": true,
}

// paramsFromArgs returns a normalized map[string]string from args["params"].
// JSON stdin produces map[string]interface{}; CLI may produce map[string]string. Both are supported.
func paramsFromArgs(args map[string]interface{}) map[string]string {
	if args == nil {
		return nil
	}
	raw, ok := args["params"]
	if !ok || raw == nil {
		return nil
	}
	out := make(map[string]string)
	switch m := raw.(type) {
	case map[string]string:
		return m
	case map[string]interface{}:
		for k, v := range m {
			out[k] = argValueToQueryString(v)
		}
		return out
	}
	return nil
}

// BuildRequest builds an HTTP request from method definition, base URL, and input args.
// baseURL is the manifest endpoint (sandbox or production) and is prepended to the method path.
//
// GET: path params from args["params"] (e.g. /api/{id}); query from args["params"] + all other top-level args
// (e.g. project_name, filter). No body.
//
// POST/PUT/PATCH: same URL and query as above; body from args["body"] (JSON); Content-Type application/json
// if body is set and not already set by method headers.
//
// Headers: (1) method.Headers from Wreken, (2) args["Authorization"], (3) args["headers"] map (any extra headers).
func BuildRequest(method *Method, baseURL string, args map[string]interface{}) (*http.Request, error) {
	// Prepend base URL to method path (normalize slashes)
	base := strings.TrimSuffix(baseURL, "/")
	path := method.Endpoint
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	fullURL := base + path
	params := paramsFromArgs(args)

	// Replace path parameters in endpoint (e.g., /api/cluster/{id} -> /api/cluster/123)
	// Values are URL-encoded to prevent path injection. Keys used in path must not
	// be added to query (path takes precedence).
	for key, value := range params {
		placeholder := "{" + key + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, url.PathEscape(value))
			fullURL = base + path
		}
	}
	// Also substitute path params from top-level args (e.g., merged from prior workflow step outputs).
	// This allows chained steps to pass values like project_uuid into path parameters.
	for key, val := range args {
		placeholder := "{" + key + "}"
		if strings.Contains(path, placeholder) {
			str := argValueToQueryString(val)
			if str != "" {
				path = strings.ReplaceAll(path, placeholder, url.PathEscape(str))
				fullURL = base + path
			}
		}
	}

	// Parse URL
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Query: merge (a) params (keys not used in path) and (b) other top-level args. Prefer params when both exist.
	query := parsedURL.Query()
	for key, value := range params {
		placeholder := "{" + key + "}"
		if !strings.Contains(method.Endpoint, placeholder) && value != "" {
			query.Set(key, value)
		}
	}
	for key, val := range args {
		if queryParamReservedKeys[key] {
			continue
		}
		// Skip if LOCATION is explicitly non-query (e.g. header or body)
		if loc := inputLocation(method, key); loc != "" && loc != "query" {
			continue
		}
		if query.Get(key) != "" {
			continue // params already set this; prefer params
		}
		str := argValueToQueryString(val)
		if str != "" {
			query.Set(key, str)
		}
	}
	parsedURL.RawQuery = query.Encode()

	// Prepare body
	var bodyReader *strings.Reader
	if bodyRaw, ok := args["body"]; ok {
		// Body is provided as JSON object
		bodyJSON, err := json.Marshal(bodyRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(bodyJSON))
	}

	// Create HTTP request
	var req *http.Request
	if bodyReader != nil {
		req, err = http.NewRequest(method.HTTPMethod, parsedURL.String(), bodyReader)
	} else {
		req, err = http.NewRequest(method.HTTPMethod, parsedURL.String(), nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers from method definition (Wreken HEADERS)
	for key, value := range method.Headers {
		req.Header.Set(key, value)
	}

	// Route args declared as LOCATION: header — skip empty strings (don't send blank headers)
	for key, val := range args {
		if inputLocation(method, key) == "header" {
			if str := argValueToQueryString(val); str != "" {
				req.Header.Set(key, str)
			}
		}
	}

	// Headers from args: Authorization (single) and optional args["headers"] map
	if authRaw, ok := args["Authorization"]; ok {
		if authStr, ok := authRaw.(string); ok {
			req.Header.Set("Authorization", authStr)
		}
	}
	if headersMap, ok := args["headers"].(map[string]interface{}); ok {
		for key, val := range headersMap {
			if str := argValueToQueryString(val); str != "" {
				req.Header.Set(key, str)
			}
		}
	}
	// Also support args["headers"] as map[string]string (e.g. from JSON)
	if headersMap, ok := args["headers"].(map[string]string); ok {
		for key, val := range headersMap {
			if val != "" {
				req.Header.Set(key, val)
			}
		}
	}

	// Set Content-Type if body is present
	if bodyReader != nil {
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	return req, nil
}

// argValueToQueryString converts an arg value to a query string value. Returns empty string to skip.
func argValueToQueryString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		// Slice/map: encode as JSON so ?filter={"a":1} works
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
