package rundeck

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &passwordResource{}
	_ resource.ResourceWithConfigure   = &passwordResource{}
	_ resource.ResourceWithImportState = &passwordResource{}
)

func NewPasswordResource() resource.Resource {
	return &passwordResource{}
}

type passwordResource struct {
	clients *RundeckClients
}

type passwordResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Path     types.String `tfsdk:"path"`
	Password types.String `tfsdk:"password"`
}

func (r *passwordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password"
}

func (r *passwordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck password in the key storage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the password (same as path).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Path to the key within the key store.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Description: "The password to store.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *passwordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *passwordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan passwordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := plan.Path.ValueString()
	password := plan.Password.ValueString()

	payloadReader := io.NopCloser(strings.NewReader(password))
	_, err := client.StorageKeyCreate(ctx, path, payloadReader, "application/x-rundeck-data-password")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating password",
			fmt.Sprintf("Could not create password at %s: %s", path, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *passwordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state passwordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := state.ID.ValueString()

	keyResp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading password",
			fmt.Sprintf("Could not read password at %s: %s", path, err.Error()),
		)
		return
	}

	if keyResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Check if the key is actually a password
	if keyResp.Meta == nil || keyResp.Meta.RundeckContentType == nil {
		resp.Diagnostics.AddError(
			"Error reading password",
			fmt.Sprintf("Invalid response for key at %s: meta or content type is nil", path),
		)
		return
	}

	if *keyResp.Meta.RundeckContentType != "application/x-rundeck-data-password" {
		// Key exists but is not a password
		resp.State.RemoveResource(ctx)
		return
	}

	// Note: We don't read back the password for security reasons
	// The state will retain the value from the configuration

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *passwordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan passwordResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := plan.Path.ValueString()
	password := plan.Password.ValueString()

	payloadReader := io.NopCloser(strings.NewReader(password))
	_, err := client.StorageKeyUpdate(ctx, path, payloadReader, "application/x-rundeck-data-password")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating password",
			fmt.Sprintf("Could not update password at %s: %s", path, err.Error()),
		)
		return
	}

	// Ensure ID is set (same as path)
	plan.ID = types.StringValue(path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *passwordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state passwordResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := state.ID.ValueString()

	_, err := client.StorageKeyDelete(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting password",
			fmt.Sprintf("Could not delete password at %s: %s", path, err.Error()),
		)
		return
	}
}

func (r *passwordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)

	// Set password to a placeholder hash during import since we can't retrieve it
	hash := sha1.Sum([]byte("imported-password-unknown-content"))
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("password"), hex.EncodeToString(hash[:]))...)
}
