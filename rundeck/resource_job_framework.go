package rundeck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

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

// jobResourceModel represents the Terraform resource model
type jobResourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	GroupName                   types.String `tfsdk:"group_name"`
	ProjectName                 types.String `tfsdk:"project_name"`
	Description                 types.String `tfsdk:"description"`
	ExecutionEnabled            types.Bool   `tfsdk:"execution_enabled"`
	DefaultTab                  types.String `tfsdk:"default_tab"`
	LogLevel                    types.String `tfsdk:"log_level"`
	AllowConcurrentExecutions   types.Bool   `tfsdk:"allow_concurrent_executions"`
	NodeFilterEditable          types.Bool   `tfsdk:"node_filter_editable"`
	Retry                       types.String `tfsdk:"retry"`
	RetryDelay                  types.String `tfsdk:"retry_delay"`
	MaxThreadCount              types.Int64  `tfsdk:"max_thread_count"`
	ContinueOnError             types.Bool   `tfsdk:"continue_on_error"`
	ContinueNextNodeOnError     types.Bool   `tfsdk:"continue_next_node_on_error"`
	RankOrder                   types.String `tfsdk:"rank_order"`
	RankAttribute               types.String `tfsdk:"rank_attribute"`
	SuccessOnEmptyNodeFilter    types.Bool   `tfsdk:"success_on_empty_node_filter"`
	PreserveOptionsOrder        types.Bool   `tfsdk:"preserve_options_order"`
	CommandOrderingStrategy     types.String `tfsdk:"command_ordering_strategy"`
	NodeFilterQuery             types.String `tfsdk:"node_filter_query"`
	NodeFilterExcludeQuery      types.String `tfsdk:"node_filter_exclude_query"`
	NodeFilterExcludePrecedence types.Bool   `tfsdk:"node_filter_exclude_precedence"`
	RunnerSelectorFilter        types.String `tfsdk:"runner_selector_filter"`
	RunnerSelectorFilterMode    types.String `tfsdk:"runner_selector_filter_mode"`
	RunnerSelectorFilterType    types.String `tfsdk:"runner_selector_filter_type"`
	Timeout                     types.String `tfsdk:"timeout"`
	Schedule                    types.String `tfsdk:"schedule"`
	ScheduleEnabled             types.Bool   `tfsdk:"schedule_enabled"`
	NodesSelectedByDefault      types.Bool   `tfsdk:"nodes_selected_by_default"`
	TimeZone                    types.String `tfsdk:"time_zone"`

	// Complex nested structures as lists
	Command                  types.List `tfsdk:"command"`
	Option                   types.List `tfsdk:"option"`
	Notification             types.List `tfsdk:"notification"`
	LogLimit                 types.List `tfsdk:"log_limit"`
	Orchestrator             types.List `tfsdk:"orchestrator"`
	GlobalLogFilter          types.List `tfsdk:"global_log_filter"`
	ProjectSchedule          types.List `tfsdk:"project_schedule"`
	ExecutionLifecyclePlugin types.List `tfsdk:"execution_lifecycle_plugin"`
}

// jobJSON represents the Rundeck Job JSON format (v44+)
type jobJSON struct {
	ID                     string             `json:"id,omitempty"`
	Name                   string             `json:"name"`
	Group                  string             `json:"group,omitempty"`
	Project                string             `json:"project"`
	Description            string             `json:"description"`
	ExecutionEnabled       bool               `json:"executionEnabled"`
	DefaultTab             string             `json:"defaultTab,omitempty"`
	LogLevel               string             `json:"loglevel,omitempty"`
	Loglimit               *string            `json:"loglimit,omitempty"`
	LogLimitAction         *string            `json:"loglimitAction,omitempty"`
	LogLimitStatus         *string            `json:"loglimitStatus,omitempty"`
	MultipleExecutions     bool               `json:"multipleExecutions,omitempty"`
	Dispatch               *jobDispatch       `json:"dispatch,omitempty"`
	Sequence               *jobSequence       `json:"sequence,omitempty"`
	Notification           interface{}        `json:"notification,omitempty"`
	Timeout                string             `json:"timeout,omitempty"`
	Retry                  *jobRetry          `json:"retry,omitempty"`
	NodeFilterEditable     bool               `json:"nodeFilterEditable"`
	NodeFilters            *jobNodeFilters    `json:"nodeFilters,omitempty"`
	Options                []interface{}      `json:"options,omitempty"`
	Plugins                interface{}        `json:"plugins,omitempty"`
	NodesSelectedByDefault bool               `json:"nodesSelectedByDefault"`
	Schedule               *jobSchedule       `json:"schedule,omitempty"`
	ScheduleEnabled        bool               `json:"scheduleEnabled"`
	TimeZone               string             `json:"timeZone,omitempty"`
	Orchestrator           interface{}        `json:"orchestrator,omitempty"`
	PluginConfig           interface{}        `json:"pluginConfig,omitempty"`
	RunnerSelector         *jobRunnerSelector `json:"runnerSelector,omitempty"`
}

type jobDispatch struct {
	ThreadCount              string `json:"threadcount,omitempty"`
	KeepGoing                bool   `json:"keepgoing,omitempty"`
	ExcludePrecedence        bool   `json:"excludePrecedence,omitempty"`
	RankOrder                string `json:"rankOrder,omitempty"`
	RankAttribute            string `json:"rankAttribute,omitempty"`
	SuccessOnEmptyNodeFilter bool   `json:"successOnEmptyNodeFilter,omitempty"`
}

type jobSequence struct {
	KeepGoing bool          `json:"keepgoing,omitempty"`
	Strategy  string        `json:"strategy,omitempty"`
	Commands  []interface{} `json:"commands"`
}

type jobRetry struct {
	Delay string `json:"delay,omitempty"`
	Retry string `json:"retry,omitempty"`
}

type jobNodeFilters struct {
	Filter        string `json:"filter,omitempty"`
	ExcludeFilter string `json:"excludeFilter,omitempty"`
}

type jobSchedule struct {
	Time       *jobScheduleTime       `json:"time,omitempty"`
	Month      *jobScheduleMonth      `json:"month,omitempty"`
	Year       *jobScheduleYear       `json:"year,omitempty"`
	Weekday    *jobScheduleWeekday    `json:"weekday,omitempty"`
	DayOfMonth *jobScheduleDayOfMonth `json:"dayofmonth,omitempty"`
	Crontab    string                 `json:"crontab,omitempty"`
}

type jobScheduleTime struct {
	Hour    string `json:"hour,omitempty"`
	Minute  string `json:"minute,omitempty"`
	Seconds string `json:"seconds,omitempty"`
}

type jobScheduleMonth struct {
	Month string `json:"month,omitempty"`
	Day   string `json:"day,omitempty"`
}

type jobScheduleYear struct {
	Year string `json:"year,omitempty"`
}

type jobScheduleWeekday struct {
	Day string `json:"day,omitempty"`
}

type jobScheduleDayOfMonth struct {
	Day string `json:"day,omitempty"`
}

type jobRunnerSelector struct {
	Filter     string `json:"filter,omitempty"`
	FilterMode string `json:"filterMode,omitempty"`
	FilterType string `json:"filterType,omitempty"`
}

func NewJobResource() resource.Resource {
	return &jobResource{}
}

func (r *jobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_job"
}

func (r *jobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Rundeck Job definition using JSON format (API v44+)",
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
		},
		Blocks: map[string]schema.Block{
			"command":                    jobCommandNestedBlock(),
			"option":                     jobOptionNestedBlock(),
			"notification":               jobNotificationNestedBlock(),
			"log_limit":                  jobLogLimitNestedBlock(),
			"orchestrator":               jobOrchestratorNestedBlock(),
			"global_log_filter":          jobGlobalLogFilterNestedBlock(),
			"project_schedule":           jobProjectScheduleNestedBlock(),
			"execution_lifecycle_plugin": jobExecutionLifecyclePluginNestedBlock(),
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

func (r *jobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan jobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform model to Rundeck JSON format
	jobData, err := r.planToJobJSON(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not convert plan to job JSON: %s", err.Error()),
		)
		return
	}

	// Marshal to JSON
	jobJSON, err := json.Marshal([]interface{}{jobData})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not marshal job to JSON: %s", err.Error()),
		)
		return
	}

	// Import the job using custom HTTP request to ensure JSON response
	apiURL := fmt.Sprintf("%s/api/%s/project/%s/jobs/import",
		r.client.BaseURL,
		r.client.APIVersion,
		plan.ProjectName.ValueString())

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jobJSON))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Rundeck-Auth-Token", r.client.Token)

	// Add query parameters
	q := httpReq.URL.Query()
	q.Add("fileformat", "json")
	q.Add("dupeOption", "create")
	q.Add("uuidOption", "preserve")
	httpReq.URL.RawQuery = q.Encode()

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not import job: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not read import response: %s", err.Error()),
		)
		return
	}

	// Parse JSON import result
	var importResult struct {
		Succeeded []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Project string `json:"project"`
		} `json:"succeeded"`
		Failed []struct {
			Error string `json:"error"`
		} `json:"failed"`
	}

	if err := json.Unmarshal(responseBody, &importResult); err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not parse import response: %s\nResponse body: %s", err.Error(), string(responseBody)),
		)
		return
	}

	if len(importResult.Failed) > 0 {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Job import failed: %s", importResult.Failed[0].Error),
		)
		return
	}

	if len(importResult.Succeeded) == 0 {
		resp.Diagnostics.AddError(
			"Error creating job",
			"Job import succeeded but no job ID was returned",
		)
		return
	}

	// Set the job ID
	plan.ID = types.StringValue(importResult.Succeeded[0].ID)

	// Set computed fields that may not have been set in the plan
	if plan.PreserveOptionsOrder.IsUnknown() {
		plan.PreserveOptionsOrder = types.BoolValue(false)
	}
	if plan.RunnerSelectorFilterMode.IsUnknown() {
		plan.RunnerSelectorFilterMode = types.StringNull()
	}
	if plan.RunnerSelectorFilterType.IsUnknown() {
		plan.RunnerSelectorFilterType = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state jobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V1
	apiCtx := context.Background()

	// Get job in JSON format
	jobResp, err := client.JobGet(apiCtx, state.ID.ValueString(), "json")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading job",
			fmt.Sprintf("Could not read job %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	if jobResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if jobResp.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Error reading job",
			fmt.Sprintf("API returned status %d", jobResp.StatusCode),
		)
		return
	}

	// Read and parse JSON response
	body, err := io.ReadAll(jobResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading job",
			fmt.Sprintf("Could not read response body: %s", err.Error()),
		)
		return
	}

	// JobGet returns an array of jobs
	var jobs []jobJSON
	if err := json.Unmarshal(body, &jobs); err != nil {
		resp.Diagnostics.AddError(
			"Error reading job",
			fmt.Sprintf("Could not parse job JSON: %s\nResponse: %s", err.Error(), string(body)),
		)
		return
	}

	if len(jobs) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	jobData := jobs[0]
	
	// Debug: Check what we're getting from API
	debugJSON, _ := json.MarshalIndent(jobData, "", "  ")
	resp.Diagnostics.AddWarning("DEBUG: Job from API", string(debugJSON))

	// Convert JSON to Terraform state
	if err := r.jobJSONToState(&jobData, &state); err != nil {
		resp.Diagnostics.AddError(
			"Error reading job",
			fmt.Sprintf("Could not convert job JSON to state: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *jobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan jobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform model to Rundeck JSON format
	jobData, err := r.planToJobJSON(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not convert plan to job JSON: %s", err.Error()),
		)
		return
	}

	// Preserve the job ID for update
	jobData.ID = plan.ID.ValueString()

	// Marshal to JSON
	jobJSON, err := json.Marshal([]interface{}{jobData})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not marshal job to JSON: %s", err.Error()),
		)
		return
	}

	// Import/update the job using custom HTTP request to ensure JSON response
	apiURL := fmt.Sprintf("%s/api/%s/project/%s/jobs/import",
		r.client.BaseURL,
		r.client.APIVersion,
		plan.ProjectName.ValueString())

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jobJSON))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not create request: %s", err.Error()),
		)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Rundeck-Auth-Token", r.client.Token)

	// Add query parameters
	q := httpReq.URL.Query()
	q.Add("fileformat", "json")
	q.Add("dupeOption", "update") // Use "update" instead of "create"
	q.Add("uuidOption", "preserve")
	httpReq.URL.RawQuery = q.Encode()

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not import job: %s", err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not read import response: %s", err.Error()),
		)
		return
	}

	// Parse JSON import result
	var importResult struct {
		Succeeded []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Project string `json:"project"`
		} `json:"succeeded"`
		Failed []struct {
			Error string `json:"error"`
		} `json:"failed"`
	}

	if err := json.Unmarshal(responseBody, &importResult); err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not parse import response: %s\nResponse body: %s", err.Error(), string(responseBody)),
		)
		return
	}

	if len(importResult.Failed) > 0 {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Job import failed: %s", importResult.Failed[0].Error),
		)
		return
	}

	// Set computed fields that may not have been set in the plan
	if plan.PreserveOptionsOrder.IsUnknown() {
		plan.PreserveOptionsOrder = types.BoolValue(false)
	}
	if plan.RunnerSelectorFilterMode.IsUnknown() {
		plan.RunnerSelectorFilterMode = types.StringNull()
	}
	if plan.RunnerSelectorFilterType.IsUnknown() {
		plan.RunnerSelectorFilterType = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *jobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state jobResourceModel

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

// planToJobJSON converts Terraform plan to Rundeck job JSON format
func (r *jobResource) planToJobJSON(ctx context.Context, plan *jobResourceModel) (*jobJSON, error) {
	job := &jobJSON{
		Name:                   plan.Name.ValueString(),
		Group:                  plan.GroupName.ValueString(),
		Project:                plan.ProjectName.ValueString(),
		Description:            plan.Description.ValueString(),
		ExecutionEnabled:       plan.ExecutionEnabled.ValueBool(),
		DefaultTab:             plan.DefaultTab.ValueString(),
		LogLevel:               plan.LogLevel.ValueString(),
		NodeFilterEditable:     plan.NodeFilterEditable.ValueBool(),
		NodesSelectedByDefault: plan.NodesSelectedByDefault.ValueBool(),
		ScheduleEnabled:        plan.ScheduleEnabled.ValueBool(),
	}

	if !plan.AllowConcurrentExecutions.IsNull() {
		job.MultipleExecutions = plan.AllowConcurrentExecutions.ValueBool()
	}

	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		job.Timeout = plan.Timeout.ValueString()
	}

	if !plan.TimeZone.IsNull() && !plan.TimeZone.IsUnknown() {
		job.TimeZone = plan.TimeZone.ValueString()
	}

	// Handle retry configuration
	if (!plan.Retry.IsNull() && !plan.Retry.IsUnknown()) || (!plan.RetryDelay.IsNull() && !plan.RetryDelay.IsUnknown()) {
		job.Retry = &jobRetry{}
		if !plan.Retry.IsNull() {
			job.Retry.Retry = plan.Retry.ValueString()
		}
		if !plan.RetryDelay.IsNull() {
			job.Retry.Delay = plan.RetryDelay.ValueString()
		}
	}

	// Handle dispatch/node filter configuration
	if !plan.NodeFilterQuery.IsNull() || !plan.NodeFilterExcludeQuery.IsNull() {
		job.NodeFilters = &jobNodeFilters{}
		if !plan.NodeFilterQuery.IsNull() {
			job.NodeFilters.Filter = plan.NodeFilterQuery.ValueString()
		}
		if !plan.NodeFilterExcludeQuery.IsNull() {
			job.NodeFilters.ExcludeFilter = plan.NodeFilterExcludeQuery.ValueString()
		}
	}

	// Handle dispatch configuration
	if !plan.MaxThreadCount.IsNull() || !plan.ContinueOnError.IsNull() || !plan.RankOrder.IsNull() {
		job.Dispatch = &jobDispatch{}
		if !plan.MaxThreadCount.IsNull() {
			job.Dispatch.ThreadCount = fmt.Sprintf("%d", plan.MaxThreadCount.ValueInt64())
		}
		if !plan.ContinueOnError.IsNull() {
			job.Dispatch.KeepGoing = plan.ContinueOnError.ValueBool()
		}
		if !plan.NodeFilterExcludePrecedence.IsNull() {
			job.Dispatch.ExcludePrecedence = plan.NodeFilterExcludePrecedence.ValueBool()
		}
		if !plan.RankOrder.IsNull() {
			job.Dispatch.RankOrder = plan.RankOrder.ValueString()
		}
		if !plan.RankAttribute.IsNull() {
			job.Dispatch.RankAttribute = plan.RankAttribute.ValueString()
		}
		if !plan.SuccessOnEmptyNodeFilter.IsNull() {
			job.Dispatch.SuccessOnEmptyNodeFilter = plan.SuccessOnEmptyNodeFilter.ValueBool()
		}
	}

	// Handle runner selector
	if !plan.RunnerSelectorFilter.IsNull() || !plan.RunnerSelectorFilterMode.IsNull() || !plan.RunnerSelectorFilterType.IsNull() {
		job.RunnerSelector = &jobRunnerSelector{}
		if !plan.RunnerSelectorFilter.IsNull() {
			job.RunnerSelector.Filter = plan.RunnerSelectorFilter.ValueString()
		}
		if !plan.RunnerSelectorFilterMode.IsNull() {
			job.RunnerSelector.FilterMode = plan.RunnerSelectorFilterMode.ValueString()
		}
		if !plan.RunnerSelectorFilterType.IsNull() {
			job.RunnerSelector.FilterType = plan.RunnerSelectorFilterType.ValueString()
		}
	}

	// Convert commands (required)
	commands, diags := convertCommandsToJSON(ctx, plan.Command)
	if diags.HasError() {
		return nil, fmt.Errorf("error converting commands: %v", diags.Errors())
	}
	if len(commands) > 0 {
		job.Sequence = &jobSequence{
			Commands: commands,
			Strategy: plan.CommandOrderingStrategy.ValueString(),
		}
		if !plan.ContinueNextNodeOnError.IsNull() {
			job.Sequence.KeepGoing = plan.ContinueNextNodeOnError.ValueBool()
		}
	}

	// Convert options (Rundeck expects an array, not a map)
	if !plan.Option.IsNull() && !plan.Option.IsUnknown() {
		options, diags := convertOptionsToJSON(ctx, plan.Option)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting options: %v", diags.Errors())
		}
		if len(options) > 0 {
			job.Options = options
		}
	}

	// Convert notifications
	if !plan.Notification.IsNull() && !plan.Notification.IsUnknown() {
		notifications, diags := convertNotificationsToJSON(ctx, plan.Notification)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting notifications: %v", diags.Errors())
		}
		if len(notifications) > 0 {
			job.Notification = notifications
		}
	}

	// TODO: Convert remaining complex structures (orchestrator, log_limit, global_log_filter, etc.)
	// For now, these can be left nil and will be added incrementally

	return job, nil
}

// jobJSONToState converts Rundeck job JSON to Terraform state
func (r *jobResource) jobJSONToState(job *jobJSON, state *jobResourceModel) error {
	state.ID = types.StringValue(job.ID)
	state.Name = types.StringValue(job.Name)
	state.GroupName = types.StringValue(job.Group)
	state.ProjectName = types.StringValue(job.Project)
	state.Description = types.StringValue(job.Description)
	state.ExecutionEnabled = types.BoolValue(job.ExecutionEnabled)
	state.DefaultTab = types.StringValue(job.DefaultTab)
	state.LogLevel = types.StringValue(job.LogLevel)
	state.AllowConcurrentExecutions = types.BoolValue(job.MultipleExecutions)
	state.NodeFilterEditable = types.BoolValue(job.NodeFilterEditable)
	state.NodesSelectedByDefault = types.BoolValue(job.NodesSelectedByDefault)
	state.ScheduleEnabled = types.BoolValue(job.ScheduleEnabled)

	if job.Timeout != "" {
		state.Timeout = types.StringValue(job.Timeout)
	}
	if job.TimeZone != "" {
		state.TimeZone = types.StringValue(job.TimeZone)
	}

	// Handle retry configuration
	if job.Retry != nil {
		if job.Retry.Retry != "" {
			state.Retry = types.StringValue(job.Retry.Retry)
		}
		if job.Retry.Delay != "" {
			state.RetryDelay = types.StringValue(job.Retry.Delay)
		}
	}

	// Handle node filters
	if job.NodeFilters != nil {
		if job.NodeFilters.Filter != "" {
			state.NodeFilterQuery = types.StringValue(job.NodeFilters.Filter)
		}
		if job.NodeFilters.ExcludeFilter != "" {
			state.NodeFilterExcludeQuery = types.StringValue(job.NodeFilters.ExcludeFilter)
		}
	}

	// Handle dispatch configuration
	if job.Dispatch != nil {
		// Parse threadcount string to int64
		if job.Dispatch.ThreadCount != "" {
			threadCount, _ := strconv.ParseInt(job.Dispatch.ThreadCount, 10, 64)
			state.MaxThreadCount = types.Int64Value(threadCount)
		}
		state.ContinueOnError = types.BoolValue(job.Dispatch.KeepGoing)
		state.NodeFilterExcludePrecedence = types.BoolValue(job.Dispatch.ExcludePrecedence)
		if job.Dispatch.RankOrder != "" {
			state.RankOrder = types.StringValue(job.Dispatch.RankOrder)
		}
		if job.Dispatch.RankAttribute != "" {
			state.RankAttribute = types.StringValue(job.Dispatch.RankAttribute)
		}
		state.SuccessOnEmptyNodeFilter = types.BoolValue(job.Dispatch.SuccessOnEmptyNodeFilter)
	}

	// Handle runner selector
	if job.RunnerSelector != nil {
		if job.RunnerSelector.Filter != "" {
			state.RunnerSelectorFilter = types.StringValue(job.RunnerSelector.Filter)
		}
		if job.RunnerSelector.FilterMode != "" {
			state.RunnerSelectorFilterMode = types.StringValue(job.RunnerSelector.FilterMode)
		}
		if job.RunnerSelector.FilterType != "" {
			state.RunnerSelectorFilterType = types.StringValue(job.RunnerSelector.FilterType)
		}
	}

	// TODO: Convert JSON structures back to Framework nested lists
	// For now, we preserve input from config and don't update these from reads
	// This allows basic job CRUD to work while we implement full bidirectional conversion

	// Set sequence metadata
	if job.Sequence != nil {
		if job.Sequence.Strategy != "" {
			state.CommandOrderingStrategy = types.StringValue(job.Sequence.Strategy)
		}
		state.ContinueNextNodeOnError = types.BoolValue(job.Sequence.KeepGoing)
	}

	// For complex nested structures, preserve null if not set
	// The create/update will send them, but read won't overwrite them
	// This is a known limitation until we implement JSON->List converters

	return nil
}
