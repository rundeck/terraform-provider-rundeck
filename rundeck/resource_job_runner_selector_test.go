package rundeck

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestJobRunnerSelector_JSONFieldNames is a regression test for #244 / #258.
// Rundeck's job runnerSelector expects the field names "runnerFilterMode" and
// "runnerFilterType". They were previously serialized as "filterMode"/"filterType",
// which Rundeck silently ignored, dropping the runner filter settings.
func TestJobRunnerSelector_JSONFieldNames(t *testing.T) {
	selector := jobRunnerSelector{
		Filter:     "tags: runner",
		FilterMode: "TAGS",
		FilterType: "TAG_FILTER_AND",
	}

	data, err := json.Marshal(selector)
	if err != nil {
		t.Fatalf("failed to marshal jobRunnerSelector: %s", err)
	}

	jsonStr := string(data)

	for _, expected := range []string{"runnerFilterMode", "runnerFilterType"} {
		if !strings.Contains(jsonStr, expected) {
			t.Errorf("expected marshaled JSON to contain %q, got: %s", expected, jsonStr)
		}
	}

	// Ensure the old (incorrect) field names are not emitted.
	for _, unexpected := range []string{`"filterMode"`, `"filterType"`} {
		if strings.Contains(jsonStr, unexpected) {
			t.Errorf("marshaled JSON should not contain legacy field %s, got: %s", unexpected, jsonStr)
		}
	}
}

// TestJobRunnerSelector_JSONRoundTrip verifies that a runnerSelector returned by
// the Rundeck API (using runnerFilterMode/runnerFilterType) deserializes back into
// the struct, ensuring no plan drift on read.
func TestJobRunnerSelector_JSONRoundTrip(t *testing.T) {
	apiJSON := `{"filter":"tags: runner","runnerFilterMode":"TAGS","runnerFilterType":"TAG_FILTER_AND"}`

	var selector jobRunnerSelector
	if err := json.Unmarshal([]byte(apiJSON), &selector); err != nil {
		t.Fatalf("failed to unmarshal jobRunnerSelector: %s", err)
	}

	if selector.Filter != "tags: runner" {
		t.Errorf("expected Filter %q, got %q", "tags: runner", selector.Filter)
	}
	if selector.FilterMode != "TAGS" {
		t.Errorf("expected FilterMode %q, got %q", "TAGS", selector.FilterMode)
	}
	if selector.FilterType != "TAG_FILTER_AND" {
		t.Errorf("expected FilterType %q, got %q", "TAG_FILTER_AND", selector.FilterType)
	}
}
