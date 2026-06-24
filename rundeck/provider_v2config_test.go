package rundeck

import (
	"net/url"
	"strings"
	"testing"
)

// TestBuildV2Configuration_UsesConfiguredAPIVersion is a regression test for #252.
// The V2 (OpenAPI) SDK's default server URL hardcodes /api/56, which caused V2
// resources (e.g. rundeck_webhook) to ignore the provider's configured
// api_version. buildV2Configuration must put the configured version into the
// resolved server URL path.
func TestBuildV2Configuration_UsesConfiguredAPIVersion(t *testing.T) {
	cases := []struct {
		name       string
		apiVersion string
	}{
		{"custom lower version", "52"},
		{"default version", "56"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			apiURL, err := url.Parse("https://rundeck.example.com/api/" + tc.apiVersion)
			if err != nil {
				t.Fatalf("parsing url: %s", err)
			}

			cfg := buildV2Configuration(apiURL, tc.apiVersion, "test")

			if cfg.Host != "rundeck.example.com" {
				t.Errorf("cfg.Host = %q, want %q", cfg.Host, "rundeck.example.com")
			}
			if cfg.Scheme != "https" {
				t.Errorf("cfg.Scheme = %q, want %q", cfg.Scheme, "https")
			}

			resolved, err := cfg.Servers.URL(0, nil)
			if err != nil {
				t.Fatalf("resolving server url: %s", err)
			}
			wantPath := "/api/" + tc.apiVersion
			if !strings.HasSuffix(resolved, wantPath) {
				t.Errorf("resolved server URL = %q, want it to end with %q", resolved, wantPath)
			}
		})
	}
}
