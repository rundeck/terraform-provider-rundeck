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
	_ datasource.DataSource              = &runnersDataSource{}
	_ datasource.DataSourceWithConfigure = &runnersDataSource{}
)

func NewRunnersDataSource() datasource.DataSource {
	return &runnersDataSource{}
}

type runnersDataSource struct {
	clients *RundeckClients
}

type runnerProviderModel struct {
	Provider    types.String `tfsdk:"provider"`
	ServiceName types.String `tfsdk:"service_name"`
	PluginName  types.String `tfsdk:"plugin_name"`
}

type runnerSummaryModel struct {
	ID                    types.String          `tfsdk:"id"`
	Name                  types.String          `tfsdk:"name"`
	Description           types.String          `tfsdk:"description"`
	Status                types.String          `tfsdk:"status"`
	Version               types.String          `tfsdk:"version"`
	AssociatedProjects    types.Int64           `tfsdk:"associated_projects"`
	LastCheckin           types.String          `tfsdk:"last_checkin"`
	RunnerAsNodeEnabled   types.Bool            `tfsdk:"runner_as_node_enabled"`
	TagNames              RunnerTagsValue       `tfsdk:"tag_names"`
	RunnerNodeFilter      types.String          `tfsdk:"runner_node_filter"`
	RemoteNodeDispatch    types.Bool            `tfsdk:"remote_node_dispatch"`
	Hostname              types.String          `tfsdk:"hostname"`
	OsFamily              types.String          `tfsdk:"os_family"`
	ReplicaType           types.String          `tfsdk:"replica_type"`
	InstallationType      types.String          `tfsdk:"installation_type"`
	Providers             []runnerProviderModel `tfsdk:"providers"`
	RunnerReplicas        types.Int64           `tfsdk:"runner_replicas"`
	HealthyRunnerReplicas types.Int64           `tfsdk:"healthy_runner_replicas"`
	RunningOperations     types.Int64           `tfsdk:"running_operations"`
	Uptime                types.Int64           `tfsdk:"uptime"`
	DateCreated           types.String          `tfsdk:"date_created"`
	LastUpdated           types.String          `tfsdk:"last_updated"`
}

type runnersDataSourceModel struct {
	ID          types.String         `tfsdk:"id"`
	ProjectName types.String         `tfsdk:"project_name"`
	Tags        types.String         `tfsdk:"tags"`
	Status      types.String         `tfsdk:"status"`
	Filter      types.String         `tfsdk:"filter"`
	LocalOnly   types.Bool           `tfsdk:"local_only"`
	Runners     []runnerSummaryModel `tfsdk:"runners"`
}

func (d *runnersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runners"
}

func (d *runnersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	runnerSummaryAttributes := map[string]schema.Attribute{
		"id":          schema.StringAttribute{Computed: true, Description: "ID of the runner."},
		"name":        schema.StringAttribute{Computed: true, Description: "Name of the runner."},
		"description": schema.StringAttribute{Computed: true, Description: "Description of the runner."},
		"status":      schema.StringAttribute{Computed: true, Description: "Current status of the runner."},
		"version":     schema.StringAttribute{Computed: true, Description: "Runner software version."},
		"associated_projects": schema.Int64Attribute{
			Computed:    true,
			Description: "Number of projects this runner is associated with.",
		},
		"last_checkin":           schema.StringAttribute{Computed: true, Description: "Timestamp of the runner's last check-in."},
		"runner_as_node_enabled": schema.BoolAttribute{Computed: true, Description: "Whether the runner acts as a node."},
		"tag_names": schema.StringAttribute{
			Computed:    true,
			Description: "Comma separated tags for the runner.",
			CustomType:  RunnerTagsType{},
		},
		"runner_node_filter":   schema.StringAttribute{Computed: true, Description: "Node filter string for the runner."},
		"remote_node_dispatch": schema.BoolAttribute{Computed: true, Description: "Whether remote node dispatch is enabled."},
		"hostname":             schema.StringAttribute{Computed: true, Description: "Hostname the runner is running on."},
		"os_family":            schema.StringAttribute{Computed: true, Description: "Operating system family of the runner host."},
		"replica_type":         schema.StringAttribute{Computed: true, Description: "Replica type of the runner (manual or ephemeral)."},
		"installation_type":    schema.StringAttribute{Computed: true, Description: "Installation type of the runner."},
		"providers": schema.ListNestedAttribute{
			Computed:    true,
			Description: "Plugin providers (node/step/etc) this runner handles.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"provider":     schema.StringAttribute{Computed: true, Description: "Provider name."},
					"service_name": schema.StringAttribute{Computed: true, Description: "Service name."},
					"plugin_name":  schema.StringAttribute{Computed: true, Description: "Plugin name."},
				},
			},
		},
		"runner_replicas":         schema.Int64Attribute{Computed: true, Description: "Number of replicas registered for this runner."},
		"healthy_runner_replicas": schema.Int64Attribute{Computed: true, Description: "Number of healthy replicas registered for this runner."},
		"running_operations":      schema.Int64Attribute{Computed: true, Description: "Number of operations currently running on the runner."},
		"uptime":                  schema.Int64Attribute{Computed: true, Description: "Runner uptime, in seconds."},
		"date_created":            schema.StringAttribute{Computed: true, Description: "Timestamp the runner was created, in RFC3339 format."},
		"last_updated":            schema.StringAttribute{Computed: true, Description: "Timestamp the runner was last updated, in RFC3339 format."},
	}

	resp.Schema = schema.Schema{
		Description: "Lists Rundeck runners, optionally scoped to a project and filtered by tags/status/filter string.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this data source, derived from the given filters.",
				Computed:    true,
			},
			"project_name": schema.StringAttribute{
				Description: "If set, list runners assigned to this project instead of all system runners.",
				Optional:    true,
			},
			"tags": schema.StringAttribute{
				Description: "Comma separated list of tags to filter runners by.",
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "Filter runners by status.",
				Optional:    true,
			},
			"filter": schema.StringAttribute{
				Description: "Generic filter string passed through to the Rundeck API.",
				Optional:    true,
			},
			"local_only": schema.BoolAttribute{
				Description: "If true, only include the local runner.",
				Optional:    true,
			},
			"runners": schema.ListNestedAttribute{
				Description: "Runners matching the given filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: runnerSummaryAttributes,
				},
			},
		},
	}
}

func (d *runnersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	if !requireMinAPIVersion(&resp.Diagnostics, clients.APIVersion, 56, "Runner data sources") {
		return
	}

	d.clients = clients
}

func (d *runnersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runnersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.clients.V2
	apiCtx := d.clients.ctx

	var runnerList *openapi.RunnerList
	var err error

	if !config.ProjectName.IsNull() && !config.ProjectName.IsUnknown() && config.ProjectName.ValueString() != "" {
		listReq := client.RunnerAPI.ListProjectRunners(apiCtx, config.ProjectName.ValueString())
		if !config.Tags.IsNull() && !config.Tags.IsUnknown() {
			listReq = listReq.Tags(config.Tags.ValueString())
		}
		if !config.Status.IsNull() && !config.Status.IsUnknown() {
			listReq = listReq.Status(config.Status.ValueString())
		}
		if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
			listReq = listReq.Filter(config.Filter.ValueString())
		}
		if !config.LocalOnly.IsNull() && !config.LocalOnly.IsUnknown() {
			listReq = listReq.LocalOnly(config.LocalOnly.ValueBool())
		}
		runnerList, _, err = listReq.Execute()
	} else {
		listReq := client.RunnerAPI.ListRunners(apiCtx)
		if !config.Tags.IsNull() && !config.Tags.IsUnknown() {
			listReq = listReq.Tags(config.Tags.ValueString())
		}
		if !config.Status.IsNull() && !config.Status.IsUnknown() {
			listReq = listReq.Status(config.Status.ValueString())
		}
		if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
			listReq = listReq.Filter(config.Filter.ValueString())
		}
		if !config.LocalOnly.IsNull() && !config.LocalOnly.IsUnknown() {
			listReq = listReq.LocalOnly(config.LocalOnly.ValueBool())
		}
		runnerList, _, err = listReq.Execute()
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing runners",
			fmt.Sprintf("Could not list runners: %s", err.Error()),
		)
		return
	}

	state := config
	state.Runners = nil

	idParts := []string{}
	for _, v := range []types.String{config.ProjectName, config.Tags, config.Status, config.Filter} {
		if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			idParts = append(idParts, v.ValueString())
		}
	}
	if !config.LocalOnly.IsNull() && !config.LocalOnly.IsUnknown() && config.LocalOnly.ValueBool() {
		idParts = append(idParts, "local_only")
	}
	if len(idParts) == 0 {
		state.ID = types.StringValue("all")
	} else {
		state.ID = types.StringValue(strings.Join(idParts, "/"))
	}

	if runnerList != nil {
		for _, summary := range runnerList.Runners {
			state.Runners = append(state.Runners, flattenRunnerSummary(summary))
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func flattenRunnerSummary(summary openapi.RunnerSummary) runnerSummaryModel {
	item := runnerSummaryModel{
		TagNames: RunnerTagsValue{StringValue: types.StringValue("")},
	}

	if summary.Id != nil {
		item.ID = types.StringValue(*summary.Id)
	}
	if summary.Name != nil {
		item.Name = types.StringValue(*summary.Name)
	}
	if summary.Description != nil {
		item.Description = types.StringValue(*summary.Description)
	}
	if summary.Status != nil {
		item.Status = types.StringValue(*summary.Status)
	}
	if summary.Version != nil {
		item.Version = types.StringValue(*summary.Version)
	}
	if summary.AssociatedProjects != nil {
		item.AssociatedProjects = types.Int64Value(int64(*summary.AssociatedProjects))
	}
	if summary.LastCheckin != nil {
		item.LastCheckin = types.StringValue(*summary.LastCheckin)
	}
	if summary.RunnerAsNodeEnabled != nil {
		item.RunnerAsNodeEnabled = types.BoolValue(*summary.RunnerAsNodeEnabled)
	}
	if summary.TagNames != nil {
		item.TagNames = RunnerTagsValue{
			StringValue: types.StringValue(normalizeRunnerTags(strings.Join(summary.TagNames, ","))),
		}
	}
	if summary.RunnerNodeFilter != nil {
		item.RunnerNodeFilter = types.StringValue(*summary.RunnerNodeFilter)
	}
	if summary.RemoteNodeDispatch != nil {
		item.RemoteNodeDispatch = types.BoolValue(*summary.RemoteNodeDispatch)
	}
	if summary.Hostname != nil {
		item.Hostname = types.StringValue(*summary.Hostname)
	}
	if summary.OsFamily != nil {
		item.OsFamily = types.StringValue(*summary.OsFamily)
	}
	if summary.ReplicaType != nil {
		item.ReplicaType = types.StringValue(string(*summary.ReplicaType))
	}
	if summary.InstallationType != nil {
		item.InstallationType = types.StringValue(string(*summary.InstallationType))
	}
	for _, p := range summary.Providers {
		provider := runnerProviderModel{}
		if p.Provider != nil {
			provider.Provider = types.StringValue(*p.Provider)
		}
		if p.ServiceName != nil {
			provider.ServiceName = types.StringValue(*p.ServiceName)
		}
		if p.PluginName != nil {
			provider.PluginName = types.StringValue(*p.PluginName)
		}
		item.Providers = append(item.Providers, provider)
	}
	if summary.RunnerReplicas != nil {
		item.RunnerReplicas = types.Int64Value(int64(*summary.RunnerReplicas))
	}
	if summary.HealthyRunnerReplicas != nil {
		item.HealthyRunnerReplicas = types.Int64Value(int64(*summary.HealthyRunnerReplicas))
	}
	if summary.RunningOperations != nil {
		item.RunningOperations = types.Int64Value(int64(*summary.RunningOperations))
	}
	if summary.Uptime != nil {
		item.Uptime = types.Int64Value(*summary.Uptime)
	}
	if summary.DateCreated != nil {
		item.DateCreated = types.StringValue(summary.DateCreated.Format("2006-01-02T15:04:05Z07:00"))
	}
	if summary.LastUpdated != nil {
		item.LastUpdated = types.StringValue(summary.LastUpdated.Format("2006-01-02T15:04:05Z07:00"))
	}

	return item
}
