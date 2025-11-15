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
	_ resource.Resource                = &privateKeyResource{}
	_ resource.ResourceWithConfigure   = &privateKeyResource{}
	_ resource.ResourceWithImportState = &privateKeyResource{}
)

func NewPrivateKeyResource() resource.Resource {
	return &privateKeyResource{}
}

type privateKeyResource struct {
	clients *RundeckClients
}

type privateKeyResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Path        types.String `tfsdk:"path"`
	KeyMaterial types.String `tfsdk:"key_material"`
}

func (r *privateKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_key"
}

func (r *privateKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck private key in the key storage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the private key (same as path).",
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
				Description: "The private key material to store, in PEM format.",
				Required:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *privateKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *privateKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan privateKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := plan.Path.ValueString()
	keyMaterial := plan.KeyMaterial.ValueString()

	payloadReader := io.NopCloser(strings.NewReader(keyMaterial))
	_, err := client.StorageKeyCreate(ctx, path, payloadReader, "application/octet-stream")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating private key",
			fmt.Sprintf("Could not create private key at %s: %s", path, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(path)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state privateKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := state.ID.ValueString()

	keyResp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading private key",
			fmt.Sprintf("Could not read private key at %s: %s", path, err.Error()),
		)
		return
	}

	if keyResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Check if the key is actually a private key
	if keyResp.Meta == nil || keyResp.Meta.RundeckContentType == nil {
		resp.Diagnostics.AddError(
			"Error reading private key",
			fmt.Sprintf("Invalid response for key at %s: meta or content type is nil", path),
		)
		return
	}

	if *keyResp.Meta.RundeckContentType != "application/octet-stream" {
		// Key exists but is not a private key
		resp.State.RemoveResource(ctx)
		return
	}

	// Note: We don't read back the key_material for security reasons
	// The state will retain the value from the configuration

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *privateKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan privateKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := plan.Path.ValueString()
	keyMaterial := plan.KeyMaterial.ValueString()

	payloadReader := io.NopCloser(strings.NewReader(keyMaterial))
	_, err := client.StorageKeyUpdate(ctx, path, payloadReader, "application/octet-stream")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating private key",
			fmt.Sprintf("Could not update private key at %s: %s", path, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *privateKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state privateKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	path := state.ID.ValueString()

	_, err := client.StorageKeyDelete(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting private key",
			fmt.Sprintf("Could not delete private key at %s: %s", path, err.Error()),
		)
		return
	}
}

func (r *privateKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("path"), req.ID)...)
	
	// Set key_material to a placeholder hash during import since we can't retrieve it
	hash := sha1.Sum([]byte("imported-key-unknown-content"))
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key_material"), hex.EncodeToString(hash[:]))...)
}

