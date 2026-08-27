package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// removeMapKeys is what Update relies on to keep assigned_projects/
// assigned_projects_config in state accurate for the projects it already
// detached via RemoveProjectAssociation, even when a later SaveRunner call
// in the same Update fails. These are plain unit tests for that pure
// function: no live server is needed to exercise the partial-failure
// correction path itself, only the (TF_ACC-gated) end-to-end behavior it
// supports.

func TestRemoveMapKeys_dropsGivenKeys(t *testing.T) {
	ctx := context.Background()

	m, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"proj-a": types.StringValue("read"),
		"proj-b": types.StringValue("write"),
		"proj-c": types.StringValue("admin"),
	})
	if diags.HasError() {
		t.Fatalf("building test map: %v", diags)
	}

	got, diags := removeMapKeys(ctx, m, map[string]struct{}{"proj-b": {}})
	if diags.HasError() {
		t.Fatalf("removeMapKeys: %v", diags)
	}

	elements := got.Elements()
	if _, exists := elements["proj-b"]; exists {
		t.Errorf("proj-b still present, want it dropped")
	}
	if len(elements) != 2 {
		t.Errorf("len(elements) = %d, want 2 (proj-a and proj-c preserved)", len(elements))
	}
	if _, ok := elements["proj-a"]; !ok {
		t.Errorf("proj-a missing, want it preserved")
	}
	if _, ok := elements["proj-c"]; !ok {
		t.Errorf("proj-c missing, want it preserved")
	}
}

func TestRemoveMapKeys_noKeysIsNoOp(t *testing.T) {
	ctx := context.Background()

	m, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"proj-a": types.StringValue("read"),
	})
	if diags.HasError() {
		t.Fatalf("building test map: %v", diags)
	}

	got, diags := removeMapKeys(ctx, m, map[string]struct{}{})
	if diags.HasError() {
		t.Fatalf("removeMapKeys: %v", diags)
	}
	if len(got.Elements()) != 1 {
		t.Errorf("len(elements) = %d, want 1 (unchanged)", len(got.Elements()))
	}
}

func TestRemoveMapKeys_nullMapStaysNull(t *testing.T) {
	ctx := context.Background()

	got, diags := removeMapKeys(ctx, types.MapNull(types.StringType), map[string]struct{}{"proj-a": {}})
	if diags.HasError() {
		t.Fatalf("removeMapKeys: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("got = %v, want it to stay null", got)
	}
}

// removeMapKeys works on any element type, since AssignedProjectsConfig's
// element type is a nested object rather than a plain string.
func TestRemoveMapKeys_nestedObjectElementType(t *testing.T) {
	ctx := context.Background()

	elemType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"access_level": types.StringType,
	}}
	objA, diags := types.ObjectValue(elemType.AttrTypes, map[string]attr.Value{
		"access_level": types.StringValue("read"),
	})
	if diags.HasError() {
		t.Fatalf("building object: %v", diags)
	}

	m, diags := types.MapValue(elemType, map[string]attr.Value{
		"proj-a": objA,
	})
	if diags.HasError() {
		t.Fatalf("building test map: %v", diags)
	}

	got, diags := removeMapKeys(ctx, m, map[string]struct{}{"proj-a": {}})
	if diags.HasError() {
		t.Fatalf("removeMapKeys: %v", diags)
	}
	if len(got.Elements()) != 0 {
		t.Errorf("len(elements) = %d, want 0", len(got.Elements()))
	}
	if got.ElementType(ctx).String() != elemType.String() {
		t.Errorf("element type = %v, want it preserved as %v", got.ElementType(ctx), elemType)
	}
}
