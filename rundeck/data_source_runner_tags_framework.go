package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &runnerTagsDataSource{}
	_ datasource.DataSourceWithConfigure = &runnerTagsDataSource{}
)

func NewRunnerTagsDataSource() datasource.DataSource {
	return &runnerTagsDataSource{}
}

type runnerTagsDataSource struct {
	clients *RundeckClients
}

type runnerTagsDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ProjectName types.String `tfsdk:"project_name"`
	Tags        types.Map    `tfsdk:"tags"`
}

func (d *runnerTagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runner_tags"
}

func (d *runnerTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up runner tags in use and their usage counts for a project. The underlying Rundeck API requires a project scope; there is no system-wide mode for this endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this data source (same as project_name).",
				Computed:    true,
			},
			"project_name": schema.StringAttribute{
				Description: "Scope tag discovery to runners associated with this project.",
				Required:    true,
			},
			"tags": schema.MapAttribute{
				Description: "Map of tag name to the number of runners using that tag.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (d *runnerTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *runnerTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config runnerTagsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.clients.V2
	apiCtx := d.clients.ctx

	tagsReq := client.RunnerAPI.ListProjectAssociatedTags(apiCtx).Project(config.ProjectName.ValueString())

	tagCounts, _, err := tagsReq.Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error listing runner tags",
			fmt.Sprintf("Could not list runner tags: %s", err.Error()),
		)
		return
	}

	tagValues := make(map[string]attr.Value)
	if tagCounts != nil && tagCounts.Tags != nil {
		for name, count := range *tagCounts.Tags {
			tagValues[name] = types.Int64Value(int64(count))
		}
	}

	tagsMap, diags := types.MapValue(types.Int64Type, tagValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := runnerTagsDataSourceModel{
		ID:          config.ProjectName,
		ProjectName: config.ProjectName,
		Tags:        tagsMap,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
