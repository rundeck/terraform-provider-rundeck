package rundeck

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type jobResource struct {
	client *RundeckClients
}

func NewJobResource() resource.Resource {
	return &jobResource{}
}

func (r *jobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *jobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Rundeck Job definition",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Job name",
			},
			"group_name": schema.StringAttribute{
				Optional:    true,
				Description: "Job group name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_name": schema.StringAttribute{
				Required:    true,
				Description: "Project name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Required:    true,
				Description: "Job description",
			},
			"execution_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"default_tab": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("nodes"),
			},
			"log_level": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("INFO"),
			},
			"allow_concurrent_executions": schema.BoolAttribute{
				Optional: true,
			},
			"node_filter_editable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"retry": schema.StringAttribute{
				Optional: true,
			},
			"retry_delay": schema.StringAttribute{
				Optional: true,
			},
			"max_thread_count": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1),
			},
			"continue_on_error": schema.BoolAttribute{
				Optional: true,
			},
			"continue_next_node_on_error": schema.BoolAttribute{
				Optional: true,
			},
			"rank_order": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("ascending"),
			},
			"rank_attribute": schema.StringAttribute{
				Optional: true,
			},
			"success_on_empty_node_filter": schema.BoolAttribute{
				Optional: true,
			},
			"preserve_options_order": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"command_ordering_strategy": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("node-first"),
			},
			"node_filter_query": schema.StringAttribute{
				Optional: true,
			},
			"node_filter_exclude_query": schema.StringAttribute{
				Optional: true,
			},
			"node_filter_exclude_precedence": schema.BoolAttribute{
				Optional: true,
			},
			"runner_selector_filter": schema.StringAttribute{
				Optional: true,
			},
			"runner_selector_filter_mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"runner_selector_filter_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"timeout": schema.StringAttribute{
				Optional: true,
			},
			"schedule": schema.StringAttribute{
				Optional: true,
			},
			"schedule_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"nodes_selected_by_default": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"time_zone": schema.StringAttribute{
				Optional: true,
			},
			// Note: Complex nested structures like log_limit, orchestrator, notification, option,
			// global_log_filter, command, project_schedule, and execution_lifecycle_plugin
			// will be implemented as JSON strings for simplicity in initial implementation
			"log_limit": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing log limit configuration",
			},
			"orchestrator": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing orchestrator configuration",
			},
			"notification": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing notification configuration",
			},
			"option": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing job options configuration",
			},
			"global_log_filter": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing global log filter configuration",
			},
			"command": schema.StringAttribute{
				Required:    true,
				Description: "JSON string containing command/workflow configuration",
			},
			"project_schedule": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing project schedule configuration",
			},
			"execution_lifecycle_plugin": schema.StringAttribute{
				Optional:    true,
				Description: "JSON string containing execution lifecycle plugin configuration",
			},
		},
	}
}

func (r *jobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*RundeckClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *RundeckClients, got: %T", req.ProviderData),
		)
		return
	}

	r.client = clients
}

// For MVP, we'll delegate to the existing SDK implementation by converting between
// Plugin Framework and SDK types. This allows us to leverage the existing complex
// XML marshaling/unmarshaling logic without rewriting 1800+ lines.

func (r *jobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: Implement Create using existing jobFromResourceData and importJob functions
	resp.Diagnostics.AddError(
		"Not Implemented",
		"Job resource Create operation is not yet implemented in Plugin Framework. "+
			"This resource requires complex XML marshaling logic that will be migrated incrementally.",
	)
}

func (r *jobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: Implement Read using existing GetJob and jobToResourceData functions
	resp.Diagnostics.AddError(
		"Not Implemented",
		"Job resource Read operation is not yet implemented in Plugin Framework",
	)
}

func (r *jobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: Implement Update using existing jobFromResourceData and importJob functions
	resp.Diagnostics.AddError(
		"Not Implemented",
		"Job resource Update operation is not yet implemented in Plugin Framework",
	)
}

func (r *jobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state struct {
		ID types.String `tfsdk:"id"`
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V1
	apiCtx := context.Background()

	_, err := client.JobDelete(apiCtx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting job",
			fmt.Sprintf("Could not delete job %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}
}

func (r *jobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

