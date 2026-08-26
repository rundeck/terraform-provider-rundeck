package rundeck

import (
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// requireMinAPIVersion adds an "Insufficient API Version" error diagnostic
// and returns false when configured is below min. subject names what's
// being gated (e.g. "Runner data sources"), used in the diagnostic message.
//
// Comparing the two version strings lexicographically, as several call sites
// in this provider still do (`clients.APIVersion < "56"`), is wrong whenever
// they differ in digit count: "9" < "56" is false under Go's string
// ordering, so a genuinely older single-digit version incorrectly passes a
// ">= 56" check. This compares them as integers instead.
func requireMinAPIVersion(diags *diag.Diagnostics, configured string, min int, subject string) bool {
	version, err := strconv.Atoi(configured)
	if err == nil && version >= min {
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
