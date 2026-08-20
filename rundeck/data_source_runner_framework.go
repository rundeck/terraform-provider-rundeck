package rundeck

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

var (
	_ datasource.DataSource              = &runnerDataSource{}
	_ datasource.DataSourceWithConfigure = &runnerDataSource{}
)

func NewRunnerDataSource() datasource.DataSource {
	return &runnerDataSource{}
}

type runnerDataSource struct {
	clients *RundeckClients
}

type runnerProjectAssociationModel struct {
	ProjectName         types.String `tfsdk:"project_name"`
	NodeFilter          types.String `tfsdk:"node_filter"`
	RunnerAsNodeEnabled types.Bool   `tfsdk:"runner_as_node_enabled"`
	RemoteNodeDispatch  types.Bool   `tfsdk:"remote_node_dispatch"`
	RunnerNodeFilter    types.String `tfsdk:"runner_node_filter"`
}

type runnerDataSourceModel struct {
	ID                  types.String                    `tfsdk:"id"`
	RunnerID            types.String                    `tfsdk:"runner_id"`
	Name                types.String                    `tfsdk:"name"`
	Description         types.String                    `tfsdk:"description"`
	Status              types.String                    `tfsdk:"status"`
	Version             types.String                    `tfsdk:"version"`
	TagNames            RunnerTagsValue                 `tfsdk:"tag_names"`
	RunnerAsNodeEnabled types.Bool                      `tfsdk:"runner_as_node_enabled"`
	RemoteNodeDispatch  types.Bool                      `tfsdk:"remote_node_dispatch"`
	RunnerNodeFilter    types.String                    `tfsdk:"runner_node_filter"`
	Hostname            types.String                    `tfsdk:"hostname"`
	OsFamily            types.String                    `tfsdk:"os_family"`
	ReplicaType         types.String                    `tfsdk:"replica_type"`
	InstallationType    types.String                    `tfsdk:"installation_type"`
	DateCreated         types.String                    `tfsdk:"date_created"`
	LastUpdated         types.String                    `tfsdk:"last_updated"`
	LastCheckin         types.String                    `tfsdk:"last_checkin"`
	Uptime              types.Int64                     `tfsdk:"uptime"`
	RunningOperations   types.Int64                     `tfsdk:"running_operations"`
	ProjectAssociations []runnerProjectAssociationModel `tfsdk:"project_associations"`
}

func (d *runnerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner"
}

func (d *runnerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a single Rundeck runner (system or project scoped).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this data source (same as runner_id).",
				Computed:    true,
			},
			"runner_id": schema.StringAttribute{
				Description: "ID of the runner to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the runner.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the runner.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Current status of the runner.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "Runner software version.",
				Computed:    true,
			},
			"tag_names": schema.StringAttribute{
				Description: "Comma separated tags for the runner.",
				Computed:    true,
				CustomType:  RunnerTagsType{},
			},
			"runner_as_node_enabled": schema.BoolAttribute{
				Description: "Whether the runner acts as a node.",
				Computed:    true,
			},
			"remote_node_dispatch": schema.BoolAttribute{
				Description: "Whether remote node dispatch is enabled for the runner.",
				Computed:    true,
			},
			"runner_node_filter": schema.StringAttribute{
				Description: "Node filter string for the runner.",
				Computed:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "Hostname the runner is running on.",
				Computed:    true,
			},
			"os_family": schema.StringAttribute{
				Description: "Operating system family of the runner host.",
				Computed:    true,
			},
			"replica_type": schema.StringAttribute{
				Description: "Replica type of the runner (manual or ephemeral).",
				Computed:    true,
			},
			"installation_type": schema.StringAttribute{
				Description: "Installation type of the runner (linux, windows, kubernetes, docker).",
				Computed:    true,
			},
			"date_created": schema.StringAttribute{
				Description: "Timestamp the runner was created, in RFC3339 format.",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp the runner was last updated, in RFC3339 format.",
				Computed:    true,
			},
			"last_checkin": schema.StringAttribute{
				Description: "Timestamp of the runner's last check-in.",
				Computed:    true,
			},
			"uptime": schema.Int64Attribute{
				Description: "Runner uptime, in seconds.",
				Computed:    true,
			},
			"running_operations": schema.Int64Attribute{
				Description: "Number of operations currently running on the runner.",
				Computed:    true,
			},
			"project_associations": schema.ListNestedAttribute{
				Description: "Per-project dispatch settings associated with this runner.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_name": schema.StringAttribute{
							Description: "Name of the associated project.",
							Computed:    true,
						},
						"node_filter": schema.StringAttribute{
							Description: "Node filter configured for this project association.",
							Computed:    true,
						},
						"runner_as_node_enabled": schema.BoolAttribute{
							Description: "Whether the runner acts as a node for this project.",
							Computed:    true,
						},
						"remote_node_dispatch": schema.BoolAttribute{
							Description: "Whether remote node dispatch is enabled for this project.",
							Computed:    true,
						},
						"runner_node_filter": schema.StringAttribute{
							Description: "Runner node filter configured for this project.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *runnerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*RundeckClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *RundeckClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Runner management requires API v56+ (Enterprise feature)
	if clients.APIVersion < "56" {
		resp.Diagnostics.AddError(
			"Insufficient API Version",
			fmt.Sprintf("Runner data sources require API version 56 or higher (currently configured: %s). Please update your provider configuration with api_version = \"56\" or higher.", clients.APIVersion),
		)
		return
	}

	d.clients = clients
}

func (d *runnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runnerDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.clients.V2
	apiCtx := d.clients.ctx
	runnerId := config.RunnerID.ValueString()

	// Use the general RunnerInfo endpoint rather than ProjectRunnerInfo, matching
	// the precedent set by rundeck_project_runner (ProjectRunnerInfo is unreliable).
	runnerInfo, apiResp, err := client.RunnerAPI.RunnerInfo(apiCtx, runnerId).Execute()

	if apiResp != nil && apiResp.StatusCode == 404 {
		resp.Diagnostics.AddError(
			"Runner Not Found",
			fmt.Sprintf("No runner found with ID %s.", runnerId),
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading runner",
			fmt.Sprintf("Could not read runner %s: %s", runnerId, err.Error()),
		)
		return
	}

	if runnerInfo == nil {
		resp.Diagnostics.AddError(
			"Runner Not Found",
			fmt.Sprintf("No runner found with ID %s.", runnerId),
		)
		return
	}

	state := runnerDataSourceModel{
		ID:       config.RunnerID,
		RunnerID: config.RunnerID,
	}

	if runnerInfo.Id != nil {
		state.ID = types.StringValue(*runnerInfo.Id)
		state.RunnerID = types.StringValue(*runnerInfo.Id)
	}
	if runnerInfo.Name != nil {
		state.Name = types.StringValue(*runnerInfo.Name)
	}
	if runnerInfo.Description != nil {
		state.Description = types.StringValue(*runnerInfo.Description)
	}
	if runnerInfo.Status != nil {
		state.Status = types.StringValue(*runnerInfo.Status)
	}
	if runnerInfo.Version != nil {
		state.Version = types.StringValue(*runnerInfo.Version)
	}
	if runnerInfo.TagNames != nil {
		state.TagNames = RunnerTagsValue{
			StringValue: types.StringValue(normalizeRunnerTags(strings.Join(runnerInfo.TagNames, ","))),
		}
	} else {
		state.TagNames = RunnerTagsValue{StringValue: types.StringValue("")}
	}
	if runnerInfo.RunnerAsNodeEnabled != nil {
		state.RunnerAsNodeEnabled = types.BoolValue(*runnerInfo.RunnerAsNodeEnabled)
	}
	if runnerInfo.RemoteNodeDispatch != nil {
		state.RemoteNodeDispatch = types.BoolValue(*runnerInfo.RemoteNodeDispatch)
	}
	if runnerInfo.RunnerNodeFilter != nil {
		state.RunnerNodeFilter = types.StringValue(*runnerInfo.RunnerNodeFilter)
	}
	if runnerInfo.Hostname != nil {
		state.Hostname = types.StringValue(*runnerInfo.Hostname)
	}
	if runnerInfo.OsFamily != nil {
		state.OsFamily = types.StringValue(*runnerInfo.OsFamily)
	}
	if runnerInfo.ReplicaType != nil {
		state.ReplicaType = types.StringValue(string(*runnerInfo.ReplicaType))
	}
	if runnerInfo.InstallationType != nil {
		state.InstallationType = types.StringValue(string(*runnerInfo.InstallationType))
	}
	if runnerInfo.DateCreated != nil {
		state.DateCreated = types.StringValue(runnerInfo.DateCreated.Format("2006-01-02T15:04:05Z07:00"))
	}
	if runnerInfo.LastUpdated != nil {
		state.LastUpdated = types.StringValue(runnerInfo.LastUpdated.Format("2006-01-02T15:04:05Z07:00"))
	}
	if runnerInfo.LastCheckin != nil {
		state.LastCheckin = types.StringValue(*runnerInfo.LastCheckin)
	}
	if runnerInfo.Uptime != nil {
		state.Uptime = types.Int64Value(*runnerInfo.Uptime)
	}
	if runnerInfo.RunningOperations != nil {
		state.RunningOperations = types.Int64Value(int64(*runnerInfo.RunningOperations))
	}

	state.ProjectAssociations = flattenRunnerProjectAssociations(runnerInfo.ProjectAssociations)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// flattenRunnerProjectAssociations unions the keys across the API's four
// parallel per-project maps into a single list of per-project objects.
func flattenRunnerProjectAssociations(assoc *openapi.RunnerProjectAssociations) []runnerProjectAssociationModel {
	if assoc == nil {
		return nil
	}

	projectNames := map[string]struct{}{}
	if assoc.ProjectNodeFilters != nil {
		for k := range *assoc.ProjectNodeFilters {
			projectNames[k] = struct{}{}
		}
	}
	if assoc.ProjectRunnerAsNodeEnabled != nil {
		for k := range *assoc.ProjectRunnerAsNodeEnabled {
			projectNames[k] = struct{}{}
		}
	}
	if assoc.ProjectRemoteNodeDispatch != nil {
		for k := range *assoc.ProjectRemoteNodeDispatch {
			projectNames[k] = struct{}{}
		}
	}
	if assoc.ProjectRunnerNodeFilter != nil {
		for k := range *assoc.ProjectRunnerNodeFilter {
			projectNames[k] = struct{}{}
		}
	}

	if len(projectNames) == 0 {
		return nil
	}

	result := make([]runnerProjectAssociationModel, 0, len(projectNames))
	for name := range projectNames {
		item := runnerProjectAssociationModel{ProjectName: types.StringValue(name)}
		if assoc.ProjectNodeFilters != nil {
			if v, ok := (*assoc.ProjectNodeFilters)[name]; ok {
				item.NodeFilter = types.StringValue(v)
			}
		}
		if assoc.ProjectRunnerAsNodeEnabled != nil {
			if v, ok := (*assoc.ProjectRunnerAsNodeEnabled)[name]; ok {
				item.RunnerAsNodeEnabled = types.BoolValue(v)
			}
		}
		if assoc.ProjectRemoteNodeDispatch != nil {
			if v, ok := (*assoc.ProjectRemoteNodeDispatch)[name]; ok {
				item.RemoteNodeDispatch = types.BoolValue(v)
			}
		}
		if assoc.ProjectRunnerNodeFilter != nil {
			if v, ok := (*assoc.ProjectRunnerNodeFilter)[name]; ok {
				item.RunnerNodeFilter = types.StringValue(v)
			}
		}
		result = append(result, item)
	}

	return result
}
