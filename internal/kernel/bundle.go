// bundle.go loads integration bundles and resolves methods from Wrekenfiles.
package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IntegrationBundle represents a loaded integration bundle.
type IntegrationBundle struct {
	Project   string
	Library   string
	Version   string
	Wrekenfile map[string]interface{}
}

// LoadIntegrationBundle loads an integration bundle from disk.
// integration format: "project.library@version" (e.g., "weaviate.lyrid@v1")
func LoadIntegrationBundle(projectRoot, integration string) (*IntegrationBundle, error) {
	// Parse integration spec: project.library@version
	parts := strings.Split(integration, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid integration format: %q (expected project.library@version)", integration)
	}

	projectLibrary := parts[0]
	version := parts[1]

	// Parse project.library
	libParts := strings.SplitN(projectLibrary, ".", 2)
	if len(libParts) != 2 {
		return nil, fmt.Errorf("invalid integration format: %q (expected project.library@version)", integration)
	}

	project := libParts[0]
	library := libParts[1]

	// Load wrekenfile.yaml
	wrekenPath := filepath.Join(projectRoot, ".swytchcode", "integrations", project, library, version, "wrekenfile.yaml")
	data, err := os.ReadFile(wrekenPath)
	if err != nil {
		return nil, fmt.Errorf("integration %s not installed. Run: swytchcode get %s", integration, project)
	}

	var wrekenfile map[string]interface{}
	if err := yaml.Unmarshal(data, &wrekenfile); err != nil {
		return nil, fmt.Errorf("failed to parse wrekenfile: %w", err)
	}

	return &IntegrationBundle{
		Project:    project,
		Library:    library,
		Version:    version,
		Wrekenfile: wrekenfile,
	}, nil
}

// Method represents a method definition from Wreken METHODS section.
type Method struct {
	CanonicalID string
	HTTPMethod  string
	Endpoint    string
	Headers     map[string]string
	Inputs      interface{}
	Returns     interface{}
	// ... other fields from wrekenfile
}

// ResolveMethod resolves a canonical_id to a Method from the Wreken METHODS section.
func ResolveMethod(bundle *IntegrationBundle, canonicalID string) (*Method, error) {
	// Look for METHODS section
	methodsRaw, ok := bundle.Wrekenfile["METHODS"]
	if !ok {
		return nil, fmt.Errorf("METHODS section not found in wrekenfile")
	}

	methods, ok := methodsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("METHODS must be a map")
	}

	// Find method by canonical_id key
	methodEntry, ok := methods[canonicalID]
	if !ok {
		return nil, fmt.Errorf("method %q not found in wrekenfile", canonicalID)
	}

	methodMap, ok := methodEntry.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("method entry must be a map")
	}

	method := &Method{
		CanonicalID: canonicalID,
	}

	// Extract HTTP method and endpoint
	if httpRaw, ok := methodMap["HTTP"]; ok {
		if httpMap, ok := httpRaw.(map[string]interface{}); ok {
			if methodRaw, ok := httpMap["METHOD"].(string); ok {
				method.HTTPMethod = methodRaw
			}
			if endpointRaw, ok := httpMap["ENDPOINT"].(string); ok {
				method.Endpoint = endpointRaw
			}
			// Extract headers
			if headersRaw, ok := httpMap["HEADERS"]; ok {
				if headersMap, ok := headersRaw.(map[string]interface{}); ok {
					method.Headers = make(map[string]string)
					for k, v := range headersMap {
						if vStr, ok := v.(string); ok {
							method.Headers[k] = vStr
						}
					}
				}
			}
		}
	}

	// Extract INPUTS
	if inputsRaw, ok := methodMap["INPUTS"]; ok {
		method.Inputs = inputsRaw
	}

	// Extract RETURNS
	if returnsRaw, ok := methodMap["RETURNS"]; ok {
		method.Returns = returnsRaw
	}

	return method, nil
}
