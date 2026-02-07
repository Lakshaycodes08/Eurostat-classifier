// proposals.go provides helpers for proposal paths and removal (e.g. removing proposals when a library is upgraded or removed).
package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// removeProposalsForLibrary deletes proposal files in .swytchcode/proposals/
// that belong to the given library. Naming conventions:
//   - Method proposals: <library>.<method>.json (e.g. stripe.customers.search.json)
//   - Workflow proposals: workflow.<library>.<name>.json
//
// It returns the number of files removed and any error (only the first error
// is returned if multiple removes fail).
func removeProposalsForLibrary(projectRoot, library string) (removed int, err error) {
	proposalsDir := filepath.Join(projectRoot, ".swytchcode", "proposals")
	entries, err := os.ReadDir(proposalsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	prefix := library + "."
	workflowPrefix := "workflow." + library + "."

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		base := strings.TrimSuffix(name, ".json")
		belongs := base == library || strings.HasPrefix(base, prefix) || strings.HasPrefix(base, workflowPrefix)
		if !belongs {
			continue
		}
		p := filepath.Join(proposalsDir, name)
		if removeErr := os.Remove(p); removeErr != nil {
			if err == nil {
				err = removeErr
			}
			continue
		}
		removed++
	}
	return removed, err
}
