package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A job reference lives in a list of commands, and several of its attributes
// are Optional + Computed. UseStateForUnknown cannot plan them: it reads prior
// state at the same path, so at the same list index, and a reordered workflow
// hands each reference the values of whichever command used to sit there. The
// modifiers below plan those attributes from the reference itself instead.

// jobRefNodeStepFromConfig plans run_for_each_node and node_step, two aliases
// of the API's nodeStep flag: the one left out of the configuration takes the
// value of the one that is set, and with neither set both take the value the
// provider sends in that case. What is sent and what is read back are thus both
// known at plan time.
func jobRefNodeStepFromConfig() planmodifier.Bool {
	return jobRefNodeStepModifier{unset: false}
}

// errorHandlerJobRefNodeStepFromConfig is jobRefNodeStepFromConfig for a
// reference held by an error handler, which has its own default.
func errorHandlerJobRefNodeStepFromConfig() planmodifier.Bool {
	return jobRefNodeStepModifier{unset: errorHandlerJobRefNodeStepDefault}
}

// jobRefNodeStepModifier holds the value both aliases take when neither is
// configured.
type jobRefNodeStepModifier struct {
	unset bool
}

func (m jobRefNodeStepModifier) Description(_ context.Context) string {
	return fmt.Sprintf("When not configured, takes the value of its alias, or %t.", m.unset)
}

func (m jobRefNodeStepModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jobRefNodeStepModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	var runForEachNode, nodeStep types.Bool
	jobRef := req.Path.ParentPath()
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, jobRef.AtName("run_for_each_node"), &runForEachNode)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, jobRef.AtName("node_step"), &nodeStep)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The alias is set, but to a value only known at apply.
	if runForEachNode.IsUnknown() || nodeStep.IsUnknown() {
		return
	}

	value, explicit := jobRefNodeStepExplicit(map[string]attr.Value{
		"run_for_each_node": runForEachNode,
		"node_step":         nodeStep,
	})
	if !explicit {
		value = m.unset
	}
	resp.PlanValue = types.BoolValue(value)
}

// jobRefIdentityFromConfig plans the attributes that designate the referenced
// job — uuid, name, group_name, project_name — when the configuration leaves
// them out.
//
// A reference by name is stored by Rundeck as it was sent: what is not
// configured is absent, and plans as null. That also clears a value that got
// there by another route, such as a project set on the reference from the GUI.
//
// A reference by uuid is resolved by Rundeck, which may return the name, group
// and project it found. Prior state is reused only when it describes that same
// uuid; otherwise the value is left for apply to settle.
func jobRefIdentityFromConfig() planmodifier.String {
	return jobRefIdentityModifier{}
}

type jobRefIdentityModifier struct{}

func (m jobRefIdentityModifier) Description(_ context.Context) string {
	return "When not configured, is null for a reference by name, and follows the referenced job for a reference by uuid."
}

func (m jobRefIdentityModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jobRefIdentityModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	uuid := req.Path.ParentPath().AtName("uuid")

	var configUUID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, uuid, &configUUID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if configUUID.IsNull() {
		resp.PlanValue = types.StringNull()
		return
	}

	// From here on the plan holds either the value of an unchanged resource or
	// the unknown the framework gives a Computed attribute of a changed one.
	if !req.PlanValue.IsUnknown() || configUUID.IsUnknown() || req.State.Raw.IsNull() {
		return
	}

	var stateUUID types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, uuid, &stateUUID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if stateUUID.Equal(configUUID) {
		resp.PlanValue = req.StateValue
	}
}
