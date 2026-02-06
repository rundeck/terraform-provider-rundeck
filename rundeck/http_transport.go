package rundeck

import (
	"fmt"
	"net/http"
	"runtime"
)

// userAgentTransport is a custom http.RoundTripper that injects a User-Agent header
// into all outgoing HTTP requests. This enables tracking of provider usage in SaaS analytics.
type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

// RoundTrip implements the http.RoundTripper interface by adding a User-Agent header
// to each request before delegating to the base transport.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	clonedReq := req.Clone(req.Context())

	// Set the User-Agent header
	clonedReq.Header.Set("User-Agent", t.userAgent)

	// Delegate to the base transport
	return t.base.RoundTrip(clonedReq)
}

// newUserAgentTransport creates a new userAgentTransport with the specified version.
// If base is nil, http.DefaultTransport is used.
func newUserAgentTransport(base http.RoundTripper, version string) *userAgentTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	userAgent := buildUserAgent(version)

	return &userAgentTransport{
		base:      base,
		userAgent: userAgent,
	}
}

// newHTTPClientWithUserAgent creates a new http.Client with a custom User-Agent transport.
// This is the primary function used by the provider to create HTTP clients for API calls.
func newHTTPClientWithUserAgent(version string) *http.Client {
	return &http.Client{
		Transport: newUserAgentTransport(nil, version),
	}
}

// buildUserAgent constructs the User-Agent string in the format:
// terraform-provider-rundeck/<version> (go<go-version>; <os>)
//
// Examples:
//   - terraform-provider-rundeck/1.2.0 (go1.24; darwin)
//   - terraform-provider-rundeck/dev (go1.24; linux)
func buildUserAgent(version string) string {
	goVersion := runtime.Version()
	goos := runtime.GOOS

	// Format: terraform-provider-rundeck/<version> (go<version>; <os>)
	return fmt.Sprintf("terraform-provider-rundeck/%s (%s; %s)", version, goVersion, goos)
}
