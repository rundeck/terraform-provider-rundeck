package rundeck

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

var (
	_ resource.Resource                = &projectRunnerResource{}
	_ resource.ResourceWithConfigure   = &projectRunnerResource{}
	_ resource.ResourceWithImportState = &projectRunnerResource{}
)

func NewProjectRunnerResource() resource.Resource {
	return &projectRunnerResource{}
}

type projectRunnerResource struct {
	clients *RundeckClients
}

type projectRunnerResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ProjectName         types.String `tfsdk:"project_name"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	TagNames            types.String `tfsdk:"tag_names"`
	InstallationType    types.String `tfsdk:"installation_type"`
	ReplicaType         types.String `tfsdk:"replica_type"`
	RunnerAsNodeEnabled types.Bool   `tfsdk:"runner_as_node_enabled"`
	RemoteNodeDispatch  types.Bool   `tfsdk:"remote_node_dispatch"`
	RunnerNodeFilter    types.String `tfsdk:"runner_node_filter"`
	RunnerID            types.String `tfsdk:"runner_id"`
	Token               types.String `tfsdk:"token"`
	DownloadToken       types.String `tfsdk:"download_token"`
}

func (r *projectRunnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_runner"
}

func (r *projectRunnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck project-level runner.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the runner.",
				Computed:    true,
			},
			"project_name": schema.StringAttribute{
				Description: "Name of the project where the runner will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the runner.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the runner.",
				Required:    true,
			},
			"tag_names": schema.StringAttribute{
				Description: "Comma separated tags for the runner.",
				Optional:    true,
			},
			"installation_type": schema.StringAttribute{
				Description: "Installation type of the runner (linux, windows, kubernetes, docker).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("linux"),
				Validators: []validator.String{
					stringvalidator.OneOf("linux", "windows", "kubernetes", "docker"),
				},
			},
			"replica_type": schema.StringAttribute{
				Description: "Replica type of the runner (manual or ephemeral).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("manual"),
				Validators: []validator.String{
					stringvalidator.OneOf("manual", "ephemeral"),
				},
			},
			"runner_as_node_enabled": schema.BoolAttribute{
				Description: "Enable the runner to act as a node.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"remote_node_dispatch": schema.BoolAttribute{
				Description: "Enable remote node dispatch for the runner.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"runner_node_filter": schema.StringAttribute{
				Description: "Node filter string for the runner.",
				Optional:    true,
			},
			"runner_id": schema.StringAttribute{
				Description: "ID of the created runner.",
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: "Authentication token for the runner.",
				Computed:    true,
			},
			"download_token": schema.StringAttribute{
				Description: "Download token for the runner package.",
				Computed:    true,
			},
		},
	}
}

func (r *projectRunnerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	// Project runner requires API v56+ (Enterprise feature)
	if clients.APIVersion < "56" {
		resp.Diagnostics.AddError(
			"Insufficient API Version",
			fmt.Sprintf("Project runner resources require API version 56 or higher (currently configured: %s). Please update your provider configuration with api_version = \"56\" or higher.", clients.APIVersion),
		)
		return
	}

	r.clients = clients
}

func (r *projectRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectRunnerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	projectName := plan.ProjectName.ValueString()

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()

	// Create the runner request
	runnerRequest := openapi.NewCreateRunnerRequest(name, description)

	if !plan.TagNames.IsNull() && !plan.TagNames.IsUnknown() {
		// Normalize tags before sending to API and update plan
		normalizedTags := normalizeRunnerTags(plan.TagNames.ValueString())
		plan.TagNames = types.StringValue(normalizedTags)
		runnerRequest.SetTagNames(normalizedTags)
	}

	installationType := plan.InstallationType.ValueString()
	if installationType == "" {
		installationType = "linux"
	}
	// Enum values are lowercase (linux, windows, docker, kubernetes) with new SDK
	runnerRequest.SetInstallationType(strings.ToLower(installationType))

	replicaType := plan.ReplicaType.ValueString()
	if replicaType == "" {
		replicaType = "manual"
	}
	// Enum values are lowercase (manual, ephemeral) with new SDK
	runnerRequest.SetReplicaType(strings.ToLower(replicaType))

	// Create project runner request
	projectRunnerRequest := openapi.NewCreateProjectRunnerRequest(name, description)
	projectRunnerRequest.SetNewRunnerRequest(*runnerRequest)

	// Create the runner for the project
	response, httpResp, err := client.RunnerAPI.CreateProjectRunner(apiCtx, projectName).CreateProjectRunnerRequest(*projectRunnerRequest).Execute()
	if err != nil {
		errorMsg := err.Error()
		if httpResp != nil {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			errorMsg = fmt.Sprintf("%s - Response: %s", err.Error(), string(bodyBytes))
		}
		resp.Diagnostics.AddError(
			"Error creating project runner",
			fmt.Sprintf("Could not create project runner: %s", errorMsg),
		)
		return
	}

	// Get the runner ID for node dispatch configuration
	var runnerId string
	if response.RunnerId != nil {
		runnerId = *response.RunnerId
		// Store composite ID as project:runner_id
		plan.ID = types.StringValue(fmt.Sprintf("%s:%s", projectName, runnerId))
		plan.RunnerID = types.StringValue(runnerId)

		// Debug: Log what we got
		resp.Diagnostics.AddWarning("DEBUG: Runner Created", fmt.Sprintf("Project: %s, RunnerID: %s, Composite ID: %s", projectName, runnerId, plan.ID.ValueString()))
	} else {
		resp.Diagnostics.AddError(
			"Error creating project runner",
			"API did not return a runner ID",
		)
		return
	}

	// Configure node dispatch settings if any are set
	if !plan.RunnerAsNodeEnabled.IsNull() || !plan.RemoteNodeDispatch.IsNull() || !plan.RunnerNodeFilter.IsNull() {
		nodeDispatchRequest := openapi.NewSaveProjectRunnerNodeDispatchSettingsRequest(runnerId)

		if !plan.RunnerAsNodeEnabled.IsNull() {
			nodeDispatchRequest.SetRunnerAsNodeEnabled(plan.RunnerAsNodeEnabled.ValueBool())
		}

		if !plan.RemoteNodeDispatch.IsNull() {
			nodeDispatchRequest.SetRemoteNodeDispatch(plan.RemoteNodeDispatch.ValueBool())
		}

		if !plan.RunnerNodeFilter.IsNull() && !plan.RunnerNodeFilter.IsUnknown() {
			nodeDispatchRequest.SetRunnerNodeFilter(plan.RunnerNodeFilter.ValueString())
		}

		_, _, err := client.RunnerAPI.SaveProjectRunnerNodeDispatchSettings(apiCtx, projectName).SaveProjectRunnerNodeDispatchSettingsRequest(*nodeDispatchRequest).Execute()
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Warning configuring node dispatch",
				fmt.Sprintf("Runner created but failed to configure node dispatch: %s", err.Error()),
			)
		}
	}

	// Set the computed fields
	if response.Token != nil {
		plan.Token = types.StringValue(*response.Token)
	}

	if response.DownloadTk != nil {
		plan.DownloadToken = types.StringValue(*response.DownloadTk)
	}

	// Set state with normalized values
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectRunnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx

	// Parse composite ID (project:runner_id)
	idParts := strings.SplitN(state.ID.ValueString(), ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Expected ID format 'project:runner_id', got: %s", state.ID.ValueString()),
		)
		return
	}

	projectName := idParts[0]
	runnerId := idParts[1]

	// Get runner info - use general RunnerInfo endpoint since ProjectRunnerInfo seems unreliable
	runnerInfo, apiResp, err := client.RunnerAPI.RunnerInfo(apiCtx, runnerId).Execute()

	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading project runner",
			fmt.Sprintf("Could not read project runner %s for project %s: %s", runnerId, projectName, err.Error()),
		)
		return
	}

	if runnerInfo == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Set the attributes
	if runnerInfo.Name != nil {
		state.Name = types.StringValue(*runnerInfo.Name)
	}

	if runnerInfo.Description != nil {
		state.Description = types.StringValue(*runnerInfo.Description)
	}

	if runnerInfo.TagNames != nil {
		// Normalize tags to prevent plan drift from API case/order changes
		tagNames := normalizeRunnerTags(strings.Join(runnerInfo.TagNames, ","))
		state.TagNames = types.StringValue(tagNames)
	}

	if runnerInfo.Id != nil {
		state.RunnerID = types.StringValue(*runnerInfo.Id)
		// Reconstruct composite ID for project runner (project:runner_id)
		state.ID = types.StringValue(fmt.Sprintf("%s:%s", projectName, *runnerInfo.Id))
	}

	// Token and DownloadToken are only returned on Create, not on Read
	// Preserve the existing values from state to prevent drift
	// (state.Token and state.DownloadToken are already set from req.State.Get())

	// With new feature flags, API returns lowercase values directly
	if runnerInfo.InstallationType != nil {
		state.InstallationType = types.StringValue(string(*runnerInfo.InstallationType))
	}

	if runnerInfo.ReplicaType != nil {
		state.ReplicaType = types.StringValue(string(*runnerInfo.ReplicaType))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectRunnerResourceModel
	var state projectRunnerResourceModel

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

	// Preserve ID and computed fields from state
	plan.ID = state.ID
	plan.RunnerID = state.RunnerID
	plan.Token = state.Token
	plan.DownloadToken = state.DownloadToken

	// Parse composite ID (project:runner_id)
	idParts := strings.SplitN(state.ID.ValueString(), ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Expected ID format 'project:runner_id', got: %s", plan.ID.ValueString()),
		)
		return
	}

	projectName := idParts[0]
	runnerId := idParts[1]

	// Create save runner request
	saveRequest := openapi.NewSaveProjectRunnerRequest(runnerId)

	// Always set current values
	saveRequest.SetName(plan.Name.ValueString())
	saveRequest.SetDescription(plan.Description.ValueString())

	if !plan.TagNames.IsNull() && !plan.TagNames.IsUnknown() {
		// Normalize tags before sending to API and update plan
		normalizedTags := normalizeRunnerTags(plan.TagNames.ValueString())
		plan.TagNames = types.StringValue(normalizedTags)
		saveRequest.SetTagNames(normalizedTags)
	}

	installationType := plan.InstallationType.ValueString()
	if installationType == "" {
		installationType = "linux"
	}
	// Enum values are lowercase (linux, windows, docker, kubernetes) with new SDK
	saveRequest.SetInstallationType(strings.ToLower(installationType))

	replicaType := plan.ReplicaType.ValueString()
	if replicaType == "" {
		replicaType = "manual"
	}
	// Enum values are lowercase (manual, ephemeral) with new SDK
	saveRequest.SetReplicaType(strings.ToLower(replicaType))

	// Execute the save request
	_, apiResp, err := client.RunnerAPI.SaveProjectRunner(apiCtx, projectName, runnerId).SaveProjectRunnerRequest(*saveRequest).Execute()

	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project runner",
			fmt.Sprintf("Could not update project runner %s for project %s: %s", runnerId, projectName, err.Error()),
		)
		return
	}

	// Update node dispatch settings if changed
	nodeDispatchRequest := openapi.NewSaveProjectRunnerNodeDispatchSettingsRequest(runnerId)

	if !plan.RunnerAsNodeEnabled.IsNull() {
		nodeDispatchRequest.SetRunnerAsNodeEnabled(plan.RunnerAsNodeEnabled.ValueBool())
	}

	if !plan.RemoteNodeDispatch.IsNull() {
		nodeDispatchRequest.SetRemoteNodeDispatch(plan.RemoteNodeDispatch.ValueBool())
	}

	if !plan.RunnerNodeFilter.IsNull() && !plan.RunnerNodeFilter.IsUnknown() {
		nodeDispatchRequest.SetRunnerNodeFilter(plan.RunnerNodeFilter.ValueString())
	}

	_, _, err = client.RunnerAPI.SaveProjectRunnerNodeDispatchSettings(apiCtx, projectName).SaveProjectRunnerNodeDispatchSettingsRequest(*nodeDispatchRequest).Execute()
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Warning updating node dispatch",
			fmt.Sprintf("Runner updated but failed to configure node dispatch: %s", err.Error()),
		)
	}

	// Set state with normalized values
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectRunnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx

	// Parse composite ID (project:runner_id)
	idParts := strings.SplitN(state.ID.ValueString(), ":", 2)
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Expected ID format 'project:runner_id', got: %s", state.ID.ValueString()),
		)
		return
	}

	projectName := idParts[0]
	runnerId := idParts[1]

	_, err := client.RunnerAPI.DeleteProjectRunner(apiCtx, projectName, runnerId).Execute()
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Warning deleting project runner",
			fmt.Sprintf("Failed to delete runner %s from project %s: %s. Runner might be automatically cleaned up.", runnerId, projectName, err.Error()),
		)
	}
}

func (r *projectRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The ID should be in format "project_name:runner_id"
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Import ID must be in format 'project_name:runner_id', got: %s", req.ID),
		)
		return
	}

	projectName := idParts[0]
	runnerId := idParts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), runnerId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_name"), projectName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("runner_id"), runnerId)...)
}
