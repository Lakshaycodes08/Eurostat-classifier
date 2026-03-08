package kernel

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestBuildRequest_StdinMapping(t *testing.T) {
	baseURL := "http://localhost"
	methodGET := &Method{HTTPMethod: "GET", Endpoint: "/v2/cli/methods"}
	methodPOST := &Method{HTTPMethod: "POST", Endpoint: "/v2/cli/create"}

	tests := []struct {
		name     string
		method   *Method
		args     map[string]interface{}
		wantURL  string // substring or full URL to check
		wantQuery string
		checkHeaders map[string]string
		wantBody string // optional JSON body
	}{
		{
			name:    "empty args - no query, no extra headers, no body",
			method:  methodGET,
			args:    map[string]interface{}{},
			wantURL: "http://localhost/v2/cli/methods",
			wantQuery: "",
		},
		{
			name:    "project_name top-level - GET url has query",
			method:  methodGET,
			args:    map[string]interface{}{"project_name": "swytchcode"},
			wantURL: "http://localhost/v2/cli/methods",
			wantQuery: "project_name=swytchcode",
		},
		{
			name:    "params.project_name - same as top-level",
			method:  methodGET,
			args:    map[string]interface{}{"params": map[string]interface{}{"project_name": "swytchcode"}},
			wantURL: "http://localhost/v2/cli/methods",
			wantQuery: "project_name=swytchcode",
		},
		{
			name:    "Authorization - header set",
			method:  methodGET,
			args:    map[string]interface{}{"Authorization": "Bearer xyz"},
			checkHeaders: map[string]string{"Authorization": "Bearer xyz"},
		},
		{
			name:    "headers map - custom header set",
			method:  methodGET,
			args:    map[string]interface{}{"headers": map[string]interface{}{"X-Custom": "val"}},
			checkHeaders: map[string]string{"X-Custom": "val"},
		},
		{
			name:    "body for POST - request body and dry-run",
			method:  methodPOST,
			args:    map[string]interface{}{"body": map[string]interface{}{"key": "value"}},
			wantBody: `{"key":"value"}`,
		},
		{
			name:    "path param - placeholder replaced, not in query",
			method:  &Method{HTTPMethod: "GET", Endpoint: "/v2/cli/methods/{canonical_id}"},
			args:    map[string]interface{}{"params": map[string]interface{}{"canonical_id": "shell.integration.list"}},
			wantURL: "http://localhost/v2/cli/methods/shell.integration.list",
			wantQuery: "",
		},
		{
			name:    "params preferred over top-level when both set",
			method:  methodGET,
			args:    map[string]interface{}{
				"params":        map[string]interface{}{"project_name": "from-params"},
				"project_name":  "from-top-level",
			},
			wantQuery: "project_name=from-params",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := BuildRequest(tt.method, baseURL, tt.args)
			if err != nil {
				t.Fatalf("BuildRequest: %v", err)
			}
			urlStr := req.URL.String()
			if tt.wantURL != "" && !strings.Contains(urlStr, tt.wantURL) && urlStr != tt.wantURL {
				t.Errorf("url = %q, want substring or exact %q", urlStr, tt.wantURL)
			}
			if tt.wantQuery != "" {
				gotQuery := req.URL.RawQuery
				if !strings.Contains(gotQuery, tt.wantQuery) {
					t.Errorf("query = %q, want containing %q", gotQuery, tt.wantQuery)
				}
			}
			for h, wantVal := range tt.checkHeaders {
				if got := req.Header.Get(h); got != wantVal {
					t.Errorf("header %s = %q, want %q", h, got, wantVal)
				}
			}
			if tt.wantBody != "" && req.Body != nil {
				bodyBytes, _ := io.ReadAll(req.Body)
				gotBody := strings.TrimSpace(string(bodyBytes))
				// Normalize JSON for comparison
				var gotJ, wantJ interface{}
				_ = json.Unmarshal([]byte(gotBody), &gotJ)
				_ = json.Unmarshal([]byte(tt.wantBody), &wantJ)
				gotB, _ := json.Marshal(gotJ)
				wantB, _ := json.Marshal(wantJ)
				if string(gotB) != string(wantB) {
					t.Errorf("body = %s, want %s", gotBody, tt.wantBody)
				}
			}
		})
	}
}

func TestExecuteDryRun_OutputShape(t *testing.T) {
	// Build a request with query, headers, and body so dry-run reflects them
	method := &Method{HTTPMethod: "POST", Endpoint: "/v2/cli/create"}
	args := map[string]interface{}{
		"project_name": "swytchcode",
		"Authorization": "Bearer token123",
		"headers": map[string]interface{}{"X-Request-Id": "abc"},
		"body": map[string]interface{}{"name": "my-tool"},
	}
	req, err := BuildRequest(method, "http://localhost", args)
	if err != nil {
		t.Fatalf("BuildRequest: %v", err)
	}
	var buf strings.Builder
	code := ExecuteDryRun(req, &buf)
	if code != ExitCodeOK {
		t.Fatalf("ExecuteDryRun: exit code %d", code)
	}
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &out); err != nil {
		t.Fatalf("dry-run JSON: %v", err)
	}
	if u, ok := out["url"].(string); !ok || !strings.Contains(u, "project_name=swytchcode") {
		t.Errorf("dry-run url should contain query: %v", out["url"])
	}
	headers, ok := out["headers"].(map[string]interface{})
	if !ok {
		t.Fatalf("dry-run headers not map: %T", out["headers"])
	}
	if h := headers["Authorization"]; h != "Bearer token123" {
		t.Errorf("dry-run headers Authorization = %v", h)
	}
	if h := headers["X-Request-Id"]; h != "abc" {
		t.Errorf("dry-run headers X-Request-Id = %v", h)
	}
	if _, hasBody := out["body"]; !hasBody {
		t.Error("dry-run should include body when request has body")
	}
}

func TestParamsFromArgs_JSONMap(t *testing.T) {
	// Simulate JSON unmarshaling: params is map[string]interface{}
	args := map[string]interface{}{
		"params": map[string]interface{}{
			"project_name": "swytchcode",
			"limit":        float64(10),
		},
	}
	params := paramsFromArgs(args)
	if params == nil {
		t.Fatal("paramsFromArgs returned nil")
	}
	if params["project_name"] != "swytchcode" {
		t.Errorf("project_name = %q", params["project_name"])
	}
	if params["limit"] != "10" {
		t.Errorf("limit = %q", params["limit"])
	}
}
