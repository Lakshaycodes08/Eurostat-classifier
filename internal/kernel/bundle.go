// bundle.go loads integration bundles and resolves methods from Wrekenfiles.
package kernel

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"gitlab.com/swytchcode/cli/internal/constants"
	"gitlab.com/swytchcode/cli/internal/registry"
	"gitlab.com/swytchcode/cli/internal/util"
	"gopkg.in/yaml.v3"
)

// IntegrationBundle represents a loaded integration bundle.
type IntegrationBundle struct {
	Project            string
	Library            string
	Version            string
	SandboxEndpoint    string
	ProductionEndpoint string
	Wrekenfile         map[string]interface{}
}

// safeName rejects path components that could enable directory traversal.
// Valid components contain only alphanumerics, hyphens, underscores, and dots,
// and must not be empty or the special ".." value.
func safeName(s string) error {
	if s == "" || s == ".." {
		return fmt.Errorf("invalid path component %q", s)
	}
	if strings.ContainsAny(s, "/\\") {
		return fmt.Errorf("path component %q must not contain slashes", s)
	}
	return nil
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

	// Validate path components to prevent directory traversal
	for _, component := range []string{project, library, version} {
		if err := safeName(component); err != nil {
			return nil, fmt.Errorf("invalid integration %q: %w", integration, err)
		}
	}

	// Load wrekenfile.yaml
	wrekenPath := util.Join(util.IntegrationVersionDir(projectRoot, project, library, version), constants.WrekenfileYAMLFile)
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
	methodsRaw, ok := bundle.Wrekenfile[constants.WrekenMethods]
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
	if httpRaw, ok := methodMap[constants.WrekenHTTP]; ok {
		if httpMap, ok := httpRaw.(map[string]interface{}); ok {
			if methodRaw, ok := httpMap[constants.WrekenHTTPMethod].(string); ok {
				method.HTTPMethod = methodRaw
			}
			if endpointRaw, ok := httpMap[constants.WrekenEndpoint].(string); ok {
				method.Endpoint = endpointRaw
			}
			// Extract headers
			if headersRaw, ok := httpMap[constants.WrekenHeaders]; ok {
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
	if inputsRaw, ok := methodMap[constants.WrekenInputs]; ok {
		method.Inputs = inputsRaw
	}

	// Extract RETURNS
	if returnsRaw, ok := methodMap[constants.WrekenReturns]; ok {
		method.Returns = returnsRaw
	}

	return method, nil
}

// ---------------------------------------------------------------------------
// Multi-library bundle support
// ---------------------------------------------------------------------------

// BundleMap maps library_uuid → kernel IntegrationBundle.
// Used during workflow execution to route each step to its correct bundle.
type BundleMap map[string]*IntegrationBundle

// maxWrekenSize is the maximum allowed size of a decoded wrekenfile (10 MB).
const maxWrekenSize = 10 * 1024 * 1024

// ParseRegistryBundle converts a registry API bundle (base64 YAML content)
// into a kernel IntegrationBundle ready for use with ResolveMethod.
func ParseRegistryBundle(rb registry.IntegrationBundle) (*IntegrationBundle, error) {
	yamlBytes, err := base64.StdEncoding.DecodeString(rb.Files.Wreken.Content)
	if err != nil {
		return nil, fmt.Errorf("decode wrekenfile for %q: %w", rb.Integration, err)
	}
	if len(yamlBytes) > maxWrekenSize {
		return nil, fmt.Errorf("wrekenfile for %q is too large (%d bytes, max %d)", rb.Integration, len(yamlBytes), maxWrekenSize)
	}

	var wrekenfile map[string]interface{}
	if err := yaml.Unmarshal(yamlBytes, &wrekenfile); err != nil {
		return nil, fmt.Errorf("parse wrekenfile for %q: %w", rb.Integration, err)
	}

	// Derive project and library from integration string "project.library" (no @version here)
	project := ""
	library := rb.Integration
	if i := strings.Index(rb.Integration, "."); i > 0 {
		project = rb.Integration[:i]
		library = rb.Integration[i+1:]
	}

	return &IntegrationBundle{
		Project:            project,
		Library:            library,
		Version:            rb.Version,
		SandboxEndpoint:    rb.SandboxEndpoint,
		ProductionEndpoint: rb.ProductionEndpoint,
		Wrekenfile:         wrekenfile,
	}, nil
}

// FetchBundleMap fetches all bundles required for a workflow and returns a BundleMap
// keyed by library_uuid. Calls the workflow-bundles endpoint so the backend resolves
// which libraries are needed (multi-library aware).
func FetchBundleMap(ctx context.Context, client *registry.Client, projectName, canonicalID string) (BundleMap, error) {
	resp, err := client.GetWorkflowBundles(ctx, projectName, canonicalID)
	if err != nil {
		return nil, fmt.Errorf("fetch workflow bundles: %w", err)
	}

	bundleMap := make(BundleMap, len(resp.Bundles))
	for _, rb := range resp.Bundles {
		b, err := ParseRegistryBundle(rb)
		if err != nil {
			return nil, err
		}
		// Key by library_uuid; fall back to integration name if uuid missing
		key := rb.LibraryUUID
		if key == "" {
			key = rb.Integration
		}
		bundleMap[key] = b
	}
	return bundleMap, nil
}
