package rundeck

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rundeck/go-rundeck/rundeck"
)

var (
	_ resource.Resource                = &publicKeyResource{}
	_ resource.ResourceWithConfigure   = &publicKeyResource{}
	_ resource.ResourceWithImportState = &publicKeyResource{}
)

func NewPublicKeyResource() resource.Resource {
	return &publicKeyResource{}
}

type publicKeyResource struct {
	clients *RundeckClients
}

type publicKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Path        types.String `tfsdk:"path"`
	KeyMaterial types.String `tfsdk:"key_material"`
	URL         types.String `tfsdk:"url"`
	Delete      types.Bool   `tfsdk:"delete"`
}

func (r *publicKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_key"
}

func (r *publicKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck public key in the key storage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the public key (same as path).",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "Path to the key within the key store.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_material": schema.StringAttribute{
				Description: "The public key data to store, in the usual OpenSSH public key file format.",
				Optional:    true,
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "URL at which the key content can be retrieved.",
				Computed:    true,
			},
			"delete": schema.BoolAttribute{
				Description: "True if the key should be deleted when the resource is deleted. Defaults to true if key_material is provided in the configuration.",
				Computed:    true,
			},
		},
	}
}

func (r *publicKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *publicKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan publicKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := plan.Path.ValueString()
	keyMaterial := plan.KeyMaterial.ValueString()

	shouldDelete := false
	if keyMaterial != "" {
		keyMaterialReader := io.NopCloser(strings.NewReader(keyMaterial))
		_, err := client.StorageKeyCreate(ctx, path, keyMaterialReader, "application/pgp-keys")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating public key",
				fmt.Sprintf("Could not create public key at %s: %s", path, err.Error()),
			)
			return
		}
		shouldDelete = true
	}

	plan.ID = types.StringValue(path)
	plan.Delete = types.BoolValue(shouldDelete)

	// Read the key to get URL
	keyResp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading public key after creation",
			fmt.Sprintf("Could not read public key at %s: %s", path, err.Error()),
		)
		return
	}

	if keyResp.URL != nil {
		plan.URL = types.StringValue(*keyResp.URL)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *publicKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state publicKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := state.ID.ValueString()

	keyResp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading public key",
			fmt.Sprintf("Could not read public key at %s: %s", path, err.Error()),
		)
		return
	}

	if keyResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Check if the key is actually a public key
	if keyResp.Meta == nil {
		resp.Diagnostics.AddError(
			"Error reading public key",
			fmt.Sprintf("Invalid response for key at %s: meta is nil", path),
		)
		return
	}

	if keyResp.Meta.RundeckKeyType != rundeck.Public {
		// Key exists but is not a public key
		resp.State.RemoveResource(ctx)
		return
	}

	if keyResp.URL != nil {
		state.URL = types.StringValue(*keyResp.URL)
	}

	// Note: We don't read back the key_material for public keys
	// The state will retain the value from the configuration

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *publicKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan publicKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if key_material changed
	var state publicKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.KeyMaterial.Equal(state.KeyMaterial) {
		client := r.clients.V1
		path := plan.Path.ValueString()
		keyMaterial := plan.KeyMaterial.ValueString()

		keyMaterialReader := io.NopCloser(strings.NewReader(keyMaterial))
		_, err := client.StorageKeyUpdate(ctx, path, keyMaterialReader, "application/pgp-keys")
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating public key",
				fmt.Sprintf("Could not update public key at %s: %s", path, err.Error()),
			)
			return
		}
	}

	// Read the key to get updated URL
	client := r.clients.V1
	path := plan.Path.ValueString()
	keyResp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading public key after update",
			fmt.Sprintf("Could not read public key at %s: %s", path, err.Error()),
		)
		return
	}

	if keyResp.URL != nil {
		plan.URL = types.StringValue(*keyResp.URL)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *publicKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state publicKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only delete if the delete flag is true
	if state.Delete.ValueBool() {
		client := r.clients.V1
		path := state.ID.ValueString()

		_, err := client.StorageKeyDelete(ctx, path)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error deleting public key",
				fmt.Sprintf("Could not delete public key at %s: %s", path, err.Error()),
			)
			return
		}
	}
}

func (r *publicKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
	
	// Set delete to false for imported resources (we didn't create them)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("delete"), false)...)
}

