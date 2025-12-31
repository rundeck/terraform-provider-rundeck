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
// Uses DynamicType as ElementType to bypass schema validation, then normalizes to ObjectType
type NotificationListType struct {
	basetypes.ListType
	ObjectType types.ObjectType // Target ObjectType for normalization
}

// String returns a human-readable string of the type name
func (t NotificationListType) String() string {
	return "NotificationListType"
}

// ValueFromList creates a NotificationListValue from a ListValue
func (t NotificationListType) ValueFromList(ctx context.Context, in basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Normalize the list if it's DynamicType
	if !in.IsNull() && !in.IsUnknown() {
		normalizedList, normDiags := normalizeNotificationObjectsFromDynamic(ctx, in, t.ObjectType)
		diags.Append(normDiags...)
		if !diags.HasError() {
			return NotificationListValue{
				ListValue: normalizedList,
			}, diags
		}
	}

	return NotificationListValue{
		ListValue: in,
	}, diags
}

// ValueFromTerraform creates a value from Terraform data
func (t NotificationListType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	// First convert from DynamicType (or whatever type Terraform provides)
	attrValue, err := t.ListType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}

	listValue, ok := attrValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	// Normalize DynamicType list to ObjectType list with all attributes present
	normalizedList, diags := normalizeNotificationObjectsFromDynamic(ctx, listValue, t.ObjectType)
	if diags.HasError() {
		return nil, fmt.Errorf("error normalizing notification list: %v", diags)
	}

	// normalizedList is already a ListValue with ObjectType, so we can use it directly
	return NotificationListValue{
		ListValue: normalizedList,
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

// normalizeNotificationObjectsFromDynamic converts DynamicType list to ObjectType list
// Handles both DynamicType (from HCL) and ObjectType (from state) inputs
func normalizeNotificationObjectsFromDynamic(ctx context.Context, listValue basetypes.ListValue, targetObjectType types.ObjectType) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if listValue.IsNull() || listValue.IsUnknown() {
		return types.ListNull(targetObjectType), diags
	}

	// Try to extract as Dynamic values first (from HCL parsing)
	var dynamicValues []types.Dynamic
	if err := listValue.ElementsAs(ctx, &dynamicValues, false); err == nil {
		// Convert Dynamic values to Object values
		normalizedNotifications := make([]attr.Value, 0, len(dynamicValues))
		for _, dynVal := range dynamicValues {
			if dynVal.IsNull() || dynVal.IsUnknown() {
				continue
			}

			// Convert Dynamic to Object by converting to tftypes.Value first
			tfValue, err := dynVal.ToTerraformValue(ctx)
			if err == nil {
				// Convert tftypes.Value to Object
				objType := basetypes.ObjectType{AttrTypes: targetObjectType.AttrTypes}
				objAttrValue, err := objType.ValueFromTerraform(ctx, tfValue)
				if err == nil {
					if objValue, ok := objAttrValue.(types.Object); ok {
						normalizedObj, objDiags := normalizeNotificationObject(ctx, objValue, targetObjectType)
						diags.Append(objDiags...)
						if !diags.HasError() {
							normalizedNotifications = append(normalizedNotifications, normalizedObj)
						}
					}
				}
			}
		}

		if len(normalizedNotifications) == 0 {
			return types.ListNull(targetObjectType), diags
		}
		return types.ListValueMust(targetObjectType, normalizedNotifications), diags
	}

	// Fallback: try as Object values (from state)
	var notifications []types.Object
	diags.Append(listValue.ElementsAs(ctx, &notifications, false)...)
	if diags.HasError() {
		return listValue, diags
	}

	// Normalize each notification object to ensure all attributes are present
	normalizedNotifications := make([]attr.Value, len(notifications))
	for i, notif := range notifications {
		normalizedObj, objDiags := normalizeNotificationObject(ctx, notif, targetObjectType)
		diags.Append(objDiags...)
		if diags.HasError() {
			return listValue, diags
		}
		normalizedNotifications[i] = normalizedObj
	}

	// Create normalized list
	if len(normalizedNotifications) == 0 {
		return types.ListNull(targetObjectType), diags
	}
	return types.ListValueMust(targetObjectType, normalizedNotifications), diags
}

// normalizeNotificationObject ensures a notification object has all required attributes
// Missing attributes are filled with null values
func normalizeNotificationObject(ctx context.Context, notif types.Object, targetObjectType types.ObjectType) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	attrs := notif.Attributes()
	normalizedAttrs := make(map[string]attr.Value)

	// Ensure all required attributes are present
	if typeVal, ok := attrs["type"]; ok {
		normalizedAttrs["type"] = typeVal
	} else {
		normalizedAttrs["type"] = types.StringNull()
	}

	if webhookUrls, ok := attrs["webhook_urls"]; ok {
		normalizedAttrs["webhook_urls"] = webhookUrls
	} else {
		normalizedAttrs["webhook_urls"] = types.ListNull(types.StringType)
	}

	if format, ok := attrs["format"]; ok {
		normalizedAttrs["format"] = format
	} else {
		normalizedAttrs["format"] = types.StringNull()
	}

	if httpMethod, ok := attrs["http_method"]; ok {
		normalizedAttrs["http_method"] = httpMethod
	} else {
		normalizedAttrs["http_method"] = types.StringNull()
	}

	if email, ok := attrs["email"]; ok {
		normalizedAttrs["email"] = email
	} else {
		normalizedAttrs["email"] = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"recipients": types.ListType{ElemType: types.StringType},
				"subject":    types.StringType,
				"attach_log": types.BoolType,
			},
		})
	}

	if plugin, ok := attrs["plugin"]; ok {
		normalizedAttrs["plugin"] = plugin
	} else {
		normalizedAttrs["plugin"] = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":   types.StringType,
				"config": types.MapType{ElemType: types.StringType},
			},
		})
	}

	// Create normalized object
	normalizedObj, objDiags := types.ObjectValue(targetObjectType.AttrTypes, normalizedAttrs)
	diags.Append(objDiags...)
	return normalizedObj, diags
}

// normalizeNotificationObjects ensures all notification objects have all required attributes
// Missing attributes are filled with null values
// DEPRECATED: Use normalizeNotificationObjectsFromDynamic instead
func normalizeNotificationObjects(ctx context.Context, listValue basetypes.ListValue) (basetypes.ListValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	if listValue.IsNull() || listValue.IsUnknown() {
		return listValue, diags
	}

	// Define the complete notification object type
	notificationObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"webhook_urls": types.ListType{ElemType: types.StringType},
			"format":       types.StringType,
			"http_method":  types.StringType,
			"email": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"recipients": types.ListType{ElemType: types.StringType},
						"subject":    types.StringType,
						"attach_log": types.BoolType,
					},
				},
			},
			"plugin": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					},
				},
			},
		},
	}

	var notifications []types.Object
	diags.Append(listValue.ElementsAs(ctx, &notifications, false)...)
	if diags.HasError() {
		return listValue, diags
	}

	// Normalize each notification object to ensure all attributes are present
	normalizedNotifications := make([]attr.Value, len(notifications))
	for i, notif := range notifications {
		attrs := notif.Attributes()
		normalizedAttrs := make(map[string]attr.Value)

		// Ensure all required attributes are present
		if typeVal, ok := attrs["type"]; ok {
			normalizedAttrs["type"] = typeVal
		} else {
			normalizedAttrs["type"] = types.StringNull()
		}

		if webhookUrls, ok := attrs["webhook_urls"]; ok {
			normalizedAttrs["webhook_urls"] = webhookUrls
		} else {
			normalizedAttrs["webhook_urls"] = types.ListNull(types.StringType)
		}

		if format, ok := attrs["format"]; ok {
			normalizedAttrs["format"] = format
		} else {
			normalizedAttrs["format"] = types.StringNull()
		}

		if httpMethod, ok := attrs["http_method"]; ok {
			normalizedAttrs["http_method"] = httpMethod
		} else {
			normalizedAttrs["http_method"] = types.StringNull()
		}

		if email, ok := attrs["email"]; ok {
			normalizedAttrs["email"] = email
		} else {
			normalizedAttrs["email"] = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"recipients": types.ListType{ElemType: types.StringType},
					"subject":    types.StringType,
					"attach_log": types.BoolType,
				},
			})
		}

		if plugin, ok := attrs["plugin"]; ok {
			normalizedAttrs["plugin"] = plugin
		} else {
			normalizedAttrs["plugin"] = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":   types.StringType,
					"config": types.MapType{ElemType: types.StringType},
				},
			})
		}

		// Create normalized object
		normalizedObj, objDiags := types.ObjectValue(notificationObjectType.AttrTypes, normalizedAttrs)
		diags.Append(objDiags...)
		if diags.HasError() {
			return listValue, diags
		}
		normalizedNotifications[i] = normalizedObj
	}

	// Create normalized list
	normalizedList := types.ListValueMust(notificationObjectType, normalizedNotifications)
	return normalizedList, diags
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
