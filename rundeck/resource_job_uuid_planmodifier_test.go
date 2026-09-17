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

// State written before the uuid attribute existed records the job's UUID under
// id alone. Pinning such a job to the UUID it already has must plan as no
// change: treating the absent attribute as a difference would replace every job
// in the estate, taking their execution history with them.
func TestRequiresReplaceOnUUIDChange_priorStateWithoutUUID(t *testing.T) {
	const jobUUID = "6bf08fc5-835e-4dea-a83f-56953879b497"

	cases := []struct {
		name            string
		configured      string
		wantReplace     bool
		wantDiagnostics int
	}{
		{"pinned to the id already in state", jobUUID, false, 0},
		{"pinned to a different uuid", "11111111-2222-3333-4444-555555555555", true, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				Path:        path.Root("uuid"),
				ConfigValue: types.StringValue(tc.configured),
				PlanValue:   types.StringValue(tc.configured),
				StateValue:  types.StringNull(),
				State:       testUUIDState(jobUUID, nil),
				Plan:        tfsdk.Plan(testUUIDState(jobUUID, &tc.configured)),
			}
			resp := &planmodifier.StringResponse{}

			requiresReplaceOnUUIDChange().PlanModifyString(context.Background(), req, resp)

			if resp.RequiresReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantReplace)
			}
			if got := len(resp.Diagnostics.Warnings()); got != tc.wantDiagnostics {
				t.Errorf("warnings = %d, want %d — a replacement must always come with one", got, tc.wantDiagnostics)
			}
		})
	}
}

func testUUIDState(id string, uuid *string) tfsdk.State {
	objectType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":   tftypes.String,
		"uuid": tftypes.String,
	}}

	uuidValue := tftypes.NewValue(tftypes.String, nil)
	if uuid != nil {
		uuidValue = tftypes.NewValue(tftypes.String, *uuid)
	}

	return tfsdk.State{
		Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id":   schema.StringAttribute{Computed: true},
				"uuid": schema.StringAttribute{Optional: true, Computed: true},
			},
		},
		Raw: tftypes.NewValue(objectType, map[string]tftypes.Value{
			"id":   tftypes.NewValue(tftypes.String, id),
			"uuid": uuidValue,
		}),
	}
}
