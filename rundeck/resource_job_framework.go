package rundeck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Schedule               interface{}        `json:"schedule,omitempty"`
	Schedules              []interface{}      `json:"schedules,omitempty"`
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
				Computed: true,
				Default:  booldefault.StaticBool(false),
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
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"continue_next_node_on_error": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
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
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"preserve_options_order": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
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
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"runner_selector_filter": schema.StringAttribute{
				Optional: true,
			},
			"runner_selector_filter_mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"runner_selector_filter_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
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
	// JSON job import requires API v46+ (Rundeck 5.0.0+)
	apiVersion := r.client.APIVersion
	if apiVersion < "46" {
		apiVersion = "46"
	}
	apiURL := fmt.Sprintf("%s/api/%s/project/%s/jobs/import",
		r.client.BaseURL,
		apiVersion,
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
			Index     int    `json:"index"`
			ID        string `json:"id"`
			Name      string `json:"name"`
			Project   string `json:"project"`
			Href      string `json:"href"`
			Permalink string `json:"permalink"`
		} `json:"succeeded"`
		Failed []struct {
			Index   int    `json:"index"`
			Error   string `json:"error"`
			Message string `json:"message"`
		} `json:"failed"`
		Skipped []interface{} `json:"skipped"`
	}

	if err := json.Unmarshal(responseBody, &importResult); err != nil {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Could not parse import response: %s\nHTTP Status: %d\nResponse body: %s", err.Error(), httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	if httpResp.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("API returned status %d\nResponse body: %s", httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	if len(importResult.Failed) > 0 {
		errorMsg := importResult.Failed[0].Error
		if errorMsg == "" {
			errorMsg = importResult.Failed[0].Message
		}
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Job import failed: %s\nFull response: %s", errorMsg, string(responseBody)),
		)
		return
	}

	if len(importResult.Succeeded) == 0 {
		resp.Diagnostics.AddError(
			"Error creating job",
			fmt.Sprintf("Job import succeeded but no job ID was returned\nHTTP Status: %d\nResponse body: %s\nParsed result: succeeded=%d, failed=%d, skipped=%d",
				httpResp.StatusCode, string(responseBody), len(importResult.Succeeded), len(importResult.Failed), len(importResult.Skipped)),
		)
		return
	}

	// Set the job ID from import result
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

	// Read back from API to ensure state matches reality (eliminates drift)
	// This is critical for avoiding plan drift on subsequent refreshes
	apiJobData, err := GetJobJSON(r.client.V1, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading job after creation",
			fmt.Sprintf("Job was created but could not be read back: %s", err.Error()),
		)
		return
	}

	// Update plan with values from API
	if err = r.jobJSONAPIToState(apiJobData, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error reading job after creation",
			fmt.Sprintf("Could not convert job JSON to state: %s", err.Error()),
		)
		return
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

	// Use GetJobJSON to get job details
	jobData, err := GetJobJSON(client, state.ID.ValueString())
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading job",
			fmt.Sprintf("Could not read job %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Convert JSON API response to Terraform state
	if err := r.jobJSONAPIToState(jobData, &state); err != nil {
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
	// JSON job import requires API v46+ (Rundeck 5.0.0+)
	apiVersion := r.client.APIVersion
	if apiVersion < "46" {
		apiVersion = "46"
	}
	apiURL := fmt.Sprintf("%s/api/%s/project/%s/jobs/import",
		r.client.BaseURL,
		apiVersion,
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
			Index     int    `json:"index"`
			ID        string `json:"id"`
			Name      string `json:"name"`
			Project   string `json:"project"`
			Href      string `json:"href"`
			Permalink string `json:"permalink"`
		} `json:"succeeded"`
		Failed []struct {
			Index   int    `json:"index"`
			Error   string `json:"error"`
			Message string `json:"message"`
		} `json:"failed"`
		Skipped []interface{} `json:"skipped"`
	}

	if err := json.Unmarshal(responseBody, &importResult); err != nil {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Could not parse import response: %s\nHTTP Status: %d\nResponse body: %s", err.Error(), httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	if httpResp.StatusCode != 200 {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("API returned status %d\nResponse body: %s", httpResp.StatusCode, string(responseBody)),
		)
		return
	}

	if len(importResult.Failed) > 0 {
		errorMsg := importResult.Failed[0].Error
		if errorMsg == "" {
			errorMsg = importResult.Failed[0].Message
		}
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Job import failed: %s\nFull response: %s", errorMsg, string(responseBody)),
		)
		return
	}

	if len(importResult.Succeeded) == 0 {
		resp.Diagnostics.AddError(
			"Error updating job",
			fmt.Sprintf("Job import succeeded but no job ID was returned\nHTTP Status: %d\nResponse body: %s\nParsed result: succeeded=%d, failed=%d, skipped=%d",
				httpResp.StatusCode, string(responseBody), len(importResult.Succeeded), len(importResult.Failed), len(importResult.Skipped)),
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
	// Support both formats:
	// 1. Just job ID: "job-uuid"
	// 2. Project and job ID: "project-name/job-uuid"
	id := req.ID

	// Check if the ID contains a slash (project/job format)
	if strings.Contains(id, "/") {
		// Parse the project/job format (legacy import format)
		parts := strings.SplitN(id, "/", 2)
		if len(parts) != 2 {
			resp.Diagnostics.AddError(
				"Invalid Import ID",
				fmt.Sprintf("Expected import ID in format 'project-name/job-id' or just 'job-id', got: %s", id),
			)
			return
		}
		// Use just the job ID part
		id = parts[1]
	}

	// Set the ID - the Framework will automatically call Read() to populate the rest
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Override the ID if we parsed it from project/job-id format
	if id != req.ID {
		resp.State.SetAttribute(ctx, path.Root("id"), id)
	}
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

	// Convert execution_lifecycle_plugin
	if !plan.ExecutionLifecyclePlugin.IsNull() && !plan.ExecutionLifecyclePlugin.IsUnknown() {
		plugins, diags := convertExecutionLifecyclePluginsToJSON(ctx, plan.ExecutionLifecyclePlugin)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting execution lifecycle plugins: %v", diags.Errors())
		}
		if len(plugins) > 0 {
			job.Plugins = plugins
		}
	}

	// Convert project_schedule
	if !plan.ProjectSchedule.IsNull() && !plan.ProjectSchedule.IsUnknown() {
		schedules, diags := convertProjectSchedulesToJSON(ctx, plan.ProjectSchedule)
		if diags.HasError() {
			return nil, fmt.Errorf("error converting project schedules: %v", diags.Errors())
		}
		if len(schedules) > 0 {
			job.Schedules = schedules
			job.Schedule = nil // Set cron schedule to null when using project schedules
		}
	}

	// TODO: Convert remaining complex structures (orchestrator, log_limit, global_log_filter)
	// For now, these can be left nil and will be added incrementally

	return job, nil
}

// jobJSONToState converts Rundeck job JSON to Terraform state
func (r *jobResource) jobJSONToState(job *jobJSON, state *jobResourceModel) error {
	state.ID = types.StringValue(job.ID)
	state.Name = types.StringValue(job.Name)
	state.GroupName = types.StringValue(job.Group)
	// Project name is not returned by JobGet API, preserve from current state
	// state.ProjectName already has the value from state/plan
	state.Description = types.StringValue(job.Description)
	state.ExecutionEnabled = types.BoolValue(job.ExecutionEnabled)
	state.DefaultTab = types.StringValue(job.DefaultTab)
	state.LogLevel = types.StringValue(job.LogLevel)
	state.AllowConcurrentExecutions = types.BoolValue(job.MultipleExecutions)
	state.NodeFilterEditable = types.BoolValue(job.NodeFilterEditable)
	// NodesSelectedByDefault is not reliably returned by JSON API, preserve from state if not present
	// Only update if explicitly returned as true
	if job.NodesSelectedByDefault {
		state.NodesSelectedByDefault = types.BoolValue(true)
	}
	// Otherwise keep the value from state/plan
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

// jobJSONAPIToState converts API JSON response to Terraform state
func (r *jobResource) jobJSONAPIToState(job *JobJSON, state *jobResourceModel) error {
	// Map JSON fields directly to Terraform state
	state.ID = types.StringValue(job.ID)
	state.Name = types.StringValue(job.Name)

	// Only set group_name if API returns a non-empty value
	if job.Group != "" {
		state.GroupName = types.StringValue(job.Group)
	}

	// Project name is preserved from current state as API doesn't always return it
	if job.Project != "" {
		state.ProjectName = types.StringValue(job.Project)
	}

	// Only set description if non-empty, otherwise preserve from state
	if job.Description != "" {
		state.Description = types.StringValue(job.Description)
	}
	state.ExecutionEnabled = types.BoolValue(job.ExecutionEnabled)
	state.ScheduleEnabled = types.BoolValue(job.ScheduleEnabled)

	// Only set optional string fields if non-empty
	if job.DefaultTab != "" {
		state.DefaultTab = types.StringValue(job.DefaultTab)
	}
	if job.LogLevel != "" {
		state.LogLevel = types.StringValue(job.LogLevel)
	}

	state.AllowConcurrentExecutions = types.BoolValue(job.AllowConcurrentExec)
	state.NodeFilterEditable = types.BoolValue(job.NodeFilterEditable)

	// NodesSelectedByDefault - only update if explicitly true from API
	if job.NodesSelectedByDefault {
		state.NodesSelectedByDefault = types.BoolValue(true)
	}

	if job.Timeout != "" {
		state.Timeout = types.StringValue(job.Timeout)
	}

	// Handle retry
	if job.Retry != nil {
		if retry, ok := job.Retry["retry"]; ok {
			state.Retry = types.StringValue(retry)
		}
		if delay, ok := job.Retry["delay"]; ok {
			state.RetryDelay = types.StringValue(delay)
		}
	}

	// Handle node filters
	if job.NodeFilters != nil {
		if filter, ok := job.NodeFilters["filter"].(string); ok && filter != "" {
			state.NodeFilterQuery = types.StringValue(filter)
		}
		if excludeFilter, ok := job.NodeFilters["excludeFilter"].(string); ok && excludeFilter != "" {
			state.NodeFilterExcludeQuery = types.StringValue(excludeFilter)
		}
	}

	// Handle dispatch
	if job.Dispatch != nil {
		if threadCount, ok := job.Dispatch["threadcount"].(string); ok {
			if tc, err := strconv.ParseInt(threadCount, 10, 64); err == nil {
				state.MaxThreadCount = types.Int64Value(tc)
			}
		}
		if keepGoing, ok := job.Dispatch["keepgoing"].(bool); ok {
			state.ContinueNextNodeOnError = types.BoolValue(keepGoing)
		}
		if rankOrder, ok := job.Dispatch["rankOrder"].(string); ok {
			state.RankOrder = types.StringValue(rankOrder)
		}
		if rankAttr, ok := job.Dispatch["rankAttribute"].(string); ok {
			state.RankAttribute = types.StringValue(rankAttr)
		}
	}

	// Handle sequence
	if job.Sequence != nil {
		if keepGoing, ok := job.Sequence["keepgoing"].(bool); ok {
			state.ContinueNextNodeOnError = types.BoolValue(keepGoing)
		}
		if strategy, ok := job.Sequence["strategy"].(string); ok {
			state.CommandOrderingStrategy = types.StringValue(strategy)
		}
	}

	// Parse execution lifecycle plugins
	if job.Plugins != nil {
		if execLifecycle, ok := job.Plugins["ExecutionLifecycle"].(map[string]interface{}); ok && len(execLifecycle) > 0 {
			var pluginsList []attr.Value

			for pluginType, configVal := range execLifecycle {
				// Each plugin is a map entry where key=type, value=config
				pluginAttrs := map[string]attr.Value{
					"type": types.StringValue(pluginType),
				}

				// Convert config to map[string]string
				if configMap, ok := configVal.(map[string]interface{}); ok {
					configStrMap := make(map[string]attr.Value)
					for k, v := range configMap {
						if strVal, ok := v.(string); ok {
							configStrMap[k] = types.StringValue(strVal)
						}
					}
					if len(configStrMap) > 0 {
						pluginAttrs["config"] = types.MapValueMust(types.StringType, configStrMap)
					} else {
						// Empty config
						pluginAttrs["config"] = types.MapNull(types.StringType)
					}
				} else {
					// No config
					pluginAttrs["config"] = types.MapNull(types.StringType)
				}

				pluginObj := types.ObjectValueMust(
					map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					},
					pluginAttrs,
				)
				pluginsList = append(pluginsList, pluginObj)
			}

			if len(pluginsList) > 0 {
				state.ExecutionLifecyclePlugin = types.ListValueMust(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"type":   types.StringType,
							"config": types.MapType{ElemType: types.StringType},
						},
					},
					pluginsList,
				)
			}
		}
	}

	// Parse commands from sequence
	if job.Sequence != nil {
		if cmds, ok := job.Sequence["commands"].([]interface{}); ok && len(cmds) > 0 {
			// TODO: Implement full command parsing
			// This is complex due to the variety of command types (shell, script, job, plugin)
			// For now, import will require manual adjustment of command blocks
			// Or re-import from Terraform config
		}
	}

	// Parse options
	if len(job.Options) > 0 {
		// TODO: Implement full options parsing
		// For now, preserve from state to avoid drift
	}

	// Parse notifications
	if job.Notification != nil && len(job.Notification) > 0 {
		// TODO: Implement full notification parsing
		// For now, preserve from state to avoid drift
	}

	return nil
}
