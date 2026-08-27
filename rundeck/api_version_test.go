package rundeck

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestRequireMinAPIVersion_sufficient(t *testing.T) {
	var diags diag.Diagnostics
	if !requireMinAPIVersion(&diags, "56", 44, "Test resources") {
		t.Error("got false, want true for a sufficient version")
	}
	if diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestRequireMinAPIVersion_insufficient(t *testing.T) {
	var diags diag.Diagnostics
	if requireMinAPIVersion(&diags, "20", 44, "Test resources") {
		t.Error("got true, want false for an insufficient version")
	}
	if !diags.HasError() {
		t.Fatal("want an error diagnostic for an insufficient version")
	}
	if summary := diags[0].Summary(); summary != "Insufficient API Version" {
		t.Errorf("summary = %q, want %q", summary, "Insufficient API Version")
	}
}

// A numeric-but-too-low version and a non-numeric one are different
// problems and should produce different diagnostics: the first is a real
// version mismatch, the second is a configuration mistake.
func TestRequireMinAPIVersion_invalid(t *testing.T) {
	var diags diag.Diagnostics
	if requireMinAPIVersion(&diags, "v56", 44, "Test resources") {
		t.Error("got true, want false for a non-numeric version")
	}
	if !diags.HasError() {
		t.Fatal("want an error diagnostic for a non-numeric version")
	}
	if summary := diags[0].Summary(); summary != "Invalid API Version" {
		t.Errorf("summary = %q, want %q", summary, "Invalid API Version")
	}
}

// api_version commonly comes from an environment variable, which can carry
// leading/trailing whitespace (e.g. a trailing newline) without the user
// noticing.
func TestRequireMinAPIVersion_trimsWhitespace(t *testing.T) {
	var diags diag.Diagnostics
	if !requireMinAPIVersion(&diags, " 56\n", 44, "Test resources") {
		t.Error("got false, want true for a sufficient version with surrounding whitespace")
	}
	if diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}
