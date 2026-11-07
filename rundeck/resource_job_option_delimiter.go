package rundeck

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// optionValuesListDelimiterDefault is the delimiter Rundeck uses when none is set.
const optionValuesListDelimiterDefault = ","

// optionValuesListDelimiterFromConfig plans an unconfigured delimiter from the
// option's choices, not from the option that sat at its index in prior state.
func optionValuesListDelimiterFromConfig() planmodifier.String {
	return optionValuesListDelimiterModifier{}
}

type optionValuesListDelimiterModifier struct{}

func (m optionValuesListDelimiterModifier) Description(_ context.Context) string {
	return fmt.Sprintf("When not configured: %q on an option with value_choices, null on one without, unknown while value_choices is unknown.", optionValuesListDelimiterDefault)
}

func (m optionValuesListDelimiterModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m optionValuesListDelimiterModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	var choices types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("value_choices"), &choices)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if choices.IsUnknown() {
		return
	}

	// Rundeck returns no delimiter for an option without values.
	if len(choices.Elements()) == 0 {
		resp.PlanValue = types.StringNull()
		return
	}

	resp.PlanValue = types.StringValue(optionValuesListDelimiterDefault)
}

// validateOptionValuesListDelimiter rejects choices that Rundeck, joining and
// splitting them on the delimiter in effect, would not return as configured.
func validateOptionValuesListDelimiter(option path.Path, optionName string, attrs map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics

	delimiter, _ := attrs["values_list_delimiter"].(types.String)
	choices, _ := attrs["value_choices"].(types.List)
	if delimiter.IsUnknown() || choices.IsUnknown() {
		return diags
	}

	effective := delimiter.ValueString()
	if effective == "" {
		effective = optionValuesListDelimiterDefault
	}

	for i, element := range choices.Elements() {
		choice, ok := element.(types.String)
		if !ok || choice.IsNull() || choice.IsUnknown() {
			continue
		}
		if strings.Contains(choice.ValueString(), effective) {
			diags.AddAttributeError(
				option.AtName("value_choices").AtListIndex(i),
				"Value choice contains the values list delimiter",
				fmt.Sprintf("Option %q has the choice %q, which contains the values list delimiter %q. Rundeck joins the choices with it and splits them again, so this choice would come back in pieces. Set values_list_delimiter to a separator that no choice contains.", optionName, choice.ValueString(), effective),
			)
		}
	}
	if diags.HasError() {
		return diags
	}

	known := make([]string, 0, len(choices.Elements()))
	for _, element := range choices.Elements() {
		choice, ok := element.(types.String)
		if !ok || choice.IsNull() || choice.IsUnknown() {
			return diags
		}
		known = append(known, choice.ValueString())
	}
	if len(known) > 0 && !slices.Equal(strings.Split(strings.Join(known, effective), effective), known) {
		diags.AddAttributeError(
			option.AtName("value_choices"),
			"Value choices form the values list delimiter",
			fmt.Sprintf("Option %q has choices that form the values list delimiter %q where they are joined. Rundeck joins the choices with it and splits them again, so they would come back cut differently. Set values_list_delimiter to a separator that cannot form across two choices.", optionName, effective),
		)
	}

	return diags
}
