package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ basetypes.StringTypable                    = (*RunnerTagsType)(nil)
	_ basetypes.StringValuableWithSemanticEquals = (*RunnerTagsValue)(nil)
)

// RunnerTagsType is a custom type for runner tags
// It implements semantic equality so tags are considered equal
// regardless of order or case (since Rundeck normalizes them)
type RunnerTagsType struct {
	basetypes.StringType
}

// String returns a human-readable string of the type name
func (t RunnerTagsType) String() string {
	return "RunnerTagsType"
}

// ValueFromString creates a RunnerTagsValue from a StringValue
func (t RunnerTagsType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return RunnerTagsValue{
		StringValue: in,
	}, nil
}

// ValueFromTerraform creates a value from Terraform data
func (t RunnerTagsType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	return RunnerTagsValue{
		StringValue: stringValue,
	}, nil
}

// Equal checks if two types are equal
func (t RunnerTagsType) Equal(o attr.Type) bool {
	other, ok := o.(RunnerTagsType)
	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

// ValueType returns the value type for this type
func (t RunnerTagsType) ValueType(ctx context.Context) attr.Value {
	return RunnerTagsValue{}
}

// RunnerTagsValue is the value type for runner tags
type RunnerTagsValue struct {
	basetypes.StringValue
}

// Type returns the type
func (v RunnerTagsValue) Type(ctx context.Context) attr.Type {
	return RunnerTagsType{}
}

// Equal implements standard equality (delegates to base StringValue)
func (v RunnerTagsValue) Equal(o attr.Value) bool {
	other, ok := o.(RunnerTagsValue)
	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals implements semantic equality for runner tags
// Two tag strings are semantically equal if they normalize to the same value
// This prevents plan drift when Rundeck normalizes tags (lowercase + sorted)
func (v RunnerTagsValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Convert the new value to RunnerTagsValue
	newValue, ok := newValuable.(RunnerTagsValue)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)
		return false, diags
	}

	// Handle null/unknown values with standard equality
	if v.IsNull() || v.IsUnknown() || newValue.IsNull() || newValue.IsUnknown() {
		return v.Equal(newValue), diags
	}

	// Normalize both tag strings and compare
	oldNormalized := normalizeRunnerTags(v.ValueString())
	newNormalized := normalizeRunnerTags(newValue.ValueString())

	return oldNormalized == newNormalized, diags
}

// Validate implements attribute validation (optional)
func (v RunnerTagsValue) ValidateAttribute(ctx context.Context, req xattr.ValidateAttributeRequest, resp *xattr.ValidateAttributeResponse) {
	// No additional validation needed beyond what StringValue provides
}
