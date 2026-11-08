package rundeck

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// No acceptance test fails without the modifier, so check it is wired.
func TestJobSchema_valuesListDelimiterPlanModifier(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewJobResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	option, ok := resp.Schema.Blocks["option"].(schema.ListNestedBlock)
	if !ok {
		t.Fatalf("option is %T, want schema.ListNestedBlock", resp.Schema.Blocks["option"])
	}
	delimiter, ok := option.NestedObject.Attributes["values_list_delimiter"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("values_list_delimiter is %T, want schema.StringAttribute", option.NestedObject.Attributes["values_list_delimiter"])
	}
	if len(delimiter.PlanModifiers) != 1 {
		t.Fatalf("values_list_delimiter has %d plan modifiers, want 1", len(delimiter.PlanModifiers))
	}
	if _, ok := delimiter.PlanModifiers[0].(optionValuesListDelimiterModifier); !ok {
		t.Errorf("values_list_delimiter plan modifier is %T, want optionValuesListDelimiterModifier", delimiter.PlanModifiers[0])
	}
}

func TestValidateOptionValuesListDelimiter(t *testing.T) {
	choices := func(values ...attr.Value) types.List {
		return types.ListValueMust(types.StringType, values)
	}
	str := types.StringValue

	option := path.Root("option").AtListIndex(3)
	choice := func(i int) path.Path { return option.AtName("value_choices").AtListIndex(i) }

	// wantPath is nil when no error is expected.
	cases := []struct {
		name         string
		delimiter    types.String
		choices      types.List
		wantPath     *path.Path
		wantInDetail string
	}{
		{"default delimiter, no comma", types.StringNull(), choices(str("a"), str("b")), nil, ""},
		{"default delimiter, comma in a choice", types.StringNull(), choices(str("c"), str("a,b")), ptr(choice(1)), `delimiter ","`},
		{"custom delimiter, comma in a choice", str("|"), choices(str("a,b"), str("c")), nil, ""},
		{"custom delimiter in a choice", str("|"), choices(str("x|y")), ptr(choice(0)), `delimiter "|"`},
		{"empty delimiter falls back to a comma", str(""), choices(str("a|b")), nil, ""},
		{"empty delimiter, comma in a choice", str(""), choices(str("a,b")), ptr(choice(0)), `delimiter ","`},
		{"long delimiter, clear of every choice", str("||"), choices(str("a"), str("b|")), nil, ""},
		{"long delimiter formed between two choices", str("||"), choices(str("a|"), str("b")), ptr(option.AtName("value_choices")), `delimiter "||"`},
		{"delimiter without choices", str("|"), types.ListNull(types.StringType), ptr(option.AtName("values_list_delimiter")), `"environment"`},
		{"delimiter with empty choices", str("|"), choices(), ptr(option.AtName("values_list_delimiter")), `"environment"`},
		{"no delimiter, no choices", types.StringNull(), types.ListNull(types.StringType), nil, ""},
		{"delimiter, choices unknown", str("|"), types.ListUnknown(types.StringType), nil, ""},
		{"choices unknown", types.StringNull(), types.ListUnknown(types.StringType), nil, ""},
		{"one choice unknown", types.StringNull(), choices(types.StringUnknown(), str("a")), nil, ""},
		{"delimiter unknown", types.StringUnknown(), choices(str("a,b")), nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs := testOptionAttrs(tc.delimiter)
			attrs["value_choices"] = tc.choices

			diags := validateOptionValuesListDelimiter(option, "environment", attrs)

			if tc.wantPath == nil {
				if diags.HasError() {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return
			}
			if len(diags) != 1 {
				t.Fatalf("got %d diagnostics, want 1: %v", len(diags), diags)
			}
			withPath, ok := diags[0].(diag.DiagnosticWithPath)
			if !ok || !withPath.Path().Equal(*tc.wantPath) {
				t.Errorf("diagnostic %v is not on %s", diags[0], tc.wantPath)
			}
			if !strings.Contains(diags[0].Detail(), tc.wantInDetail) {
				t.Errorf("detail %q does not quote %s", diags[0].Detail(), tc.wantInDetail)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

// The plan follows the option's choices, never prior state.
func TestOptionValuesListDelimiterModifier(t *testing.T) {
	choicesType := tftypes.List{ElementType: tftypes.String}
	cases := []struct {
		name       string
		configured types.String
		choices    tftypes.Value
		want       types.String
	}{
		{"choices", types.StringNull(), tftypes.NewValue(choicesType, []tftypes.Value{tftypes.NewValue(tftypes.String, "a")}), types.StringValue(",")},
		{"choices, one unknown", types.StringNull(), tftypes.NewValue(choicesType, []tftypes.Value{tftypes.NewValue(tftypes.String, tftypes.UnknownValue)}), types.StringValue(",")},
		{"no choices", types.StringNull(), tftypes.NewValue(choicesType, nil), types.StringNull()},
		{"empty choices", types.StringNull(), tftypes.NewValue(choicesType, []tftypes.Value{}), types.StringNull()},
		{"choices unknown", types.StringNull(), tftypes.NewValue(choicesType, tftypes.UnknownValue), types.StringUnknown()},
		{"configured", types.StringValue("|"), tftypes.NewValue(choicesType, []tftypes.Value{tftypes.NewValue(tftypes.String, "a")}), types.StringValue("|")},
		{"configured, unknown", types.StringUnknown(), tftypes.NewValue(choicesType, []tftypes.Value{tftypes.NewValue(tftypes.String, "a")}), types.StringUnknown()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			planned := types.StringUnknown()
			if !tc.configured.IsNull() {
				planned = tc.configured
			}
			req := planmodifier.StringRequest{
				Path:        path.Root("option").AtListIndex(0).AtName("values_list_delimiter"),
				Config:      testOptionChoicesConfig(tc.choices),
				ConfigValue: tc.configured,
				PlanValue:   planned,
				// Left by the option that used to sit at this index.
				StateValue: types.StringValue(";"),
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

			optionValuesListDelimiterFromConfig().PlanModifyString(context.Background(), req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("PlanValue = %s, want %s", resp.PlanValue, tc.want)
			}
		})
	}
}

func testOptionChoicesConfig(choices tftypes.Value) tfsdk.Config {
	optionType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"value_choices": tftypes.List{ElementType: tftypes.String},
	}}
	rootType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"option": tftypes.List{ElementType: optionType},
	}}

	return tfsdk.Config{
		Schema: schema.Schema{Blocks: map[string]schema.Block{
			"option": schema.ListNestedBlock{NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"value_choices": schema.ListAttribute{Optional: true, ElementType: types.StringType},
			}}},
		}},
		Raw: tftypes.NewValue(rootType, map[string]tftypes.Value{
			"option": tftypes.NewValue(tftypes.List{ElementType: optionType}, []tftypes.Value{
				tftypes.NewValue(optionType, map[string]tftypes.Value{"value_choices": choices}),
			}),
		}),
	}
}
