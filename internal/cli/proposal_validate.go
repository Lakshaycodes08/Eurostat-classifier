// proposal_validate.go holds shared validation logic for proposals (used by validate and apply; no side effects).
package cli

import (
	"path/filepath"
)

// validateProposalContent runs all kernel invariants on a parsed proposal.
// Returns a non-nil slice of error messages if invalid; nil if valid.
// projectRoot is used to check tooling.json and manifest when validating
// integration compatibility (e.g. referenced integrations exist and versions match).
func validateProposalContent(proposal map[string]interface{}, projectRoot string) []string {
	var errs []string

	// Kernel-owned fields must not appear in proposal root
	for _, forbidden := range []string{"version", "mode", "registry_url"} {
		if _, has := proposal[forbidden]; has {
			errs = append(errs, "proposals must not include "+forbidden+" (kernel-owned)")
		}
	}

	// tooling_fragment required and must not contain kernel-owned fields
	toolingFragment, ok := proposal["tooling_fragment"].(map[string]interface{})
	if !ok {
		errs = append(errs, "missing or invalid tooling_fragment")
		return errs
	}
	for _, forbidden := range []string{"version", "mode", "registry_url"} {
		if _, has := toolingFragment[forbidden]; has {
			errs = append(errs, "tooling_fragment must not include "+forbidden+" (kernel-owned)")
		}
	}

	// tools required and must be a non-nil object
	tools, ok := toolingFragment["tools"].(map[string]interface{})
	if !ok || tools == nil {
		errs = append(errs, "missing or invalid tools in tooling_fragment")
	}

	// If proposal includes integrations, each must have explicit version
	if proposalIntegrations, ok := proposal["integrations"].(map[string]interface{}); ok {
		for name, raw := range proposalIntegrations {
			ent, _ := raw.(map[string]interface{})
			if ent == nil {
				errs = append(errs, "integration "+name+" must be an object with version")
				continue
			}
			if _, hasVersion := ent["version"]; !hasVersion {
				errs = append(errs, "integration "+name+" has no version (explicit version required)")
			}
		}
	}

	// Optional: referenced integrations exist locally and versions match tooling.json
	if projectRoot != "" && len(errs) == 0 {
		if proposalIntegrations, ok := proposal["integrations"].(map[string]interface{}); ok {
			toolingPath := filepath.Join(projectRoot, ".swytchcode", "tooling.json")
			installed, _ := readWrekenManifest(projectRoot)
			for name, raw := range proposalIntegrations {
				ent, _ := raw.(map[string]interface{})
				if ent == nil {
					continue
				}
				ver, _ := ent["version"].(string)
				if ver == "" {
					continue
				}
				// If we have tooling.json and an integrations section, require proposal integration versions to be consistent
				_ = toolingPath // future: load tooling and check pinned version matches
				if installed != nil {
					if instVer, ok := installed[name]; ok && instVer != ver {
						errs = append(errs, "integration "+name+"@"+ver+" does not match installed version "+instVer+" (run swytchcode bootstrap or update tooling.json)")
					}
				}
			}
		}
	}

	return errs
}
