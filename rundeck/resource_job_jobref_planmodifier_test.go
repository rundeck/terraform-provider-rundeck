package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// An unconfigured alias plans the other one's value, or the default.
func TestJobRefNodeStepModifier(t *testing.T) {
	unknown := tftypes.UnknownValue
	cases := []struct {
		name     string
		modifier planmodifier.Bool
		self     any // configured value of the attribute under test: nil or bool
		alias    any // configured value of the other alias: nil, bool or unknown
		want     types.Bool
	}{
		{"alias true", jobRefNodeStepFromConfig(), nil, true, types.BoolValue(true)},
		{"alias false", errorHandlerJobRefNodeStepFromConfig(), nil, false, types.BoolValue(false)},
		{"neither set, command", jobRefNodeStepFromConfig(), nil, nil, types.BoolValue(false)},
		{"neither set, error handler", errorHandlerJobRefNodeStepFromConfig(), nil, nil, types.BoolValue(true)},
		{"alias unknown", jobRefNodeStepFromConfig(), nil, unknown, types.BoolUnknown()},
		{"self set, alias unset", errorHandlerJobRefNodeStepFromConfig(), false, nil, types.BoolValue(false)},
		{"self set, alias opposite", jobRefNodeStepFromConfig(), false, true, types.BoolValue(false)},
	}

	attributes := []struct{ self, other string }{
		{"run_for_each_node", "node_step"},
		{"node_step", "run_for_each_node"},
	}

	for _, attribute := range attributes {
		for _, tc := range cases {
			t.Run(attribute.self+"/"+tc.name, func(t *testing.T) {
				configValue, planValue := types.BoolNull(), types.BoolUnknown()
				if b, ok := tc.self.(bool); ok {
					configValue, planValue = types.BoolValue(b), types.BoolValue(b)
				}
				req := planmodifier.BoolRequest{
					Path:        path.Root("command").AtListIndex(0).AtName("job").AtListIndex(0).AtName(attribute.self),
					Config:      testJobRefNodeStepConfig(map[string]any{attribute.self: tc.self, attribute.other: tc.alias}),
					ConfigValue: configValue,
					PlanValue:   planValue,
				}
				resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}

				tc.modifier.PlanModifyBool(context.Background(), req, resp)

				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
				}
				if !resp.PlanValue.Equal(tc.want) {
					t.Errorf("PlanValue = %s, want %s", resp.PlanValue, tc.want)
				}
			})
		}
	}
}

func testJobRefNodeStepConfig(values map[string]any) tfsdk.Config {
	jobType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"run_for_each_node": tftypes.Bool,
		"node_step":         tftypes.Bool,
	}}
	commandType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"job": tftypes.List{ElementType: jobType},
	}}
	rootType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"command": tftypes.List{ElementType: commandType},
	}}

	job := map[string]tftypes.Value{}
	for name := range jobType.AttributeTypes {
		job[name] = tftypes.NewValue(tftypes.Bool, values[name])
	}

	return tfsdk.Config{
		Schema: schema.Schema{Blocks: map[string]schema.Block{
			"command": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"job": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"run_for_each_node": schema.BoolAttribute{Optional: true, Computed: true},
					"node_step":         schema.BoolAttribute{Optional: true, Computed: true},
				}}},
			}}},
		}},
		Raw: tftypes.NewValue(rootType, map[string]tftypes.Value{
			"command": tftypes.NewValue(tftypes.List{ElementType: commandType}, []tftypes.Value{
				tftypes.NewValue(commandType, map[string]tftypes.Value{
					"job": tftypes.NewValue(tftypes.List{ElementType: jobType}, []tftypes.Value{
						tftypes.NewValue(jobType, job),
					}),
				}),
			}),
		}),
	}
}
