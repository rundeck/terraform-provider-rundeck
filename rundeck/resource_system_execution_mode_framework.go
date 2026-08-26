package rundeck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &systemExecutionModeResource{}
	_ resource.ResourceWithConfigure   = &systemExecutionModeResource{}
	_ resource.ResourceWithImportState = &systemExecutionModeResource{}
)

const (
	executionModeActive  = "active"
	executionModePassive = "passive"

	// The mode is a property of the server, not of a named object, so there is
	// only ever one of these per Rundeck instance and the id is a constant.
	systemExecutionModeID = "system"

	// minExecutionModeAPIVersion is the lowest API version on which
	// GET .../system/executions/status reliably reports passive mode instead
	// of returning HTTP 503 (rundeck/rundeck#5846).
	minExecutionModeAPIVersion = 36

	// executionModeRequestTimeout bounds how long a single request to the
	// execution mode endpoints may take, so a hung server or network
	// partition doesn't block terraform apply/plan/destroy indefinitely.
	executionModeRequestTimeout = 30 * time.Second
)

// NewSystemExecutionModeResource is a helper function to simplify the provider implementation.
func NewSystemExecutionModeResource() resource.Resource {
	return &systemExecutionModeResource{}
}

type systemExecutionModeResource struct {
	client *RundeckClients
}

type systemExecutionModeResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Mode types.String `tfsdk:"mode"`
}

func (r *systemExecutionModeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_execution_mode"
}

func (r *systemExecutionModeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages whether a Rundeck server executes jobs. This is server-wide runtime state, " +
			"not per-project configuration: in passive mode no job runs, whether scheduled or started by hand. " +
			"Rundeck also reads a `rundeck.executionMode` property at startup; that property decides the mode " +
			"a restarted server comes back in, and this resource decides the mode it runs in from the next apply " +
			"onwards. Keep the two consistent unless a restart is meant to change the mode.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always \"system\": a Rundeck server has a single execution mode.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "Either \"active\" (jobs run) or \"passive\" (no job runs).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(executionModeActive, executionModePassive),
				},
			},
		},
	}
}

func (r *systemExecutionModeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	// GET .../system/executions/status returns HTTP 503 instead of the
	// expected JSON body when the server is passive on API versions below 36
	// (rundeck/rundeck#5846), which would otherwise surface as a confusing raw
	// API error on every plan/refresh/import instead of a clear diagnostic.
	if version, err := strconv.Atoi(clients.APIVersion); err != nil || version < minExecutionModeAPIVersion {
		resp.Diagnostics.AddError(
			"Insufficient API Version",
			fmt.Sprintf("rundeck_system_execution_mode requires API version %d or higher (currently configured: %s), "+
				"since the status endpoint does not reliably report passive mode before then. "+
				"Please update your provider configuration with api_version = \"%d\" or higher.",
				minExecutionModeAPIVersion, clients.APIVersion, minExecutionModeAPIVersion),
		)
		return
	}

	r.client = clients
}

// apiRequest issues a request against the system execution mode endpoints and
// returns the executionMode the server reports back.
func (r *systemExecutionModeResource) apiRequest(ctx context.Context, method, path string) (string, error) {
	url := fmt.Sprintf("%s/api/%s/system/executions/%s", r.client.BaseURL, r.client.APIVersion, path)

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", fmt.Errorf("could not create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Rundeck-Auth-Token", r.client.Token)

	httpResp, err := (&http.Client{Timeout: executionModeRequestTimeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("could not execute request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read response: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result struct {
		ExecutionMode string `json:"executionMode"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("could not parse response: %w", err)
	}
	if result.ExecutionMode == "" {
		return "", fmt.Errorf("response carried no executionMode: %s", string(body))
	}

	return result.ExecutionMode, nil
}

// setMode switches the server to the requested mode and returns the mode it
// reports afterwards.
func (r *systemExecutionModeResource) setMode(ctx context.Context, mode string) (string, error) {
	path := "disable"
	if mode == executionModeActive {
		path = "enable"
	}
	return r.apiRequest(ctx, http.MethodPost, path)
}

// Create does not create anything: the execution mode always exists. It brings
// the server to the requested mode and starts tracking it.
func (r *systemExecutionModeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemExecutionModeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wanted := plan.Mode.ValueString()
	got, err := r.setMode(ctx, wanted)
	if err != nil {
		resp.Diagnostics.AddError("Error setting execution mode", err.Error())
		return
	}
	plan.ID = types.StringValue(systemExecutionModeID)
	if got != wanted {
		// The POST already changed the live server even though the
		// confirmation read didn't echo the requested mode back (e.g. a
		// lagging read on a clustered/HA setup). Record what the server
		// actually reports rather than erroring out with nothing saved to
		// state: an untracked resource would otherwise re-issue the same
		// POST against an already-changed server on the next apply.
		plan.Mode = types.StringValue(got)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error setting execution mode",
			fmt.Sprintf("Asked for %q but the server reports %q. Tracking the reported mode; re-apply once the server has settled.", wanted, got),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemExecutionModeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemExecutionModeResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mode, err := r.apiRequest(ctx, http.MethodGet, "status")
	if err != nil {
		resp.Diagnostics.AddError("Error reading execution mode", err.Error())
		return
	}

	state.ID = types.StringValue(systemExecutionModeID)
	state.Mode = types.StringValue(mode)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemExecutionModeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemExecutionModeResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wanted := plan.Mode.ValueString()
	got, err := r.setMode(ctx, wanted)
	if err != nil {
		resp.Diagnostics.AddError("Error setting execution mode", err.Error())
		return
	}
	plan.ID = types.StringValue(systemExecutionModeID)
	if got != wanted {
		// See the identical comment in Create: persist what the server
		// actually reports instead of leaving state stuck on the old mode,
		// which would otherwise silently mask the fact that the POST already
		// took effect.
		plan.Mode = types.StringValue(got)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError(
			"Error setting execution mode",
			fmt.Sprintf("Asked for %q but the server reports %q. Tracking the reported mode; re-apply once the server has settled.", wanted, got),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete stops tracking the mode and deliberately leaves the server as it is.
// There is nothing to remove: a Rundeck server always has an execution mode,
// and silently flipping it on `terraform destroy` would either halt a live
// server or start jobs nobody asked to start.
func (r *systemExecutionModeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Execution mode left unchanged",
		"Removing rundeck_system_execution_mode from state does not change the server: it keeps whatever mode it is in. "+
			"Set the mode explicitly beforehand if the server should be left in a particular state.",
	)
}

func (r *systemExecutionModeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	mode, err := r.apiRequest(ctx, http.MethodGet, "status")
	if err != nil {
		resp.Diagnostics.AddError("Error importing execution mode", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, systemExecutionModeResourceModel{
		ID:   types.StringValue(systemExecutionModeID),
		Mode: types.StringValue(mode),
	})...)
}
