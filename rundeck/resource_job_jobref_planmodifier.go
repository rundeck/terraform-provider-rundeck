package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// These modifiers plan a job reference from itself; UseStateForUnknown would
// read the command that sat at the same list index in prior state.

// jobRefNodeStepFromConfig plans an unset run_for_each_node or node_step from
// its alias, or from the value the provider sends when neither is set.
func jobRefNodeStepFromConfig() planmodifier.Bool {
	return jobRefNodeStepModifier{unset: false}
}

// errorHandlerJobRefNodeStepFromConfig is jobRefNodeStepFromConfig for a
// reference held by an error handler, which has its own default.
func errorHandlerJobRefNodeStepFromConfig() planmodifier.Bool {
	return jobRefNodeStepModifier{unset: errorHandlerJobRefNodeStepDefault}
}

// jobRefNodeStepModifier holds the value used when neither alias is set.
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

	// The modifier sits on both aliases; read the one it is not on.
	jobRef := req.Path.ParentPath()
	alias := jobRef.AtName("node_step")
	if req.Path.Equal(alias) {
		alias = jobRef.AtName("run_for_each_node")
	}

	var aliasValue types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, alias, &aliasValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if aliasValue.IsNull() {
		resp.PlanValue = types.BoolValue(m.unset)
		return
	}
	resp.PlanValue = aliasValue
}

// jobRefIdentityFromConfig plans an unset identity field: null on a reference by
// name; on one by uuid, prior state if it holds the same uuid, else unknown.
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
