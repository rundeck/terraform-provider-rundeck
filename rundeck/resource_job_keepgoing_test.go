package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These are plain unit tests (no TF_ACC / live Rundeck required). They guard
// the mapping that keeps the two job-level keepgoing flags independent:
//   - continue_on_error            <-> workflow  <sequence keepgoing>
//   - continue_next_node_on_error  <-> node      <dispatch keepgoing>
// Previously both attributes were wired to the dispatch keepgoing, leaving
// continue_on_error inert and making the two flags impossible to differ.

// testSingleShellCommandList builds a minimal one-command list so that
// planToJobJSON produces a non-nil Sequence.
func testSingleShellCommandList(t *testing.T) types.List {
	t.Helper()

	emptyObjList := types.ListNull(types.ObjectType{})
	cmd, diags := types.ObjectValue(commandObjectType.AttrTypes, map[string]attr.Value{
		"description":                 types.StringValue("echo"),
		"shell_command":               types.StringValue("echo hi"),
		"inline_script":               types.StringNull(),
		"script_url":                  types.StringNull(),
		"script_file":                 types.StringNull(),
		"script_file_args":            types.StringNull(),
		"expand_token_in_script_file": types.BoolNull(),
		"file_extension":              types.StringNull(),
		"keep_going_on_success":       types.BoolNull(),
		"plugins":                     emptyObjList,
		"script_interpreter":          emptyObjList,
		"job":                         emptyObjList,
		"step_plugin":                 emptyObjList,
		"node_step_plugin":            emptyObjList,
		"error_handler":               emptyObjList,
	})
	if diags.HasError() {
		t.Fatalf("building command object: %v", diags)
	}

	list, diags := types.ListValue(commandObjectType, []attr.Value{cmd})
	if diags.HasError() {
		t.Fatalf("building command list: %v", diags)
	}
	return list
}

func TestPlanToJobJSON_keepGoingFlagsAreIndependent(t *testing.T) {
	cases := []struct {
		name             string
		continueOnError  bool
		continueNextNode bool
	}{
		{"workflow on, node off", true, false},
		{"workflow off, node on", false, true},
	}

	r := &jobResource{}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			plan := &jobResourceModel{
				Name:                    types.StringValue("test-job"),
				ProjectName:             types.StringValue("test-project"),
				Description:             types.StringValue("desc"),
				Command:                 testSingleShellCommandList(t),
				ContinueOnError:         types.BoolValue(tc.continueOnError),
				ContinueNextNodeOnError: types.BoolValue(tc.continueNextNode),
			}

			job, err := r.planToJobJSON(context.Background(), plan)
			if err != nil {
				t.Fatalf("planToJobJSON: %v", err)
			}

			if job.Sequence == nil {
				t.Fatal("expected job.Sequence to be set")
			}
			if job.Sequence.KeepGoing != tc.continueOnError {
				t.Errorf("workflow sequence keepgoing = %v, want %v (from continue_on_error)",
					job.Sequence.KeepGoing, tc.continueOnError)
			}

			if job.NodeFilters == nil || job.NodeFilters.Dispatch == nil {
				t.Fatal("expected job.NodeFilters.Dispatch to be set")
			}
			if job.NodeFilters.Dispatch.KeepGoing != tc.continueNextNode {
				t.Errorf("node dispatch keepgoing = %v, want %v (from continue_next_node_on_error)",
					job.NodeFilters.Dispatch.KeepGoing, tc.continueNextNode)
			}
		})
	}
}

func TestJobJSONToState_keepGoingFlagsReadIndependently(t *testing.T) {
	r := &jobResource{}
	job := &jobJSON{
		Sequence: &jobSequence{KeepGoing: true, Commands: []interface{}{}},
		NodeFilters: &jobNodeFilters{
			Dispatch: &jobDispatch{KeepGoing: false},
		},
	}

	state := &jobResourceModel{}
	if err := r.jobJSONToState(context.Background(), job, state); err != nil {
		t.Fatalf("jobJSONToState: %v", err)
	}

	if !state.ContinueOnError.ValueBool() {
		t.Error("sequence keepgoing=true should map to continue_on_error=true")
	}
	if state.ContinueNextNodeOnError.ValueBool() {
		t.Error("dispatch keepgoing=false should map to continue_next_node_on_error=false")
	}
}

func TestJobJSONAPIToState_keepGoingFlagsReadIndependently(t *testing.T) {
	r := &jobResource{}
	job := &JobJSON{
		Sequence: map[string]interface{}{"keepgoing": true},
		NodeFilters: map[string]interface{}{
			"dispatch": map[string]interface{}{"keepgoing": false},
		},
	}

	state := &jobResourceModel{}
	if err := r.jobJSONAPIToState(context.Background(), job, state); err != nil {
		t.Fatalf("jobJSONAPIToState: %v", err)
	}

	if !state.ContinueOnError.ValueBool() {
		t.Error("sequence keepgoing=true should map to continue_on_error=true")
	}
	if state.ContinueNextNodeOnError.ValueBool() {
		t.Error("dispatch keepgoing=false should map to continue_next_node_on_error=false")
	}
}
