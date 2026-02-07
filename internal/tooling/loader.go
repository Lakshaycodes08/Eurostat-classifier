// loader.go will load and parse tooling.json (tools, integrations, mode) as the kernel’s execution contract.
package tooling

// This package will provide loading and representation for tooling.json,
// which defines the execution contract:
//
//   - Tool -> library mapping.
//   - Allowed operations.
//   - Output shape guarantees.
//
// The kernel should load tooling.json once per exec invocation and treat
// it as the single source of truth for what tools exist.

