package rundeck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

// scmIntegrationResource implements both rundeck_scm_import and
// rundeck_scm_export, which are identical except for the fixed "integration"
// value ("import"/"export") passed to the shared SCM API calls.
var (
	_ resource.Resource                = &scmIntegrationResource{}
	_ resource.ResourceWithConfigure   = &scmIntegrationResource{}
	_ resource.ResourceWithImportState = &scmIntegrationResource{}
)

func NewScmImportResource() resource.Resource {
	return &scmIntegrationResource{integration: "import"}
}

func NewScmExportResource() resource.Resource {
	return &scmIntegrationResource{integration: "export"}
}

type scmIntegrationResource struct {
	clients     *RundeckClients
	integration string // "import" or "export"
}

type scmIntegrationResourceModel struct {
	ID      types.String `tfsdk:"id"`
	Project types.String `tfsdk:"project"`
	Type    types.String `tfsdk:"type"`
	Config  types.Map    `tfsdk:"config"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (r *scmIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scm_" + r.integration
}

func (r *scmIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Manages a Rundeck project's SCM %s plugin configuration (e.g. git-%s, svn-%s).", r.integration, r.integration, r.integration),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource, in the form \"project:type\".",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				Description: "Name of the project to configure SCM for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: fmt.Sprintf("SCM plugin type name (e.g. \"git-%s\", \"svn-%s\"). Changing this requires replacing the resource, since it amounts to reconfiguring from scratch.", r.integration, r.integration),
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.MapAttribute{
				Description: "Plugin-specific configuration key/value pairs. The set of required/valid keys is dynamic per plugin type - see Rundeck's SCM plugin input schema for the plugin type in use. Reference external secret storage for any credential-like values rather than embedding raw secrets here.",
				Required:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the SCM plugin is currently enabled for the project.",
				Computed:    true,
			},
		},
	}
}

func (r *scmIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildScmSetupBody builds the request body for ApiProjectSetup. Rundeck
// expects the plugin's properties nested under a "config" key
// (confirmed against a live instance - "json: expected 'config' property"
// when the properties are sent flattened at the top level), not as the
// top-level body itself.
func buildScmSetupBody(ctx context.Context, config types.Map) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	configMap := make(map[string]types.String)
	diags.Append(config.ElementsAs(ctx, &configMap, false)...)
	if diags.HasError() {
		return nil, diags
	}

	pluginConfig := make(map[string]interface{})
	for k, v := range configMap {
		pluginConfig[k] = v.ValueString()
	}

	return map[string]interface{}{
		"config": pluginConfig,
	}, diags
}

// scmActionErrorMessage formats a non-successful ScmActionResult into a
// human-readable message, including any per-field validation errors.
func scmActionErrorMessage(result *openapi.ScmActionResult) string {
	msg := "unknown error"
	if result != nil && result.Message != nil {
		msg = *result.Message
	}
	if result != nil && result.ValidationErrors != nil && len(*result.ValidationErrors) > 0 {
		var fields []string
		for k, v := range *result.ValidationErrors {
			fields = append(fields, fmt.Sprintf("%s: %s", k, v))
		}
		sort.Strings(fields)
		msg = fmt.Sprintf("%s (validation errors: %s)", msg, strings.Join(fields, "; "))
	}
	return msg
}

// scmErrorDetail extracts the most useful detail available from an SCM API
// error. On a 400, the generated SDK decodes the response body into a
// ScmActionResult and stores it as the error's Model - but its own
// Error() string only surfaces RFC7807 Title/Detail fields, which
// ScmActionResult doesn't have, so err.Error() alone is just the bare
// status line. Prefer the decoded model's Message/ValidationErrors; fall
// back to the raw response body, then to err.Error().
func scmErrorDetail(err error) string {
	apiErr, ok := err.(*openapi.GenericOpenAPIError)
	if !ok {
		return err.Error()
	}
	if model, ok := apiErr.Model().(openapi.ScmActionResult); ok {
		if detail := scmActionErrorMessage(&model); detail != "unknown error" {
			return fmt.Sprintf("%s - %s", err.Error(), detail)
		}
	}
	if body := apiErr.Body(); len(body) > 0 {
		return fmt.Sprintf("%s - Response: %s", err.Error(), string(body))
	}
	return err.Error()
}

func (r *scmIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scmIntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	project := plan.Project.ValueString()
	pluginType := plan.Type.ValueString()

	configBody, diags := buildScmSetupBody(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setupResult, _, err := client.SCMAPI.ApiProjectSetup(apiCtx, project, r.integration, pluginType).Body(configBody).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error configuring SCM %s plugin", r.integration),
			fmt.Sprintf("Could not configure %s plugin %s for project %s: %s", r.integration, pluginType, project, scmErrorDetail(err)),
		)
		return
	}
	if setupResult != nil && setupResult.Success != nil && !*setupResult.Success {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error configuring SCM %s plugin", r.integration),
			fmt.Sprintf("Rundeck rejected the %s plugin configuration for project %s: %s", r.integration, project, scmActionErrorMessage(setupResult)),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", project, pluginType))

	// Seed state immediately after successful setup, before the enable call
	// below - if enabling fails, the configured plugin must still be tracked
	// in Terraform state rather than left orphaned (same class of bug fixed
	// in rundeck_local_role's Create).
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ApiProjectSetup's own documentation says it configures AND enables the
	// plugin, but ApiProjectEnable is documented as idempotent - call it
	// explicitly anyway rather than relying on that possibly-imprecise
	// description, since it's a harmless no-op if already enabled.
	enableResult, _, err := client.SCMAPI.ApiProjectEnable(apiCtx, project, r.integration, pluginType).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error enabling SCM %s plugin", r.integration),
			fmt.Sprintf("Plugin was configured but could not be enabled for project %s: %s", project, scmErrorDetail(err)),
		)
		return
	}
	if enableResult != nil && enableResult.Success != nil && !*enableResult.Success {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error enabling SCM %s plugin", r.integration),
			fmt.Sprintf("Plugin was configured but Rundeck did not enable it for project %s: %s", project, scmActionErrorMessage(enableResult)),
		)
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

func (r *scmIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scmIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx

	idParts := strings.SplitN(state.ID.ValueString(), ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Expected ID format 'project:type', got: %s", state.ID.ValueString()),
		)
		return
	}
	project := idParts[0]

	config, apiResp, err := client.SCMAPI.ApiProjectConfig(apiCtx, project, r.integration).Execute()
	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading SCM %s configuration", r.integration),
			fmt.Sprintf("Could not read %s configuration for project %s: %s", r.integration, project, scmErrorDetail(err)),
		)
		return
	}
	if config == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Project = types.StringValue(project)
	if config.Type != nil {
		state.Type = types.StringValue(*config.Type)
		state.ID = types.StringValue(fmt.Sprintf("%s:%s", project, *config.Type))
	}
	if config.Enabled != nil {
		state.Enabled = types.BoolValue(*config.Enabled)
	}

	configValues := make(map[string]types.String)
	if config.Config != nil {
		for k, v := range *config.Config {
			configValues[k] = types.StringValue(v)
		}
	}
	configMap, diags := types.MapValueFrom(ctx, types.StringType, configValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Config = configMap

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *scmIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scmIntegrationResourceModel
	var state scmIntegrationResourceModel

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
	plan.ID = state.ID

	idParts := strings.SplitN(state.ID.ValueString(), ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Expected ID format 'project:type', got: %s", state.ID.ValueString()),
		)
		return
	}
	project := idParts[0]
	pluginType := plan.Type.ValueString()

	configBody, diags := buildScmSetupBody(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Re-invoke Setup with the new config - there's no separate "update"
	// endpoint, and Setup's own semantics are configure-or-reconfigure.
	setupResult, _, err := client.SCMAPI.ApiProjectSetup(apiCtx, project, r.integration, pluginType).Body(configBody).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating SCM %s plugin", r.integration),
			fmt.Sprintf("Could not update %s plugin configuration for project %s: %s", r.integration, project, scmErrorDetail(err)),
		)
		return
	}
	if setupResult != nil && setupResult.Success != nil && !*setupResult.Success {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating SCM %s plugin", r.integration),
			fmt.Sprintf("Rundeck rejected the updated %s plugin configuration for project %s: %s", r.integration, project, scmActionErrorMessage(setupResult)),
		)
		return
	}

	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	readReq.State = resp.State
	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State = readResp.State
}

func (r *scmIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scmIntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx

	idParts := strings.SplitN(state.ID.ValueString(), ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Expected ID format 'project:type', got: %s", state.ID.ValueString()),
		)
		return
	}
	project := idParts[0]
	pluginType := state.Type.ValueString()

	// There is no delete/clear-config endpoint for SCM plugins - Disable is
	// the closest available operation. The plugin's configuration may still
	// persist server-side in a disabled state; there's no way from this API
	// to fully remove it.
	_, _, err := client.SCMAPI.ApiProjectDisable(apiCtx, project, r.integration, pluginType).Execute()
	if err != nil {
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("Warning disabling SCM %s plugin", r.integration),
			fmt.Sprintf("Failed to disable %s plugin for project %s: %s", r.integration, project, scmErrorDetail(err)),
		)
	}
}

func (r *scmIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The ID should be in format "project:type"
	idParts := strings.SplitN(req.ID, ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'project:type', got: %s", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), idParts[1])...)
}
