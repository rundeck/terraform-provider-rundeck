package rundeck

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// requireMinAPIVersion adds an error diagnostic and returns false when
// configured can't be parsed as an integer, or parses but is below min.
// subject names what's being gated (e.g. "Runner data sources"), used in
// the diagnostic message.
//
// Comparing the two version strings lexicographically, as several call sites
// in this provider still do (`clients.APIVersion < "56"`), is wrong whenever
// they differ in digit count: "9" < "56" is false under Go's string
// ordering, so a genuinely older single-digit version incorrectly passes a
// ">= 56" check. This compares them as integers instead.
func requireMinAPIVersion(diags *diag.Diagnostics, configured string, min int, subject string) bool {
	// api_version commonly comes from an environment variable, which can
	// carry leading/trailing whitespace (e.g. a trailing newline) without
	// the user noticing; trim it before parsing so that doesn't fail here.
	version, err := strconv.Atoi(strings.TrimSpace(configured))
	if err != nil {
		diags.AddError(
			"Invalid API Version",
			fmt.Sprintf(
				"Could not parse the configured api_version %q as an integer: %s. Please set api_version to a plain number (e.g. \"%d\").",
				configured, err.Error(), min,
			),
		)
		return false
	}
	if version >= min {
		return true
	}
	diags.AddError(
		"Insufficient API Version",
		fmt.Sprintf(
			"%s require API version %d or higher (currently configured: %s). Please update your provider configuration with api_version = \"%d\" or higher.",
			subject, min, configured, min,
		),
	)
	return false
}
