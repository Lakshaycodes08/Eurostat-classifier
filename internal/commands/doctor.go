// doctor.go implements `swytchcode doctor` diagnostics (local project + auth posture).
package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gitlab.com/swytchcode/swytchcode-cli/internal/auth"
	"gitlab.com/swytchcode/swytchcode-cli/internal/kernel"
	"gitlab.com/swytchcode/swytchcode-cli/internal/manifest"
	"gitlab.com/swytchcode/swytchcode-cli/internal/registry"
	"gitlab.com/swytchcode/swytchcode-cli/internal/util"
)

// DoctorStatus is one of ok, warn, or error.
type DoctorStatus string

const (
	DoctorOK    DoctorStatus = "ok"
	DoctorWarn  DoctorStatus = "warn"
	DoctorError DoctorStatus = "error"
)

// DoctorCheck is a single diagnostic line.
type DoctorCheck struct {
	ID      string       `json:"id"`
	Status  DoctorStatus `json:"status"`
	Message string       `json:"message"`
}

// DoctorReport is the full `doctor` output (human or JSON).
type DoctorReport struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}

func (r *DoctorReport) add(id string, st DoctorStatus, msg string) {
	r.Checks = append(r.Checks, DoctorCheck{ID: id, Status: st, Message: msg})
	if st == DoctorError {
		r.OK = false
	}
}

// RunDoctor analyzes the project at projectRoot and returns a structured report.
func RunDoctor(projectRoot string) *DoctorReport {
	rep := &DoctorReport{OK: true, Checks: make([]DoctorCheck, 0, 16)}

	tooling, err := util.LoadToolingJSON(projectRoot)
	if err != nil {
		rep.add("tooling_json", DoctorError, err.Error())
		return rep
	}
	rep.add("tooling_json", DoctorOK, "tooling.json present and valid JSON")

	integs := integrationSpecsFromTooling(tooling)
	if len(integs) == 0 {
		rep.add("integrations", DoctorWarn, "no integrations declared in tooling.json (run 'swytchcode get' / 'swytchcode add')")
	} else {
		sort.Strings(integs)
		for _, spec := range integs {
			if _, err := kernel.LoadIntegrationBundle(projectRoot, spec); err != nil {
				rep.add("bundle:"+spec, DoctorError, err.Error())
			} else {
				rep.add("bundle:"+spec, DoctorOK, "bundle installed and wrekenfile parses")
			}
		}
	}

	manPath := util.ManifestPath(projectRoot)
	if _, err := os.Stat(manPath); err != nil {
		if len(integs) > 0 {
			rep.add("manifest_json", DoctorError, "manifest.json missing — run 'swytchcode get' or 'swytchcode bootstrap'")
		} else {
			rep.add("manifest_json", DoctorWarn, "manifest.json missing (expected after first 'swytchcode get')")
		}
	} else {
		m, err := manifest.Read(projectRoot)
		if err != nil {
			rep.add("manifest_json", DoctorError, "parse manifest.json: "+err.Error())
		} else {
			rep.add("manifest_json", DoctorOK, fmt.Sprintf("manifest.json OK (%d integration entries)", len(m)))
			for key, ent := range m {
				for _, pair := range []struct {
					label, url string
				}{
					{"sandbox_endpoint", ent.SandboxEndpoint},
					{"production_endpoint", ent.ProductionEndpoint},
				} {
					if pair.url == "" {
						continue
					}
					if err := kernel.ValidateExecutionBaseURL(pair.url); err != nil {
						rep.add("endpoint:"+key+":"+pair.label, DoctorError,
							fmt.Sprintf("%s for %q: %v", pair.label, key, err))
					}
				}
			}
		}
	}

	if os.Getenv("SWYTCHCODE_TOKEN") != "" {
		rep.add("auth", DoctorOK, "SWYTCHCODE_TOKEN is set (service token; not validated against API)")
	} else {
		s, err := auth.Load()
		if err != nil {
			rep.add("auth", DoctorWarn, "no SWYTCHCODE_TOKEN and no user session — fine for local exec; set token or run 'swytchcode login' for registry commands")
		} else if s.IsExpired() {
			rep.add("auth", DoctorWarn, "user session expired — run 'swytchcode login' or set SWYTCHCODE_TOKEN for registry/auth commands")
		} else {
			rep.add("auth", DoctorOK, "user session present (not expired)")
		}
	}

	if os.Getenv("SWYTCHCODE_INSECURE") == "1" {
		if registry.IsCILikeEnv() {
			rep.add("swytchcode_insecure", DoctorError, "SWYTCHCODE_INSECURE=1 blocks registry traffic in CI — unset for get/bootstrap/search in pipelines")
		} else {
			rep.add("swytchcode_insecure", DoctorWarn, "SWYTCHCODE_INSECURE=1 disables TLS verification — dev only")
		}
	}

	return rep
}

func integrationSpecsFromTooling(tooling map[string]interface{}) []string {
	seen := make(map[string]struct{})

	if raw, ok := tooling["integrations"].(map[string]interface{}); ok {
		for pl, meta := range raw {
			ver := ""
			if m, ok := meta.(map[string]interface{}); ok {
				if v, ok := m["version"].(string); ok {
					ver = v
				}
			}
			if ver == "" {
				continue
			}
			spec := pl + "@" + ver
			seen[spec] = struct{}{}
		}
	}

	if tools, ok := tooling["tools"].(map[string]interface{}); ok {
		for _, toolRaw := range tools {
			tm, ok := toolRaw.(map[string]interface{})
			if !ok {
				continue
			}
			if integ, ok := tm["integration"].(string); ok && integ != "" {
				seen[integ] = struct{}{}
			}
			if steps, ok := tm["steps"].([]interface{}); ok {
				for _, sRaw := range steps {
					sm, ok := sRaw.(map[string]interface{})
					if !ok {
						continue
					}
					if integ, ok := sm["integration"].(string); ok && integ != "" {
						seen[integ] = struct{}{}
					}
				}
			}
		}
	}

	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	return out
}

// FormatDoctorHuman renders the report for stdout (no JSON).
func FormatDoctorHuman(r *DoctorReport) string {
	var b strings.Builder
	for _, c := range r.Checks {
		icon := "✓"
		switch c.Status {
		case DoctorWarn:
			icon = "!"
		case DoctorError:
			icon = "✗"
		}
		fmt.Fprintf(&b, "%s [%s] %s\n", icon, c.ID, c.Message)
	}
	if r.OK {
		b.WriteString("\nAll checks passed (no errors).\n")
	} else {
		b.WriteString("\nSome checks failed — fix errors above.\n")
	}
	return b.String()
}
