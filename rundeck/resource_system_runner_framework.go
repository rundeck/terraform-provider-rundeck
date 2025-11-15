package rundeck

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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

type systemRunnerResource struct {
	clients *RundeckClients
}

type systemRunnerResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	TagNames            types.String `tfsdk:"tag_names"`
	AssignedProjects    types.Map    `tfsdk:"assigned_projects"`
	ProjectRunnerAsNode types.Map    `tfsdk:"project_runner_as_node"`
	InstallationType    types.String `tfsdk:"installation_type"`
	ReplicaType         types.String `tfsdk:"replica_type"`
	RunnerID            types.String `tfsdk:"runner_id"`
	Token               types.String `tfsdk:"token"`
	DownloadToken       types.String `tfsdk:"download_token"`
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
			"assigned_projects": schema.MapAttribute{
				Description: "Map of assigned projects.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"project_runner_as_node": schema.MapAttribute{
				Description: "Map of projects where runner acts as node.",
				ElementType: types.BoolType,
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
	// Convert to uppercase for API (enum values are LINUX, WINDOWS, DOCKER, KUBERNETES)
	runnerRequest.SetInstallationType(strings.ToUpper(installationType))

	replicaType := plan.ReplicaType.ValueString()
	if replicaType == "" {
		replicaType = "manual"
	}
	// Convert to uppercase for API (enum values are MANUAL, EPHEMERAL)
	runnerRequest.SetReplicaType(strings.ToUpper(replicaType))

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

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
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
		resp.Diagnostics.AddError(
			"Error reading system runner",
			fmt.Sprintf("Could not read system runner %s: %s", runnerId, err.Error()),
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
		// Convert slice to comma-separated string
		tagNames := strings.Join(runnerInfo.TagNames, ",")
		state.TagNames = types.StringValue(tagNames)
	}

	if runnerInfo.Id != nil {
		state.RunnerID = types.StringValue(*runnerInfo.Id)
	}

	// Note: InstallationType and ReplicaType ARE returned by the RunnerInfo API as enums
	// Convert from uppercase enum to lowercase for Terraform state
	if runnerInfo.InstallationType != nil {
		state.InstallationType = types.StringValue(strings.ToLower(string(*runnerInfo.InstallationType)))
	}

	if runnerInfo.ReplicaType != nil {
		state.ReplicaType = types.StringValue(strings.ToLower(string(*runnerInfo.ReplicaType)))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemRunnerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemRunnerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V2
	apiCtx := r.clients.ctx
	runnerId := plan.ID.ValueString()

	// Create save runner request
	saveRequest := openapi.NewSaveProjectRunnerRequest(runnerId)

	// Always set current values
	saveRequest.SetName(plan.Name.ValueString())
	saveRequest.SetDescription(plan.Description.ValueString())

	if !plan.TagNames.IsNull() && !plan.TagNames.IsUnknown() {
		saveRequest.SetTagNames(plan.TagNames.ValueString())
	}

	installationType := plan.InstallationType.ValueString()
	if installationType == "" {
		installationType = "linux"
	}
	// Convert to uppercase for API (enum values are LINUX, WINDOWS, DOCKER, KUBERNETES)
	saveRequest.SetInstallationType(strings.ToUpper(installationType))

	replicaType := plan.ReplicaType.ValueString()
	if replicaType == "" {
		replicaType = "manual"
	}
	// Convert to uppercase for API (enum values are MANUAL, EPHEMERAL)
	saveRequest.SetReplicaType(strings.ToUpper(replicaType))

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
		saveRequest.SetAssignedProjects(projectsMap)
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

