package rundeck

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ basetypes.ListTypable                    = (*NotificationListType)(nil)
	_ basetypes.ListValuableWithSemanticEquals = (*NotificationListValue)(nil)
)

// NotificationListType is a custom type for notification lists
// It implements semantic equality so notification lists are considered equal
// regardless of order (since Rundeck sorts them alphabetically)
type NotificationListType struct {
	basetypes.ListType
}

// String returns a human-readable string of the type name
func (t NotificationListType) String() string {
	return "NotificationListType"
}

// ValueFromList creates a NotificationListValue from a ListValue
func (t NotificationListType) ValueFromList(ctx context.Context, in basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return NotificationListValue{
		ListValue: in,
	}, nil
}

// ValueFromTerraform creates a value from Terraform data
func (t NotificationListType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.ListType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	listValue, ok := attrValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	return NotificationListValue{
		ListValue: listValue,
	}, nil
}

// Equal checks if two types are equal
func (t NotificationListType) Equal(o attr.Type) bool {
	other, ok := o.(NotificationListType)
	if !ok {
		return false
	}

	return t.ListType.Equal(other.ListType)
}

// ValueType returns the value type for this type
func (t NotificationListType) ValueType(ctx context.Context) attr.Value {
	return NotificationListValue{}
}

// NotificationListValue is the value type for notification lists
type NotificationListValue struct {
	basetypes.ListValue
}

// Type returns the type
func (v NotificationListValue) Type(ctx context.Context) attr.Type {
	return NotificationListType{}
}

// Equal implements standard equality (delegates to base ListValue)
func (v NotificationListValue) Equal(o attr.Value) bool {
	other, ok := o.(NotificationListValue)
	if !ok {
		return false
	}

	return v.ListValue.Equal(other.ListValue)
}

// ListSemanticEquals implements semantic equality for notification lists
// Two notification lists are semantically equal if they contain the same notifications
// regardless of order (since Rundeck sorts them alphabetically)
func (v NotificationListValue) ListSemanticEquals(ctx context.Context, newValuable basetypes.ListValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Convert the new value to NotificationListValue
	newValue, ok := newValuable.(NotificationListValue)
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

	// Extract and normalize both lists
	oldNormalized, oldDiags := normalizeNotificationList(ctx, v.ListValue)
	diags.Append(oldDiags...)

	newNormalized, newDiags := normalizeNotificationList(ctx, newValue.ListValue)
	diags.Append(newDiags...)

	if diags.HasError() {
		return false, diags
	}

	// Compare normalized lists (both should be sorted by type)
	return oldNormalized.Equal(newNormalized), diags
}

// normalizeNotificationList sorts notifications by type for consistent comparison
func normalizeNotificationList(ctx context.Context, listValue basetypes.ListValue) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if listValue.IsNull() || listValue.IsUnknown() {
		return listValue, diags
	}

	var notifications []types.Object
	diags.Append(listValue.ElementsAs(ctx, &notifications, false)...)
	if diags.HasError() {
		return listValue, diags
	}

	// Sort notifications alphabetically by type (same logic as convertNotificationsToJSON)
	sort.Slice(notifications, func(i, j int) bool {
		var typeI, typeJ string

		// Safely extract type from notification i
		if attrI, ok := notifications[i].Attributes()["type"]; ok {
			if s, ok := attrI.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				typeI = s.ValueString()
			}
		}

		// Safely extract type from notification j
		if attrJ, ok := notifications[j].Attributes()["type"]; ok {
			if s, ok := attrJ.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				typeJ = s.ValueString()
			}
		}

		// Treat notifications with invalid or missing types as last in sort order
		if typeI == "" {
			typeI = "\uffff" // Unicode max character - sorts last
		}
		if typeJ == "" {
			typeJ = "\uffff"
		}

		return typeI < typeJ
	})

	// Get the element type from the original list
	elementType := listValue.ElementType(ctx)

	// Convert []types.Object to []attr.Value (types.Object implements attr.Value)
	notificationValues := make([]attr.Value, len(notifications))
	for i, notif := range notifications {
		notificationValues[i] = notif
	}

	// Create a new sorted list
	sortedList := types.ListValueMust(elementType, notificationValues)

	return sortedList, diags
}

