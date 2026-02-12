// request.go builds HTTP requests from method definitions and input args.
package kernel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// BuildRequest builds an HTTP request from method definition, base URL, and input args.
func BuildRequest(method *Method, baseURL string, args map[string]interface{}) (*http.Request, error) {
	// Construct full URL: baseURL + endpoint
	fullURL := baseURL + method.Endpoint

	// Replace path parameters in endpoint (e.g., /api/cluster/{id} -> /api/cluster/123)
	endpoint := method.Endpoint
	if params, ok := args["params"].(map[string]string); ok {
		for key, value := range params {
			placeholder := "{" + key + "}"
			if strings.Contains(endpoint, placeholder) {
				endpoint = strings.ReplaceAll(endpoint, placeholder, value)
				fullURL = baseURL + endpoint
			}
		}
	}

	// Parse URL
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Add query parameters
	if params, ok := args["params"].(map[string]string); ok {
		query := parsedURL.Query()
		for key, value := range params {
			// Skip path params that were already replaced
			placeholder := "{" + key + "}"
			if !strings.Contains(method.Endpoint, placeholder) {
				query.Set(key, value)
			}
		}
		parsedURL.RawQuery = query.Encode()
	}

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

	// Set headers from method definition
	for key, value := range method.Headers {
		req.Header.Set(key, value)
	}

	// Inject auth headers from args
	// Look for Authorization or other auth fields in args
	if authRaw, ok := args["Authorization"]; ok {
		if authStr, ok := authRaw.(string); ok {
			req.Header.Set("Authorization", authStr)
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
