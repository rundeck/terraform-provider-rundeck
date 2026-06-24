package rundeck

import (
	"context"
	"fmt"
	"io"
	"sort"
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
	_ resource.Resource                = &systemRunnerResource{}
	_ resource.ResourceWithConfigure   = &systemRunnerResource{}
	_ resource.ResourceWithImportState = &systemRunnerResource{}
)

func NewSystemRunnerResource() resource.Resource {
	return &systemRunnerResource{}
}

// normalizeRunnerTags normalizes tag names by converting to lowercase and sorting alphabetically
// to prevent plan drift from API case/order changes
func normalizeRunnerTags(tagString string) string {
	if tagString == "" {
		return ""
	}
	tags := strings.Split(tagString, ",")
	normalizedTags := make([]string, len(tags))
	for i, tag := range tags {
		normalizedTags[i] = strings.ToLower(strings.TrimSpace(tag))
	}
	sort.Strings(normalizedTags)
	return strings.Join(normalizedTags, ",")
}

type systemRunnerResource struct {
	clients *RundeckClients
}

// projectRunnerConfig represents the configuration for a project assignment
// including access level and node dispatch settings
type projectRunnerConfig struct {
	AccessLevel         types.String `tfsdk:"access_level"`
	RunnerAsNodeEnabled types.Bool   `tfsdk:"runner_as_node_enabled"`
	RemoteNodeDispatch  types.Bool   `tfsdk:"remote_node_dispatch"`
	RunnerNodeFilter    types.String `tfsdk:"runner_node_filter"`
}

type systemRunnerResourceModel struct {
	ID                     types.String    `tfsdk:"id"`
	Name                   types.String    `tfsdk:"name"`
	Description            types.String    `tfsdk:"description"`
	TagNames               RunnerTagsValue `tfsdk:"tag_names"`
	AssignedProjects       types.Map       `tfsdk:"assigned_projects"`
	ProjectRunnerAsNode    types.Map       `tfsdk:"project_runner_as_node"`
	AssignedProjectsConfig types.Map       `tfsdk:"assigned_projects_config"`
	InstallationType       types.String    `tfsdk:"installation_type"`
	ReplicaType            types.String    `tfsdk:"replica_type"`
	RunnerID               types.String    `tfsdk:"runner_id"`
	Token                  types.String    `tfsdk:"token"`
	DownloadToken          types.String    `tfsdk:"download_token"`
}

func (r *systemRunnerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_runner"
}

func (r *systemRunnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck system-level runner.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the runner.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				Description: "Comma separated tags for the runner. Rundeck normalizes tags to lowercase and sorts them alphabetically. The provider handles this automatically to prevent plan drift.",
				Optional:    true,
				CustomType:  RunnerTagsType{},
			},
			"assigned_projects": schema.MapAttribute{
				Description: "Map of assigned projects.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"project_runner_as_node": schema.MapAttribute{
				Description:        "Map of projects where runner acts as node.",
				ElementType:        types.BoolType,
				Optional:           true,
				DeprecationMessage: "Use assigned_projects_config instead, which supports full per-project dispatch configuration (runner_as_node_enabled, remote_node_dispatch, runner_node_filter).",
			},
			"assigned_projects_config": schema.MapNestedAttribute{
				Description: "Map of project configurations with full dispatch settings. When a project appears in both assigned_projects and assigned_projects_config, assigned_projects_config takes precedence.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"access_level": schema.StringAttribute{
							Description: "Access level for the project. Valid values: 'read', 'execute', 'admin'.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("read", "execute", "admin"),
							},
						},
						"runner_as_node_enabled": schema.BoolAttribute{
							Description: "Enable the runner to act as a node for this project.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"remote_node_dispatch": schema.BoolAttribute{
							Description: "Enable remote node dispatch for the runner in this project.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"runner_node_filter": schema.StringAttribute{
							Description: "Node filter string for the runner in this project.",
							Optional:    true,
						},
					},
				},
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

func (r *systemRunnerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	// System runner requires API v56+ (Enterprise feature)
	if clients.APIVersion < "56" {
		resp.Diagnostics.AddError(
			"Insufficient API Version",
			fmt.Sprintf("System runner resources require API version 56 or higher (currently configured: %s). Please update your provider configuration with api_version = \"56\" or higher.", clients.APIVersion),
		)
		return
	}

	r.clients = clients
}

func (r *systemRunnerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemRunnerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()

	// For system runners, use CreateProjectRunnerRequest but set fields directly (not via newRunnerRequest)
	// The SDK uses a hybrid request type for both system and project runners
	runnerRequest := openapi.NewCreateProjectRunnerRequest(name, description)

	if !plan.TagNames.IsNull() && !plan.TagNames.IsUnknown() {
		// Send tags as-is to API, Rundeck will normalize them
		runnerRequest.SetTagNames(plan.TagNames.ValueString())
	}

	if !plan.AssignedProjects.IsNull() && !plan.AssignedProjects.IsUnknown() {
		projects := make(map[string]types.String)
		resp.Diagnostics.Append(plan.AssignedProjects.ElementsAs(ctx, &projects, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		projectsMap := make(map[string]string)
		for k, v := range projects {
			projectsMap[k] = v.ValueString()
		}
		runnerRequest.SetAssignedProjects(projectsMap)
	}

	if !plan.ProjectRunnerAsNode.IsNull() && !plan.ProjectRunnerAsNode.IsUnknown() {
		nodes := make(map[string]types.Bool)
		resp.Diagnostics.Append(plan.ProjectRunnerAsNode.ElementsAs(ctx, &nodes, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		nodesMap := make(map[string]bool)
		for k, v := range nodes {
			nodesMap[k] = v.ValueBool()
		}
		runnerRequest.SetProjectRunnerAsNode(nodesMap)
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

	// Create the system runner
	response, _, err := client.RunnerAPI.CreateRunner(apiCtx).CreateProjectRunnerRequest(*runnerRequest).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating system runner",
			fmt.Sprintf("Could not create system runner: %s", err.Error()),
		)
		return
	}

	// Set the ID and computed fields
	if response.RunnerId != nil {
		plan.ID = types.StringValue(*response.RunnerId)
		plan.RunnerID = types.StringValue(*response.RunnerId)
	}

	if response.Token != nil {
		plan.Token = types.StringValue(*response.Token)
	}

	if response.DownloadTk != nil {
		plan.DownloadToken = types.StringValue(*response.DownloadTk)
	}

	// Get runner ID for dispatch settings
	runnerId := plan.RunnerID.ValueString()

	// Handle assigned_projects_config - full dispatch configuration per project
	if !plan.AssignedProjectsConfig.IsNull() && !plan.AssignedProjectsConfig.IsUnknown() {
		projectConfigs := make(map[string]projectRunnerConfig)
		resp.Diagnostics.Append(plan.AssignedProjectsConfig.ElementsAs(ctx, &projectConfigs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Build merged project assignments (assigned_projects_config takes precedence)
		mergedProjects := make(map[string]string)

		// First, add projects from assigned_projects
		if !plan.AssignedProjects.IsNull() && !plan.AssignedProjects.IsUnknown() {
			projects := make(map[string]types.String)
			resp.Diagnostics.Append(plan.AssignedProjects.ElementsAs(ctx, &projects, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			for k, v := range projects {
				mergedProjects[k] = v.ValueString()
			}
		}

		// Then, add/override with assigned_projects_config
		for projectName, config := range projectConfigs {
			mergedProjects[projectName] = config.AccessLevel.ValueString()
		}

		// Apply project assignments if needed (if assigned_projects_config added new projects)
		if len(mergedProjects) > 0 {
			saveRequest := openapi.NewSaveProjectRunnerRequest(runnerId)
			saveRequest.SetName(name)
			saveRequest.SetDescription(description)
			saveRequest.SetAssignedProjects(mergedProjects)

			if !plan.TagNames.IsNull() && !plan.TagNames.IsUnknown() {
				saveRequest.SetTagNames(plan.TagNames.ValueString())
			}

			installationType := plan.InstallationType.ValueString()
			if installationType == "" {
				installationType = "linux"
			}
			saveRequest.SetInstallationType(strings.ToLower(installationType))

			replicaType := plan.ReplicaType.ValueString()
			if replicaType == "" {
				replicaType = "manual"
			}
			saveRequest.SetReplicaType(strings.ToLower(replicaType))

			_, _, err = client.RunnerAPI.SaveRunner(apiCtx, runnerId).SaveProjectRunnerRequest(*saveRequest).Execute()
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Warning assigning projects",
					fmt.Sprintf("Runner created but failed to update project assignments: %s", err.Error()),
				)
			}
		}

		// Apply dispatch settings for each project in assigned_projects_config
		for projectName, config := range projectConfigs {
			// Only apply dispatch settings if any are set
			if !config.RunnerAsNodeEnabled.IsNull() || !config.RemoteNodeDispatch.IsNull() || !config.RunnerNodeFilter.IsNull() {
				nodeDispatchRequest := openapi.NewSaveProjectRunnerNodeDispatchSettingsRequest(runnerId)

				if !config.RunnerAsNodeEnabled.IsNull() && !config.RunnerAsNodeEnabled.IsUnknown() {
					nodeDispatchRequest.SetRunnerAsNodeEnabled(config.RunnerAsNodeEnabled.ValueBool())
				}

				if !config.RemoteNodeDispatch.IsNull() && !config.RemoteNodeDispatch.IsUnknown() {
					nodeDispatchRequest.SetRemoteNodeDispatch(config.RemoteNodeDispatch.ValueBool())
				}

				if !config.RunnerNodeFilter.IsNull() && !config.RunnerNodeFilter.IsUnknown() {
					nodeDispatchRequest.SetRunnerNodeFilter(config.RunnerNodeFilter.ValueString())
				}

				_, _, err := client.RunnerAPI.SaveProjectRunnerNodeDispatchSettings(apiCtx, projectName).SaveProjectRunnerNodeDispatchSettingsRequest(*nodeDispatchRequest).Execute()
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Warning configuring node dispatch",
						fmt.Sprintf("Runner created but failed to configure node dispatch for project %s: %s", projectName, err.Error()),
					)
				}
			}
		}
	}

	// Set state initially
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call Read to get the normalized values from the API
	// This ensures tags and other fields match what Rundeck returns
	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}
	r.Read(ctx, readReq, &readResp)
	resp.Diagnostics.Append(readResp.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State = readResp.State
}

func (r *systemRunnerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemRunnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	runnerId := state.ID.ValueString()

	// Get runner info by ID
	runnerInfo, apiResp, err := client.RunnerAPI.RunnerInfo(apiCtx, runnerId).Execute()

	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		// Include response body in error message for troubleshooting
		var bodyStr string
		if apiResp != nil && apiResp.Body != nil {
			bodyBytes, _ := io.ReadAll(apiResp.Body)
			bodyStr = string(bodyBytes)
		}
		resp.Diagnostics.AddError(
			"Error reading system runner",
			fmt.Sprintf("Could not read system runner %s: %s\nAPI response: %s", runnerId, err.Error(), bodyStr),
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
		state.TagNames = RunnerTagsValue{
			StringValue: types.StringValue(tagNames),
		}
	}

	if runnerInfo.Id != nil {
		// Set both ID and RunnerID to the same value
		state.ID = types.StringValue(*runnerInfo.Id)
		state.RunnerID = types.StringValue(*runnerInfo.Id)
	}

	// Token and DownloadToken are only returned on Create, not on Read
	// Preserve the existing values from state to prevent drift
	// (state.Token and state.DownloadToken are already set from req.State.Get())

	// AssignedProjects, ProjectRunnerAsNode, and AssignedProjectsConfig are also preserved from state
	// The RunnerInfo API does not return per-project dispatch settings, so we rely on state
	// (state.AssignedProjects, state.ProjectRunnerAsNode, state.AssignedProjectsConfig already set from req.State.Get())

	// InstallationType and ReplicaType are returned by the RunnerInfo API
	// With new feature flags, API returns lowercase values directly
	if runnerInfo.InstallationType != nil {
		state.InstallationType = types.StringValue(string(*runnerInfo.InstallationType))
	}

	if runnerInfo.ReplicaType != nil {
		state.ReplicaType = types.StringValue(string(*runnerInfo.ReplicaType))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemRunnerResourceModel
	var state systemRunnerResourceModel

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
	runnerId := state.ID.ValueString()

	// Preserve ID and computed fields from state
	plan.ID = state.ID
	plan.RunnerID = state.RunnerID
	plan.Token = state.Token
	plan.DownloadToken = state.DownloadToken

	// Create save runner request
	saveRequest := openapi.NewSaveProjectRunnerRequest(runnerId)

	// Always set current values
	saveRequest.SetName(plan.Name.ValueString())
	saveRequest.SetDescription(plan.Description.ValueString())

	if !plan.TagNames.IsNull() && !plan.TagNames.IsUnknown() {
		// Send tags as-is to API, Rundeck will normalize them
		saveRequest.SetTagNames(plan.TagNames.ValueString())
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

	// Build merged project assignments (assigned_projects_config takes precedence)
	mergedProjects := make(map[string]string)

	// First, add projects from assigned_projects
	if !plan.AssignedProjects.IsNull() && !plan.AssignedProjects.IsUnknown() {
		projects := make(map[string]types.String)
		resp.Diagnostics.Append(plan.AssignedProjects.ElementsAs(ctx, &projects, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range projects {
			mergedProjects[k] = v.ValueString()
		}
	}

	// Then, add/override with assigned_projects_config
	if !plan.AssignedProjectsConfig.IsNull() && !plan.AssignedProjectsConfig.IsUnknown() {
		projectConfigs := make(map[string]projectRunnerConfig)
		resp.Diagnostics.Append(plan.AssignedProjectsConfig.ElementsAs(ctx, &projectConfigs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for projectName, config := range projectConfigs {
			mergedProjects[projectName] = config.AccessLevel.ValueString()
		}
	}

	// Apply merged projects only when at least one source is known. A known (even empty)
	// map still allows explicit clearing, but an unknown value must not silently wipe all
	// of the runner's project assignments.
	assignedProjectsKnown := !plan.AssignedProjects.IsNull() && !plan.AssignedProjects.IsUnknown()
	assignedProjectsConfigKnown := !plan.AssignedProjectsConfig.IsNull() && !plan.AssignedProjectsConfig.IsUnknown()
	if assignedProjectsKnown || assignedProjectsConfigKnown {
		saveRequest.SetAssignedProjects(mergedProjects)
	}

	// Execute the save request
	_, apiResp, err := client.RunnerAPI.SaveRunner(apiCtx, runnerId).SaveProjectRunnerRequest(*saveRequest).Execute()

	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating system runner",
			fmt.Sprintf("Could not update system runner %s: %s", runnerId, err.Error()),
		)
		return
	}

	// Update dispatch settings for each project in assigned_projects_config
	if !plan.AssignedProjectsConfig.IsNull() && !plan.AssignedProjectsConfig.IsUnknown() {
		projectConfigs := make(map[string]projectRunnerConfig)
		resp.Diagnostics.Append(plan.AssignedProjectsConfig.ElementsAs(ctx, &projectConfigs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for projectName, config := range projectConfigs {
			// Only apply dispatch settings if any are set
			if !config.RunnerAsNodeEnabled.IsNull() || !config.RemoteNodeDispatch.IsNull() || !config.RunnerNodeFilter.IsNull() {
				nodeDispatchRequest := openapi.NewSaveProjectRunnerNodeDispatchSettingsRequest(runnerId)

				if !config.RunnerAsNodeEnabled.IsNull() && !config.RunnerAsNodeEnabled.IsUnknown() {
					nodeDispatchRequest.SetRunnerAsNodeEnabled(config.RunnerAsNodeEnabled.ValueBool())
				}

				if !config.RemoteNodeDispatch.IsNull() && !config.RemoteNodeDispatch.IsUnknown() {
					nodeDispatchRequest.SetRemoteNodeDispatch(config.RemoteNodeDispatch.ValueBool())
				}

				if !config.RunnerNodeFilter.IsNull() && !config.RunnerNodeFilter.IsUnknown() {
					nodeDispatchRequest.SetRunnerNodeFilter(config.RunnerNodeFilter.ValueString())
				}

				_, _, err := client.RunnerAPI.SaveProjectRunnerNodeDispatchSettings(apiCtx, projectName).SaveProjectRunnerNodeDispatchSettingsRequest(*nodeDispatchRequest).Execute()
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Warning updating node dispatch",
						fmt.Sprintf("Runner updated but failed to configure node dispatch for project %s: %s", projectName, err.Error()),
					)
				}
			}
		}
	}

	// Set state with normalized values
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemRunnerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemRunnerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	runnerId := state.ID.ValueString()

	_, err := client.RunnerAPI.DeleteRunner(apiCtx, runnerId).Execute()
	if err != nil {
		// Log warning but don't fail - runner might be auto-cleaned
		resp.Diagnostics.AddWarning(
			"Warning deleting system runner",
			fmt.Sprintf("Failed to delete runner %s: %s. Runner might be automatically cleaned up.", runnerId, err.Error()),
		)
	}
}

func (r *systemRunnerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The ID should be the runner ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("runner_id"), req.ID)...)
}
