package commands

import "testing"

func TestParseIntegrationSpec(t *testing.T) {
	tests := []struct {
		spec            string
		wantProject     string
		wantLibrary     string
		wantVersion     string
		wantEmptyTriple bool
	}{
		{
			spec:        "stripe@stripe_sdk.2026-02-25.clover",
			wantProject: "stripe",
			wantLibrary: "stripe_sdk",
			wantVersion: "2026-02-25.clover",
		},
		{
			spec:        "weaviate@lyrid.v1",
			wantProject: "weaviate",
			wantLibrary: "lyrid",
			wantVersion: "v1",
		},
		{
			spec:            "nopercent",
			wantEmptyTriple: true,
		},
		{
			spec:            "only@onepart",
			wantEmptyTriple: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			p, l, v := ParseIntegrationSpec(tt.spec)
			if tt.wantEmptyTriple {
				if p != "" || l != "" || v != "" {
					t.Fatalf("want empty triple, got %q %q %q", p, l, v)
				}
				return
			}
			if p != tt.wantProject || l != tt.wantLibrary || v != tt.wantVersion {
				t.Fatalf("got project=%q library=%q version=%q want %q %q %q", p, l, v, tt.wantProject, tt.wantLibrary, tt.wantVersion)
			}
		})
	}
}
