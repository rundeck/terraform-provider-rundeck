package rundeck

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

// TestUserAgentTransport_RoundTrip verifies that the User-Agent header is correctly added to requests
func TestUserAgentTransport_RoundTrip(t *testing.T) {
	version := "1.2.0"
	expectedUA := fmt.Sprintf("terraform-provider-rundeck/%s (%s; %s)", version, runtime.Version(), runtime.GOOS)

	// Create a test server that captures the User-Agent header
	var capturedUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create transport with custom User-Agent
	transport := newUserAgentTransport(nil, version)
	client := &http.Client{Transport: transport}

	// Make a request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify User-Agent header was set
	if capturedUA != expectedUA {
		t.Errorf("User-Agent header mismatch.\nExpected: %s\nGot: %s", expectedUA, capturedUA)
	}
}

// TestUserAgentTransport_PreservesExistingHeaders verifies that other headers are not affected
func TestUserAgentTransport_PreservesExistingHeaders(t *testing.T) {
	version := "1.2.0"

	// Create a test server that captures all headers
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create transport
	transport := newUserAgentTransport(nil, version)
	client := &http.Client{Transport: transport}

	// Make a request with custom headers
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Custom-Header", "test-value")
	req.Header.Set("Authorization", "Bearer token123")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify custom headers are preserved
	if capturedHeaders.Get("X-Custom-Header") != "test-value" {
		t.Error("X-Custom-Header was not preserved")
	}
	if capturedHeaders.Get("Authorization") != "Bearer token123" {
		t.Error("Authorization header was not preserved")
	}
	if capturedHeaders.Get("User-Agent") == "" {
		t.Error("User-Agent header was not set")
	}
}

// TestBuildUserAgent verifies the User-Agent string format
func TestBuildUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "release version",
			version:  "1.2.0",
			expected: fmt.Sprintf("terraform-provider-rundeck/1.2.0 (%s; %s)", runtime.Version(), runtime.GOOS),
		},
		{
			name:     "dev version",
			version:  "dev",
			expected: fmt.Sprintf("terraform-provider-rundeck/dev (%s; %s)", runtime.Version(), runtime.GOOS),
		},
		{
			name:     "test version",
			version:  "test",
			expected: fmt.Sprintf("terraform-provider-rundeck/test (%s; %s)", runtime.Version(), runtime.GOOS),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildUserAgent(tt.version)
			if result != tt.expected {
				t.Errorf("buildUserAgent(%q) = %q, want %q", tt.version, result, tt.expected)
			}
		})
	}
}

// TestNewHTTPClientWithUserAgent verifies that the helper function creates a proper client
func TestNewHTTPClientWithUserAgent(t *testing.T) {
	version := "1.2.0"

	client := newHTTPClientWithUserAgent(version)

	if client == nil {
		t.Fatal("newHTTPClientWithUserAgent returned nil")
	}

	if client.Transport == nil {
		t.Fatal("HTTP client transport is nil")
	}

	// Verify it's our custom transport
	if _, ok := client.Transport.(*userAgentTransport); !ok {
		t.Errorf("Transport is not *userAgentTransport, got %T", client.Transport)
	}
}

// TestUserAgentTransport_DefaultTransport verifies that nil base uses DefaultTransport
func TestUserAgentTransport_DefaultTransport(t *testing.T) {
	version := "1.2.0"

	transport := newUserAgentTransport(nil, version)

	if transport.base != http.DefaultTransport {
		t.Error("Expected base transport to be http.DefaultTransport when nil is passed")
	}
}

// TestUserAgentTransport_CustomBase verifies that custom base transport is used
func TestUserAgentTransport_CustomBase(t *testing.T) {
	version := "1.2.0"
	customBase := &http.Transport{}

	transport := newUserAgentTransport(customBase, version)

	if transport.base != customBase {
		t.Error("Expected base transport to be the custom transport provided")
	}
}

// TestUserAgentTransport_OriginalRequestUnmodified verifies the original request is not mutated
func TestUserAgentTransport_OriginalRequestUnmodified(t *testing.T) {
	version := "1.2.0"

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create transport
	transport := newUserAgentTransport(nil, version)
	client := &http.Client{Transport: transport}

	// Make a request
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Capture original User-Agent (should be empty)
	originalUA := req.Header.Get("User-Agent")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Verify original request was not modified
	afterUA := req.Header.Get("User-Agent")
	if originalUA != afterUA {
		t.Errorf("Original request was modified. Before: %q, After: %q", originalUA, afterUA)
	}
}
