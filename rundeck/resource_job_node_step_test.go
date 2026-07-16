package rundeck

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Regression tests for #256: run_for_each_node on a job reference was serialized
// to a non-existent "runForEachNode" key and silently ignored by Rundeck. It must
// drive the API's "nodeStep" flag (with node_step as an alias), and round-trip
// back into both alias attributes.

func TestJobRefNodeStepExplicit(t *testing.T) {
	cases := []struct {
		name         string
		attrs        map[string]attr.Value
		wantVal      bool
		wantExplicit bool
	}{
		{"run_for_each_node true", map[string]attr.Value{"run_for_each_node": types.BoolValue(true)}, true, true},
		{"run_for_each_node false", map[string]attr.Value{"run_for_each_node": types.BoolValue(false)}, false, true},
		{"node_step alias true", map[string]attr.Value{"node_step": types.BoolValue(true)}, true, true},
		{"run_for_each_node precedence over node_step", map[string]attr.Value{"run_for_each_node": types.BoolValue(true), "node_step": types.BoolValue(false)}, true, true},
		{"neither set", map[string]attr.Value{}, false, false},
		{"null values", map[string]attr.Value{"run_for_each_node": types.BoolNull(), "node_step": types.BoolNull()}, false, false},
		{"unknown run_for_each_node falls back to node_step", map[string]attr.Value{"run_for_each_node": types.BoolUnknown(), "node_step": types.BoolValue(true)}, true, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotVal, gotExplicit := jobRefNodeStepExplicit(tc.attrs)
			if gotVal != tc.wantVal || gotExplicit != tc.wantExplicit {
				t.Errorf("jobRefNodeStepExplicit = (%v, %v), want (%v, %v)", gotVal, gotExplicit, tc.wantVal, tc.wantExplicit)
			}
			if got := jobRefNodeStep(tc.attrs); got != tc.wantVal {
				t.Errorf("jobRefNodeStep = %v, want %v", got, tc.wantVal)
			}
		})
	}
}
