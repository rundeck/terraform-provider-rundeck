package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rundeck/go-rundeck/rundeck"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &aclPolicyResource{}
	_ resource.ResourceWithConfigure   = &aclPolicyResource{}
	_ resource.ResourceWithImportState = &aclPolicyResource{}
)

// NewAclPolicyResource is a helper function to simplify the provider implementation.
func NewAclPolicyResource() resource.Resource {
	return &aclPolicyResource{}
}

// aclPolicyResource is the resource implementation.
type aclPolicyResource struct {
	clients *RundeckClients
}

// aclPolicyResourceModel describes the resource data model.
type aclPolicyResourceModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Policy types.String `tfsdk:"policy"`
}

// Metadata returns the resource type name.
func (r *aclPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_policy"
}

// Schema defines the schema for the resource.
func (r *aclPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck ACL Policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL policy (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for the ACL policy.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"policy": schema.StringAttribute{
				Description: "YAML formatted ACL Policy string.",
				Required:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *aclPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.clients = clients
}

// Create creates the resource and sets the initial Terraform state.
func (r *aclPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan aclPolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create ACL policy
	client := r.clients.V1
	name := plan.Name.ValueString()
	policy := plan.Policy.ValueString()

	request := &rundeck.SystemACLPolicyCreateRequest{
		Contents: &policy,
	}

	response, err := client.SystemACLPolicyCreate(ctx, name, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ACL policy",
			fmt.Sprintf("Could not create ACL policy %s: %s", name, err.Error()),
		)
		return
	}

	if response.StatusCode == 409 || response.StatusCode == 400 {
		resp.Diagnostics.AddError(
			"Error creating ACL policy",
			fmt.Sprintf("API returned error status %d: %v", response.StatusCode, response.Value),
		)
		return
	}

	// Set the ID
	plan.ID = types.StringValue(name)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *aclPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state aclPolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get ACL policy from Rundeck
	client := r.clients.V1
	name := state.ID.ValueString()

	response, err := client.SystemACLPolicyGet(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ACL policy",
			fmt.Sprintf("Could not read ACL policy %s: %s", name, err.Error()),
		)
		return
	}

	if response.StatusCode == 404 {
		// Resource no longer exists, remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	// Update the state
	if response.Contents != nil {
		state.Policy = types.StringValue(*response.Contents)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *aclPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan aclPolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update ACL policy
	client := r.clients.V1
	name := plan.Name.ValueString()
	policy := plan.Policy.ValueString()

	request := &rundeck.SystemACLPolicyUpdateRequest{
		Contents: &policy,
	}

	_, err := client.SystemACLPolicyUpdate(ctx, name, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating ACL policy",
			fmt.Sprintf("Could not update ACL policy %s: %s", name, err.Error()),
		)
		return
	}

	// Ensure ID is set (same as name)
	plan.ID = types.StringValue(name)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *aclPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state aclPolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete ACL policy
	client := r.clients.V1
	name := state.ID.ValueString()

	_, err := client.SystemACLPolicyDelete(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ACL policy",
			fmt.Sprintf("Could not delete ACL policy %s: %s", name, err.Error()),
		)
		return
	}
}

// ImportState imports the resource into Terraform state.
func (r *aclPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the ID as both ID and name
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
