package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Plain unit tests (no TF_ACC / live Rundeck required) for two settings the
// provider could not express:
//
//   - an option's values_list_delimiter, the separator Rundeck uses to join the
//     value choices into the option's values list (Option.toMap: `map.values`
//     is always accompanied by `map.valuesListDelimiter`, defaulted to a
//     comma). Without it, a choice containing a comma cannot be represented.
//   - a job's notify_avg_duration_threshold, which decides when the
//     on_avg_duration notification fires. ExecutionJob reads it as `?: "0"` and
//     guards the notification with `jobAverageDurationFinal > 0`, so without a
//     threshold the notification is configured but never fires.

// testOptionObjectType mirrors the option object built by
// convertOptionsFromJSON.
var testOptionObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":                      types.StringType,
		"default_value":             types.StringType,
		"description":               types.StringType,
		"label":                     types.StringType,
		"value_choices":             types.ListType{ElemType: types.StringType},
		"value_choices_url":         types.StringType,
		"required":                  types.BoolType,
		"allow_multiple_values":     types.BoolType,
		"multi_value_delimiter":     types.StringType,
		"values_list_delimiter":     types.StringType,
		"require_predefined_choice": types.BoolType,
		"validation_regex":          types.StringType,
		"obscure_input":             types.BoolType,
		"storage_path":              types.StringType,
		"type":                      types.StringType,
		"is_date":                   types.BoolType,
		"exposed_to_scripts":        types.BoolType,
		"hidden":                    types.BoolType,
		"sort_values":               types.BoolType,
		"date_format":               types.StringType,
	},
}

// testOptionAttrs builds the full attribute set of an option, with only the
// name and the delimiter under test populated.
func testOptionAttrs(delimiter attr.Value) map[string]attr.Value {
	return map[string]attr.Value{
		"name":                      types.StringValue("environment"),
		"default_value":             types.StringNull(),
		"description":               types.StringNull(),
		"label":                     types.StringNull(),
		"value_choices":             types.ListNull(types.StringType),
		"value_choices_url":         types.StringNull(),
		"required":                  types.BoolNull(),
		"allow_multiple_values":     types.BoolNull(),
		"multi_value_delimiter":     types.StringNull(),
		"values_list_delimiter":     delimiter,
		"require_predefined_choice": types.BoolNull(),
		"validation_regex":          types.StringNull(),
		"obscure_input":             types.BoolNull(),
		"storage_path":              types.StringNull(),
		"type":                      types.StringNull(),
		"is_date":                   types.BoolNull(),
		"exposed_to_scripts":        types.BoolNull(),
		"hidden":                    types.BoolNull(),
		"sort_values":               types.BoolNull(),
		"date_format":               types.StringNull(),
	}
}

func TestConvertOptionsToJSON_valuesListDelimiter(t *testing.T) {
	opt, diags := types.ObjectValue(testOptionObjectType.AttrTypes, testOptionAttrs(types.StringValue(" ")))
	if diags.HasError() {
		t.Fatalf("building option object: %v", diags)
	}

	result, diags := convertOptionsToJSON(context.Background(),
		types.ListValueMust(testOptionObjectType, []attr.Value{opt}))
	if diags.HasError() {
		t.Fatalf("convertOptionsToJSON: %v", diags)
	}
	if len(result) != 1 {
		t.Fatalf("got %d options, want 1", len(result))
	}

	optMap, ok := result[0].(map[string]interface{})
	if !ok {
		t.Fatalf("option is %T, want map", result[0])
	}
	if got := optMap["valuesListDelimiter"]; got != " " {
		t.Errorf("valuesListDelimiter = %q, want %q", got, " ")
	}
}

func TestConvertOptionsToJSON_valuesListDelimiterOmittedWhenUnset(t *testing.T) {
	opt, diags := types.ObjectValue(testOptionObjectType.AttrTypes, testOptionAttrs(types.StringNull()))
	if diags.HasError() {
		t.Fatalf("building option object: %v", diags)
	}

	result, diags := convertOptionsToJSON(context.Background(),
		types.ListValueMust(testOptionObjectType, []attr.Value{opt}))
	if diags.HasError() {
		t.Fatalf("convertOptionsToJSON: %v", diags)
	}

	optMap := result[0].(map[string]interface{})
	if v, exists := optMap["valuesListDelimiter"]; exists {
		t.Errorf("valuesListDelimiter = %v, want it omitted when unset", v)
	}
}

func TestConvertOptionsFromJSON_valuesListDelimiter(t *testing.T) {
	options := []interface{}{
		map[string]interface{}{
			"name":                "environment",
			"values":              []interface{}{"eu west", "us east"},
			"valuesListDelimiter": " ",
		},
	}

	result, diags := convertOptionsFromJSON(context.Background(), options)
	if diags.HasError() {
		t.Fatalf("convertOptionsFromJSON: %v", diags)
	}
	if len(result.Elements()) != 1 {
		t.Fatalf("got %d options, want 1", len(result.Elements()))
	}

	attrs := result.Elements()[0].(types.Object).Attributes()
	got, ok := attrs["values_list_delimiter"].(types.String)
	if !ok {
		t.Fatalf("values_list_delimiter is %T, want types.String", attrs["values_list_delimiter"])
	}
	if got.IsNull() || got.ValueString() != " " {
		t.Errorf("values_list_delimiter = %v, want %q", got, " ")
	}
}

func TestPlanToJobJSON_notifyAvgDurationThreshold(t *testing.T) {
	r := &jobResource{}
	plan := &jobResourceModel{
		Name:                       types.StringValue("test-job"),
		ProjectName:                types.StringValue("test-project"),
		Description:                types.StringValue("desc"),
		Command:                    testSingleShellCommandList(t),
		NotifyAvgDurationThreshold: types.StringValue("2h"),
	}

	job, err := r.planToJobJSON(context.Background(), plan)
	if err != nil {
		t.Fatalf("planToJobJSON: %v", err)
	}
	if job.NotifyAvgDurationThreshold != "2h" {
		t.Errorf("notifyAvgDurationThreshold = %q, want %q", job.NotifyAvgDurationThreshold, "2h")
	}
}

func TestPlanToJobJSON_notifyAvgDurationThresholdOmittedWhenUnset(t *testing.T) {
	r := &jobResource{}
	plan := &jobResourceModel{
		Name:                       types.StringValue("test-job"),
		ProjectName:                types.StringValue("test-project"),
		Description:                types.StringValue("desc"),
		Command:                    testSingleShellCommandList(t),
		NotifyAvgDurationThreshold: types.StringNull(),
	}

	job, err := r.planToJobJSON(context.Background(), plan)
	if err != nil {
		t.Fatalf("planToJobJSON: %v", err)
	}
	// Empty means the field carries `omitempty` and never reaches the API,
	// leaving Rundeck's own default in place.
	if job.NotifyAvgDurationThreshold != "" {
		t.Errorf("notifyAvgDurationThreshold = %q, want empty", job.NotifyAvgDurationThreshold)
	}
}

func TestJobJSONToState_notifyAvgDurationThreshold(t *testing.T) {
	r := &jobResource{}
	job := &jobJSON{NotifyAvgDurationThreshold: "30m"}

	state := &jobResourceModel{}
	if err := r.jobJSONToState(context.Background(), job, state); err != nil {
		t.Fatalf("jobJSONToState: %v", err)
	}
	if state.NotifyAvgDurationThreshold.ValueString() != "30m" {
		t.Errorf("notify_avg_duration_threshold = %v, want %q",
			state.NotifyAvgDurationThreshold, "30m")
	}
}

func TestJobJSONAPIToState_notifyAvgDurationThreshold(t *testing.T) {
	r := &jobResource{}
	job := &JobJSON{NotifyAvgDurationThreshold: "150%"}

	state := &jobResourceModel{}
	if err := r.jobJSONAPIToState(context.Background(), job, state); err != nil {
		t.Fatalf("jobJSONAPIToState: %v", err)
	}
	if state.NotifyAvgDurationThreshold.ValueString() != "150%" {
		t.Errorf("notify_avg_duration_threshold = %v, want %q",
			state.NotifyAvgDurationThreshold, "150%")
	}
}
