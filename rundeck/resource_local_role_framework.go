package rundeck

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

var (
	_ resource.Resource                = &localRoleResource{}
	_ resource.ResourceWithConfigure   = &localRoleResource{}
	_ resource.ResourceWithImportState = &localRoleResource{}
)

func NewLocalRoleResource() resource.Resource {
	return &localRoleResource{}
}

type localRoleResource struct {
	clients *RundeckClients
}

type localRoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Authority   types.String `tfsdk:"authority"`
	Description types.String `tfsdk:"description"`
	Members     types.Set    `tfsdk:"members"`
}

func (r *localRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_local_role"
}

func (r *localRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck Enterprise local role (requires the local user store auth realm).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the role, assigned by Rundeck.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"authority": schema.StringAttribute{
				Description: "Name of the role (the 'authority' string referenced by ACL policies).",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the role.",
				Optional:    true,
			},
			"members": schema.SetAttribute{
				Description: "Set of local usernames assigned to this role. Membership is managed by resolving usernames to numeric user IDs and diffing against the role's current membership; it does not create user accounts (see rundeck_local_user, when available). Computed because Read always reflects the role's actual live membership, unlike the state-preserved dispatch-setting maps elsewhere in this provider.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *localRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*RundeckClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *RundeckClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Local role management requires API v44+ (Enterprise local user store feature)
	if clients.APIVersion < "44" {
		resp.Diagnostics.AddError(
			"Insufficient API Version",
			fmt.Sprintf("Local role resources require API version 44 or higher (currently configured: %s). Please update your provider configuration with api_version = \"44\" or higher.", clients.APIVersion),
		)
		return
	}

	r.clients = clients
}

// roleIDFromResponse extracts the numeric "id" field from a role create/edit
// response body (an untyped map[string]interface{}, since the SDK has no
// typed request/response models for this endpoint family). JSON numbers
// decode to float64 via encoding/json's default behavior.
func roleIDFromResponse(body map[string]interface{}) (string, error) {
	raw, ok := body["id"]
	if !ok {
		return "", fmt.Errorf("response did not include an id field")
	}
	idFloat, ok := raw.(float64)
	if !ok {
		return "", fmt.Errorf("unexpected type for id field: %T", raw)
	}
	return strconv.FormatInt(int64(idFloat), 10), nil
}

// listLocalUsers returns every local user as a raw map (username, numeric id,
// and roles list), since ApiList1 has no typed response model.
func listLocalUsers(ctx context.Context, client *openapi.APIClient) ([]map[string]interface{}, error) {
	users, _, err := client.UserAPI.ApiList1(ctx).Execute()
	if err != nil {
		if apiErr, ok := err.(*openapi.GenericOpenAPIError); ok && len(apiErr.Body()) > 0 {
			return nil, fmt.Errorf("%s - Response: %s", err.Error(), string(apiErr.Body()))
		}
		return nil, err
	}
	return users, nil
}

// usernameToIDMap resolves every local username to its numeric user ID.
func usernameToIDMap(users []map[string]interface{}) map[string]int64 {
	result := make(map[string]int64)
	for _, u := range users {
		username, ok := u["username"].(string)
		if !ok {
			continue
		}
		idFloat, ok := u["id"].(float64)
		if !ok {
			continue
		}
		result[username] = int64(idFloat)
	}
	return result
}

// membersOfRole scans every local user's embedded roles list for one whose
// numeric id matches roleID, returning the set of member usernames. There is
// no direct "list members of a role" endpoint, so this always requires a full
// user list scan.
func membersOfRole(users []map[string]interface{}, roleID int64) []string {
	// Must be non-nil (not just declared-but-unassigned) so converting to a
	// Terraform Set value produces a known empty set, not null, when no
	// members match - a nil slice here reproduces the same "inconsistent
	// result after apply" class of bug fixed in resource_project_framework.go.
	members := []string{}
	for _, u := range users {
		username, ok := u["username"].(string)
		if !ok {
			continue
		}
		rolesRaw, ok := u["roles"].([]interface{})
		if !ok {
			continue
		}
		for _, roleRaw := range rolesRaw {
			roleMap, ok := roleRaw.(map[string]interface{})
			if !ok {
				continue
			}
			idFloat, ok := roleMap["id"].(float64)
			if !ok {
				continue
			}
			if int64(idFloat) == roleID {
				members = append(members, username)
				break
			}
		}
	}
	sort.Strings(members)
	return members
}

func (r *localRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan localRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx

	body := map[string]interface{}{
		"authority": plan.Authority.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}

	response, httpResp, err := client.UserAPI.ApiCreate(apiCtx).Body(body).Execute()
	if err != nil {
		errorMsg := err.Error()
		if httpResp != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			errorMsg = fmt.Sprintf("%s - Response: %s", err.Error(), string(bodyBytes))
		}
		resp.Diagnostics.AddError(
			"Error creating local role",
			fmt.Sprintf("Could not create local role %s: %s", plan.Authority.ValueString(), errorMsg),
		)
		return
	}

	roleID, err := roleIDFromResponse(response)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating local role",
			fmt.Sprintf("Role %s was created but its ID could not be determined: %s", plan.Authority.ValueString(), err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(roleID)

	// Seed state immediately, before attempting to assign membership below -
	// if that fails partway through, the role must still be tracked in
	// Terraform state (not orphaned server-side with no corresponding state
	// entry), or a future apply/CheckDestroy has no way to find or clean it
	// up. Member resolution failures below only add errors/warnings; they
	// don't overwrite plan.ID/Authority/Description, so this seed remains
	// valid regardless of how the rest of Create concludes.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Assign initial membership, if any.
	if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
		var members []string
		resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(members) > 0 {
			users, err := listLocalUsers(apiCtx, client)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error creating local role",
					fmt.Sprintf("Role %s was created but member usernames could not be resolved: %s", plan.Authority.ValueString(), err.Error()),
				)
				return
			}
			idMap := usernameToIDMap(users)

			var addIDs []int64
			for _, username := range members {
				id, ok := idMap[username]
				if !ok {
					resp.Diagnostics.AddError(
						"Unknown local user",
						fmt.Sprintf("Role %s was created, but member %q does not match any known local user.", plan.Authority.ValueString(), username),
					)
					continue
				}
				addIDs = append(addIDs, id)
			}
			if resp.Diagnostics.HasError() {
				return
			}

			memberBody := map[string]interface{}{
				"add":    addIDs,
				"remove": []int64{},
			}
			_, _, err = client.UserAPI.ApiUpdateMembers(apiCtx, roleID).Body(memberBody).Execute()
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Warning assigning role members",
					fmt.Sprintf("Role %s was created but failed to assign members: %s", plan.Authority.ValueString(), err.Error()),
				)
			}
		}
	}

	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}
	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State = readResp.State
}

func (r *localRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state localRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	roleID := state.ID.ValueString()

	roleInfo, apiResp, err := client.UserAPI.ApiGet(apiCtx, roleID).Execute()
	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading local role",
			fmt.Sprintf("Could not read local role %s: %s", roleID, err.Error()),
		)
		return
	}
	if roleInfo == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if roleInfo.Authority != nil {
		state.Authority = types.StringValue(*roleInfo.Authority)
	}
	if roleInfo.Description != nil {
		state.Description = types.StringValue(*roleInfo.Description)
	} else {
		state.Description = types.StringNull()
	}

	roleIDInt, err := strconv.ParseInt(roleID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading local role",
			fmt.Sprintf("Role ID %s is not numeric: %s", roleID, err.Error()),
		)
		return
	}

	// Determining membership requires listing every local user (there is no
	// "list members of a role" endpoint), which some tokens with role-admin
	// rights may not be authorized to do. Treat that as non-fatal: keep
	// whatever was already in state rather than failing the whole read, so
	// authority/description still refresh correctly.
	users, err := listLocalUsers(apiCtx, client)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Could not verify role membership",
			fmt.Sprintf("Local role %s was read, but its member list could not be verified (listing local users failed: %s). Preserving members as last known.", roleID, err.Error()),
		)
		// If there's no known prior value to fall back to (e.g. this is the
		// Read that follows Create, and members was never configured), an
		// empty set is the only value that can legally be returned - leaving
		// it null/unknown here would fail Terraform's "must be known after
		// apply" contract.
		if state.Members.IsUnknown() || state.Members.IsNull() {
			emptySet, diags := types.SetValueFrom(ctx, types.StringType, []string{})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			state.Members = emptySet
		}
	} else {
		members := membersOfRole(users, roleIDInt)

		membersSet, diags := types.SetValueFrom(ctx, types.StringType, members)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Members = membersSet
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *localRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan localRoleResourceModel
	var state localRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	roleID := state.ID.ValueString()
	plan.ID = state.ID

	body := map[string]interface{}{
		"authority": plan.Authority.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}

	_, apiResp, err := client.UserAPI.ApiEdit(apiCtx, roleID).Body(body).Execute()
	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating local role",
			fmt.Sprintf("Could not update local role %s: %s", roleID, err.Error()),
		)
		return
	}

	// Diff membership and only call the API if something actually changed.
	var priorMembers, plannedMembers []string
	if !state.Members.IsNull() && !state.Members.IsUnknown() {
		resp.Diagnostics.Append(state.Members.ElementsAs(ctx, &priorMembers, false)...)
	}
	if !plan.Members.IsNull() && !plan.Members.IsUnknown() {
		resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &plannedMembers, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	priorSet := make(map[string]bool, len(priorMembers))
	for _, m := range priorMembers {
		priorSet[m] = true
	}
	plannedSet := make(map[string]bool, len(plannedMembers))
	for _, m := range plannedMembers {
		plannedSet[m] = true
	}

	var toAdd, toRemove []string
	for m := range plannedSet {
		if !priorSet[m] {
			toAdd = append(toAdd, m)
		}
	}
	for m := range priorSet {
		if !plannedSet[m] {
			toRemove = append(toRemove, m)
		}
	}

	if len(toAdd) > 0 || len(toRemove) > 0 {
		users, err := listLocalUsers(apiCtx, client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating local role",
				fmt.Sprintf("Could not list local users to update role membership: %s", err.Error()),
			)
			return
		}
		idMap := usernameToIDMap(users)

		resolve := func(usernames []string) []int64 {
			var ids []int64
			for _, username := range usernames {
				id, ok := idMap[username]
				if !ok {
					resp.Diagnostics.AddError(
						"Unknown local user",
						fmt.Sprintf("Member %q does not match any known local user.", username),
					)
					continue
				}
				ids = append(ids, id)
			}
			return ids
		}

		addIDs := resolve(toAdd)
		removeIDs := resolve(toRemove)
		if resp.Diagnostics.HasError() {
			return
		}

		memberBody := map[string]interface{}{
			"add":    addIDs,
			"remove": removeIDs,
		}
		_, _, err = client.UserAPI.ApiUpdateMembers(apiCtx, roleID).Body(memberBody).Execute()
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Warning updating role members",
				fmt.Sprintf("Role %s was updated but failed to update members: %s", roleID, err.Error()),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}
	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State = readResp.State
}

func (r *localRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state localRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	roleID := state.ID.ValueString()

	_, _, err := client.UserAPI.ApiDelete(apiCtx, roleID).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting local role",
			fmt.Sprintf("Could not delete local role %s: %s", roleID, err.Error()),
		)
	}
}

func (r *localRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
