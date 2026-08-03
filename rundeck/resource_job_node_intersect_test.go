package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Plain unit tests (no TF_ACC / live Rundeck required) for node_intersect on a
// job reference's node_filters dispatch block.
//
// Rundeck maps this to jobref.nodefilters.dispatch.nodeIntersect (see
// JobExec.toMap/fromMap in rundeck/rundeck). Two behaviours from that mapping
// matter here and are asserted below:
//
//   - nodeIntersect is the only dispatch field Rundeck reads independently of a
//     node filter, and when no filter is set it emits nodefilters carrying
//     nothing but dispatch.nodeIntersect. That shape must survive the round
//     trip, otherwise the read-back drops the flag and every plan shows drift.
//   - An absent flag must stay null in state rather than read back as an
//     explicit false, so an unconfigured dispatch block does not gain a value
//     the configuration never asked for.

// testJobRefDispatchList builds a node_filters list holding a single dispatch
// block whose only set attribute is node_intersect.
func testJobRefDispatchList(t *testing.T, nodeIntersect attr.Value) types.List {
	t.Helper()

	disp, diags := types.ObjectValue(nodeFilterDispatchObjectType.AttrTypes, map[string]attr.Value{
		"thread_count":   types.Int64Null(),
		"keep_going":     types.BoolNull(),
		"rank_attribute": types.StringNull(),
		"rank_order":     types.StringNull(),
		"node_intersect": nodeIntersect,
	})
	if diags.HasError() {
		t.Fatalf("building dispatch object: %v", diags)
	}

	nf, diags := types.ObjectValue(nodeFilterObjectType.AttrTypes, map[string]attr.Value{
		"filter":             types.StringNull(),
		"exclude_filter":     types.StringNull(),
		"exclude_precedence": types.BoolNull(),
		"dispatch":           types.ListValueMust(nodeFilterDispatchObjectType, []attr.Value{disp}),
	})
	if diags.HasError() {
		t.Fatalf("building node_filters object: %v", diags)
	}

	return types.ListValueMust(nodeFilterObjectType, []attr.Value{nf})
}

// testJobRefObject builds a minimal job reference carrying the given
// node_filters list.
func testJobRefObject(t *testing.T, nodeFilters types.List) types.Object {
	t.Helper()

	jobRef, diags := types.ObjectValue(jobRefObjectType.AttrTypes, map[string]attr.Value{
		"uuid":                 types.StringNull(),
		"name":                 types.StringValue("target-job"),
		"group_name":           types.StringNull(),
		"project_name":         types.StringNull(),
		"run_for_each_node":    types.BoolNull(),
		"node_step":            types.BoolNull(),
		"args":                 types.StringNull(),
		"import_options":       types.BoolNull(),
		"child_nodes":          types.BoolNull(),
		"fail_on_disable":      types.BoolNull(),
		"ignore_notifications": types.BoolNull(),
		"node_filters":         nodeFilters,
	})
	if diags.HasError() {
		t.Fatalf("building job reference object: %v", diags)
	}
	return jobRef
}

// testCommandWithJobRefs builds a one-command list whose command references a
// job both directly and through its error handler, so a single conversion
// exercises both code paths.
func testCommandWithJobRefs(t *testing.T, jobRef types.Object) types.List {
	t.Helper()

	handler, diags := types.ObjectValue(errorHandlerObjectType.AttrTypes, map[string]attr.Value{
		"description":                 types.StringNull(),
		"shell_command":               types.StringNull(),
		"inline_script":               types.StringNull(),
		"script_url":                  types.StringNull(),
		"script_file":                 types.StringNull(),
		"script_file_args":            types.StringNull(),
		"expand_token_in_script_file": types.BoolNull(),
		"file_extension":              types.StringNull(),
		"keep_going_on_success":       types.BoolNull(),
		"script_interpreter":          types.ListNull(scriptInterpreterObjectType),
		"job":                         types.ListValueMust(jobRefObjectType, []attr.Value{jobRef}),
		"step_plugin":                 types.ListNull(stepPluginObjectType),
		"node_step_plugin":            types.ListNull(stepPluginObjectType),
	})
	if diags.HasError() {
		t.Fatalf("building error handler object: %v", diags)
	}

	cmd, diags := types.ObjectValue(commandObjectType.AttrTypes, map[string]attr.Value{
		"description":                 types.StringNull(),
		"shell_command":               types.StringNull(),
		"inline_script":               types.StringNull(),
		"script_url":                  types.StringNull(),
		"script_file":                 types.StringNull(),
		"script_file_args":            types.StringNull(),
		"expand_token_in_script_file": types.BoolNull(),
		"file_extension":              types.StringNull(),
		"keep_going_on_success":       types.BoolNull(),
		"plugins":                     types.ListNull(commandPluginsObjectType),
		"script_interpreter":          types.ListNull(scriptInterpreterObjectType),
		"job":                         types.ListValueMust(jobRefObjectType, []attr.Value{jobRef}),
		"step_plugin":                 types.ListNull(stepPluginObjectType),
		"node_step_plugin":            types.ListNull(stepPluginObjectType),
		"error_handler":               types.ListValueMust(errorHandlerObjectType, []attr.Value{handler}),
	})
	if diags.HasError() {
		t.Fatalf("building command object: %v", diags)
	}

	return types.ListValueMust(commandObjectType, []attr.Value{cmd})
}

// jobRefDispatchJSON digs out jobref.nodefilters.dispatch from a converted
// command map, failing the test if any level is missing.
func jobRefDispatchJSON(t *testing.T, jobref interface{}) map[string]interface{} {
	t.Helper()

	ref, ok := jobref.(map[string]interface{})
	if !ok {
		t.Fatalf("jobref is %T, want map", jobref)
	}
	nf, ok := ref["nodefilters"].(map[string]interface{})
	if !ok {
		t.Fatalf("jobref.nodefilters is %T, want map", ref["nodefilters"])
	}
	disp, ok := nf["dispatch"].(map[string]interface{})
	if !ok {
		t.Fatalf("jobref.nodefilters.dispatch is %T, want map", nf["dispatch"])
	}
	return disp
}

func TestConvertCommandsToJSON_nodeIntersect(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			commands := testCommandWithJobRefs(t,
				testJobRefObject(t, testJobRefDispatchList(t, types.BoolValue(tc.value))))

			result, diags := convertCommandsToJSON(context.Background(), commands)
			if diags.HasError() {
				t.Fatalf("convertCommandsToJSON: %v", diags)
			}
			if len(result) != 1 {
				t.Fatalf("got %d commands, want 1", len(result))
			}

			cmd, ok := result[0].(map[string]interface{})
			if !ok {
				t.Fatalf("command is %T, want map", result[0])
			}

			// Direct job reference.
			if got := jobRefDispatchJSON(t, cmd["jobref"])["nodeIntersect"]; got != tc.value {
				t.Errorf("jobref dispatch nodeIntersect = %v, want %v", got, tc.value)
			}

			// Error handler job reference.
			handler, ok := cmd["errorhandler"].(map[string]interface{})
			if !ok {
				t.Fatalf("errorhandler is %T, want map", cmd["errorhandler"])
			}
			if got := jobRefDispatchJSON(t, handler["jobref"])["nodeIntersect"]; got != tc.value {
				t.Errorf("error handler dispatch nodeIntersect = %v, want %v", got, tc.value)
			}
		})
	}
}

func TestConvertCommandsToJSON_nodeIntersectOmittedWhenUnset(t *testing.T) {
	commands := testCommandWithJobRefs(t,
		testJobRefObject(t, testJobRefDispatchList(t, types.BoolNull())))

	result, diags := convertCommandsToJSON(context.Background(), commands)
	if diags.HasError() {
		t.Fatalf("convertCommandsToJSON: %v", diags)
	}

	cmd := result[0].(map[string]interface{})
	ref := cmd["jobref"].(map[string]interface{})
	nf, ok := ref["nodefilters"].(map[string]interface{})
	if !ok {
		t.Fatalf("jobref.nodefilters is %T, want map", ref["nodefilters"])
	}
	if _, exists := nf["dispatch"]; exists {
		t.Errorf("dispatch = %v, want it omitted when no field is set", nf["dispatch"])
	}
}

// dispatchStateAttrs digs node_filters[0].dispatch[0] out of a job reference
// object read back from the API.
func dispatchStateAttrs(t *testing.T, jobList attr.Value) map[string]attr.Value {
	t.Helper()

	jobs, ok := jobList.(types.List)
	if !ok || jobs.IsNull() || len(jobs.Elements()) == 0 {
		t.Fatalf("job reference list is %v, want one element", jobList)
	}
	jobAttrs := jobs.Elements()[0].(types.Object).Attributes()

	nfList, ok := jobAttrs["node_filters"].(types.List)
	if !ok || nfList.IsNull() || len(nfList.Elements()) == 0 {
		t.Fatalf("node_filters is %v, want one element", jobAttrs["node_filters"])
	}
	nfAttrs := nfList.Elements()[0].(types.Object).Attributes()

	dispList, ok := nfAttrs["dispatch"].(types.List)
	if !ok || dispList.IsNull() || len(dispList.Elements()) == 0 {
		t.Fatalf("dispatch is %v, want one element", nfAttrs["dispatch"])
	}
	return dispList.Elements()[0].(types.Object).Attributes()
}

// testJobRefCommandJSON mirrors what Rundeck emits for a job reference whose
// only dispatch setting is nodeIntersect: nodefilters carries the dispatch map
// and nothing else (JobExec.toMap).
func testJobRefCommandJSON(dispatch map[string]interface{}) []interface{} {
	jobref := map[string]interface{}{
		"name":        "target-job",
		"group":       "",
		"nodefilters": map[string]interface{}{"dispatch": dispatch},
	}
	return []interface{}{
		map[string]interface{}{
			"jobref":       jobref,
			"errorhandler": map[string]interface{}{"jobref": jobref},
		},
	}
}

func TestConvertCommandsFromJSON_nodeIntersect(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value bool
	}{
		{"true", true},
		{"false", false},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			commands := testJobRefCommandJSON(map[string]interface{}{"nodeIntersect": tc.value})

			result, diags := convertCommandsFromJSON(context.Background(), commands)
			if diags.HasError() {
				t.Fatalf("convertCommandsFromJSON: %v", diags)
			}
			if len(result.Elements()) != 1 {
				t.Fatalf("got %d commands, want 1", len(result.Elements()))
			}
			cmdAttrs := result.Elements()[0].(types.Object).Attributes()

			// Direct job reference.
			ni := dispatchStateAttrs(t, cmdAttrs["job"])["node_intersect"].(types.Bool)
			if ni.IsNull() || ni.ValueBool() != tc.value {
				t.Errorf("jobref node_intersect = %v, want %v", ni, tc.value)
			}

			// Error handler job reference.
			handlers, ok := cmdAttrs["error_handler"].(types.List)
			if !ok || handlers.IsNull() || len(handlers.Elements()) == 0 {
				t.Fatalf("error_handler is %v, want one element", cmdAttrs["error_handler"])
			}
			handlerAttrs := handlers.Elements()[0].(types.Object).Attributes()
			ni = dispatchStateAttrs(t, handlerAttrs["job"])["node_intersect"].(types.Bool)
			if ni.IsNull() || ni.ValueBool() != tc.value {
				t.Errorf("error handler node_intersect = %v, want %v", ni, tc.value)
			}
		})
	}
}

func TestConvertCommandsFromJSON_nodeIntersectAbsentStaysNull(t *testing.T) {
	// A dispatch block carrying another field but no nodeIntersect: the flag
	// must stay null rather than read back as an explicit false.
	commands := testJobRefCommandJSON(map[string]interface{}{"keepgoing": true})

	result, diags := convertCommandsFromJSON(context.Background(), commands)
	if diags.HasError() {
		t.Fatalf("convertCommandsFromJSON: %v", diags)
	}
	cmdAttrs := result.Elements()[0].(types.Object).Attributes()

	if ni := dispatchStateAttrs(t, cmdAttrs["job"])["node_intersect"].(types.Bool); !ni.IsNull() {
		t.Errorf("node_intersect = %v, want null when the API omits nodeIntersect", ni)
	}
}

// TestAccJob_cmd_referred_job_nodeIntersect exercises node_intersect against a
// live Rundeck, on a job reference with no node filter of its own (the shape
// Rundeck emits as nodefilters carrying only dispatch.nodeIntersect) and on one
// that also sets a filter, both directly under a command and under an error
// handler. The second step asserts the setting round-trips with an empty plan.
func TestAccJob_cmd_referred_job_nodeIntersect(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_cmd_referred_job_nodeIntersect,
				Check: resource.ComposeTestCheckFunc(
					// Reference carrying nothing but node_intersect.
					resource.TestCheckResourceAttr("rundeck_job.caller",
						"command.0.job.0.node_filters.0.dispatch.0.node_intersect", "true"),
					// Same setting on an error handler reference.
					resource.TestCheckResourceAttr("rundeck_job.caller",
						"command.0.error_handler.0.job.0.node_filters.0.dispatch.0.node_intersect", "true"),
					// Reference combining a filter with node_intersect.
					resource.TestCheckResourceAttr("rundeck_job.caller",
						"command.1.job.0.node_filters.0.filter", "name: tacobell"),
					resource.TestCheckResourceAttr("rundeck_job.caller",
						"command.1.job.0.node_filters.0.dispatch.0.node_intersect", "true"),
				),
			},
			{
				// Re-apply with the same config: this must produce an empty plan.
				Config:   testAccJobConfig_cmd_referred_job_nodeIntersect,
				PlanOnly: true,
			},
		},
	})
}

const testAccJobConfig_cmd_referred_job_nodeIntersect = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-job-node-intersect"
  description = "Test project for node_intersect on job references"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "target" {
  project_name      = rundeck_project.test.name
  name              = "target-job"
  description       = "Job to be referenced"
  execution_enabled = true
  command {
    shell_command = "echo 'I am the target job'"
  }
}

resource "rundeck_job" "caller" {
  project_name      = rundeck_project.test.name
  name              = "caller-job"
  description       = "Job referencing another job with node_intersect"
  execution_enabled = true

  command {
    description = "call target on the intersection with this job's nodes"
    job {
      name         = rundeck_job.target.name
      project_name = rundeck_project.test.name
      node_filters {
        dispatch {
          node_intersect = true
        }
      }
    }
    error_handler {
      job {
        name         = rundeck_job.target.name
        project_name = rundeck_project.test.name
        node_filters {
          dispatch {
            node_intersect = true
          }
        }
      }
    }
  }

  command {
    description = "call target with both a filter and node_intersect"
    job {
      name         = rundeck_job.target.name
      project_name = rundeck_project.test.name
      node_filters {
        filter = "name: tacobell"
        dispatch {
          node_intersect = true
        }
      }
    }
  }
}
`
