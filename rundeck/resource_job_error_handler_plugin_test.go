package rundeck

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// An error handler may itself be a plugin step (flow-control, for instance).
// Both directions used to drop it: the write side never emitted "type" and
// "configuration", so Rundeck stored a handler with no action and silently
// discarded it; the read side then returned no block at all and the apply
// failed with `block count changed from 1 to 0` — while the plan looked fine.

// testCommandWithPluginErrorHandler builds a command whose error handler is a
// plugin step, using the shared object types so the shapes stay in sync with
// the schema.
func testCommandWithPluginErrorHandler(t *testing.T, nodeStepPlugin bool) types.List {
	t.Helper()

	plugin, diags := types.ObjectValue(stepPluginObjectType.AttrTypes, map[string]attr.Value{
		"type": types.StringValue("flow-control"),
		"config": types.MapValueMust(types.StringType, map[string]attr.Value{
			"halt":   types.StringValue("true"),
			"fail":   types.StringValue("false"),
			"status": types.StringValue("inactive-region"),
		}),
	})
	if diags.HasError() {
		t.Fatalf("building plugin object: %v", diags)
	}

	handlerAttrs := map[string]attr.Value{
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
		"job":                         types.ListNull(jobRefObjectType),
		"step_plugin":                 types.ListNull(stepPluginObjectType),
		"node_step_plugin":            types.ListNull(stepPluginObjectType),
	}
	if nodeStepPlugin {
		handlerAttrs["node_step_plugin"] = types.ListValueMust(stepPluginObjectType, []attr.Value{plugin})
	} else {
		handlerAttrs["step_plugin"] = types.ListValueMust(stepPluginObjectType, []attr.Value{plugin})
	}

	handler, diags := types.ObjectValue(errorHandlerObjectType.AttrTypes, handlerAttrs)
	if diags.HasError() {
		t.Fatalf("building error handler object: %v", diags)
	}

	cmd, diags := types.ObjectValue(commandObjectType.AttrTypes, map[string]attr.Value{
		"description":                 types.StringNull(),
		"shell_command":               types.StringValue("false"),
		"inline_script":               types.StringNull(),
		"script_url":                  types.StringNull(),
		"script_file":                 types.StringNull(),
		"script_file_args":            types.StringNull(),
		"expand_token_in_script_file": types.BoolNull(),
		"file_extension":              types.StringNull(),
		"keep_going_on_success":       types.BoolNull(),
		"plugins":                     types.ListNull(commandPluginsObjectType),
		"script_interpreter":          types.ListNull(scriptInterpreterObjectType),
		"job":                         types.ListNull(jobRefObjectType),
		"step_plugin":                 types.ListNull(stepPluginObjectType),
		"node_step_plugin":            types.ListNull(stepPluginObjectType),
		"error_handler":               types.ListValueMust(errorHandlerObjectType, []attr.Value{handler}),
	})
	if diags.HasError() {
		t.Fatalf("building command object: %v", diags)
	}

	return types.ListValueMust(commandObjectType, []attr.Value{cmd})
}

func TestConvertCommandsToJSON_pluginErrorHandler(t *testing.T) {
	for _, tc := range []struct {
		name         string
		nodeStep     bool
		wantNodeStep bool
	}{
		{"workflow step", false, false},
		{"node step", true, true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			result, diags := convertCommandsToJSON(
				context.Background(), testCommandWithPluginErrorHandler(t, tc.nodeStep))
			if diags.HasError() {
				t.Fatalf("convertCommandsToJSON: %v", diags)
			}

			cmd, ok := result[0].(map[string]interface{})
			if !ok {
				t.Fatalf("command is %T, want map", result[0])
			}
			handler, ok := cmd["errorhandler"].(map[string]interface{})
			if !ok {
				t.Fatalf("errorhandler is %T, want map", cmd["errorhandler"])
			}

			if got := handler["type"]; got != "flow-control" {
				t.Errorf("errorhandler type = %v, want flow-control", got)
			}
			if got := handler["nodeStep"]; got != tc.wantNodeStep {
				t.Errorf("errorhandler nodeStep = %v, want %v", got, tc.wantNodeStep)
			}

			config, ok := handler["configuration"].(map[string]string)
			if !ok {
				t.Fatalf("errorhandler configuration is %T, want map[string]string", handler["configuration"])
			}
			for k, want := range map[string]string{
				"halt": "true", "fail": "false", "status": "inactive-region",
			} {
				if config[k] != want {
					t.Errorf("configuration[%q] = %q, want %q", k, config[k], want)
				}
			}
		})
	}
}

// The read side must recognise the same shape, otherwise the round-trip stays
// broken even though the write side now emits it.
func TestConvertCommandsFromJSON_pluginErrorHandler(t *testing.T) {
	for _, tc := range []struct {
		name     string
		nodeStep interface{} // Rundeck answers with a bool or the string "true"
		wantAttr string
	}{
		{"workflow step (bool)", false, "step_plugin"},
		{"node step (bool)", true, "node_step_plugin"},
		{"node step (string)", "true", "node_step_plugin"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			commands := []interface{}{
				map[string]interface{}{
					"exec": "false",
					"errorhandler": map[string]interface{}{
						"type":     "flow-control",
						"nodeStep": tc.nodeStep,
						"configuration": map[string]interface{}{
							"halt": "true",
						},
					},
				},
			}

			list, diags := convertCommandsFromJSON(context.Background(), commands)
			if diags.HasError() {
				t.Fatalf("convertCommandsFromJSON: %v", diags)
			}

			cmdAttrs := list.Elements()[0].(types.Object).Attributes()
			handlers, ok := cmdAttrs["error_handler"].(types.List)
			if !ok || len(handlers.Elements()) != 1 {
				t.Fatalf("error_handler is %v, want one element", cmdAttrs["error_handler"])
			}

			handlerAttrs := handlers.Elements()[0].(types.Object).Attributes()
			plugins, ok := handlerAttrs[tc.wantAttr].(types.List)
			if !ok || plugins.IsNull() || len(plugins.Elements()) != 1 {
				t.Fatalf("%s is %v, want one element", tc.wantAttr, handlerAttrs[tc.wantAttr])
			}

			pluginAttrs := plugins.Elements()[0].(types.Object).Attributes()
			if got := pluginAttrs["type"].(types.String).ValueString(); got != "flow-control" {
				t.Errorf("plugin type = %q, want flow-control", got)
			}
		})
	}
}

// The converter tests above would still have passed while the handler was being
// dropped, because the drop happened on Rundeck's side: it accepted the job,
// stored no handler, and only the read-back revealed it. This acceptance test
// closes that gap by reading the stored job straight from the API.
//
// The workflow is `sequential` on purpose: Rundeck refuses a workflow-step
// handler on a node-oriented workflow ("Error Handlers for Node Steps must also
// be Node Steps"), which is a constraint of the server, not of the provider.
func TestAccJob_pluginErrorHandler(t *testing.T) {
	var jobID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_pluginErrorHandler,
				Check: resource.ComposeTestCheckFunc(
					testAccJobGetID("rundeck_job.error_handler_plugin", &jobID),

					resource.TestCheckResourceAttr("rundeck_job.error_handler_plugin",
						"command.0.error_handler.0.step_plugin.0.type", "flow-control"),
					resource.TestCheckResourceAttr("rundeck_job.error_handler_plugin",
						"command.0.error_handler.0.step_plugin.0.config.halt", "true"),

					// What actually reached Rundeck, which is the point of the fix.
					testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
						sequence, ok := jobData["sequence"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("sequence is %T, want map", jobData["sequence"])
						}
						commands, ok := sequence["commands"].([]interface{})
						if !ok || len(commands) == 0 {
							return fmt.Errorf("sequence.commands is %T, want a non-empty list", sequence["commands"])
						}
						command, ok := commands[0].(map[string]interface{})
						if !ok {
							return fmt.Errorf("command is %T, want map", commands[0])
						}

						handler, ok := command["errorhandler"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("error handler missing from the stored job: %v", command)
						}
						if got := handler["type"]; got != "flow-control" {
							return fmt.Errorf("error handler type = %v, want flow-control", got)
						}

						// A workflow step: it runs once, not per node.
						switch nodeStep := handler["nodeStep"].(type) {
						case bool:
							if nodeStep {
								return fmt.Errorf("error handler nodeStep = true, want false")
							}
						case string:
							if nodeStep == "true" {
								return fmt.Errorf("error handler nodeStep = %q, want false", nodeStep)
							}
						}

						config, ok := handler["configuration"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("error handler configuration is %T, want map", handler["configuration"])
						}
						for key, want := range map[string]string{"halt": "true", "fail": "false"} {
							if got := config[key]; got != want {
								return fmt.Errorf("error handler configuration[%q] = %v, want %q", key, got, want)
							}
						}
						return nil
					}),
				),
			},
		},
	})
}

const testAccJobConfig_pluginErrorHandler = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-error-handler-plugin"
  description = "Plugin error handler test project"

  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "error_handler_plugin" {
  project_name = rundeck_project.test.name
  name = "error-handler-plugin-test"
  description = "Job whose error handler is a plugin step"
  execution_enabled = true

  # Sequential: Rundeck rejects a workflow-step handler on a node-oriented
  # workflow.
  command_ordering_strategy = "sequential"

  command {
    shell_command = "false"

    error_handler {
      step_plugin {
        type = "flow-control"
        config = {
          halt   = "true"
          fail   = "false"
          status = "halted-by-handler"
        }
      }
    }
  }
}
`
