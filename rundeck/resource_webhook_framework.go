package rundeck

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &webhookResource{}
	_ resource.ResourceWithConfigure   = &webhookResource{}
	_ resource.ResourceWithImportState = &webhookResource{}
)

// normalizeRoles sorts a comma-separated roles string alphabetically
// to prevent plan drift caused by API reordering.
func normalizeRoles(roles string) string {
	parts := strings.Split(roles, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	sort.Strings(parts)
	return strings.Join(parts, ",")
}

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	clients *RundeckClients
}

type webhookResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Project     types.String `tfsdk:"project"`
	Name        types.String `tfsdk:"name"`
	User        types.String `tfsdk:"user"`
	Roles       types.String `tfsdk:"roles"`
	EventPlugin types.String `tfsdk:"event_plugin"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Config      types.Object `tfsdk:"config"`
	AuthToken   types.String `tfsdk:"auth_token"`
	Rules       types.List   `tfsdk:"rules"`
}

// webhookConfigModel represents the webhook configuration for different plugin types.
// Rules are managed as a top-level resource block, not inside config.
type webhookConfigModel struct {
	LogLevel types.String `tfsdk:"log_level"`

	JobID      types.String `tfsdk:"job_id"`
	ArgString  types.String `tfsdk:"arg_string"`
	NodeFilter types.String `tfsdk:"node_filter"`
	AsUser     types.String `tfsdk:"as_user"`

	KeyStoragePath   types.String `tfsdk:"key_storage_path"`
	BatchKey         types.String `tfsdk:"batch_key"`
	EventIDKey       types.String `tfsdk:"event_id_key"`
	ReturnProcessing types.Bool   `tfsdk:"return_processing_info"`

	Secret        types.String `tfsdk:"secret"`
	AutoSubscribe types.Bool   `tfsdk:"auto_subscribe"`
}

// webhookAdvancedRule represents an action rule for advanced-run-job plugin
type webhookAdvancedRule struct {
	Name       types.String `tfsdk:"name"`
	JobID      types.String `tfsdk:"job_id"`
	JobName    types.String `tfsdk:"job_name"`
	NodeFilter types.String `tfsdk:"node_filter"`
	User       types.String `tfsdk:"user"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	Policy     types.String `tfsdk:"policy"`
	JobOptions types.List   `tfsdk:"job_options"`
	Conditions types.List   `tfsdk:"conditions"`
}

// webhookJobOption represents a job option key-value pair
type webhookJobOption struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// webhookCondition represents a routing condition for advanced-run-job
type webhookCondition struct {
	Path      types.String `tfsdk:"path"`
	Condition types.String `tfsdk:"condition"`
	Value     types.String `tfsdk:"value"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck webhook for triggering automation from external systems.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project": schema.StringAttribute{
				Description: "The project name that owns the webhook.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the webhook. Cannot be changed after creation - requires replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user": schema.StringAttribute{
				Description: "The username the webhook executes as. Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.StringAttribute{
				Description: "Comma-separated list of roles for authorization. Cannot be changed after creation - requires replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"event_plugin": schema.StringAttribute{
				Description: "The plugin type (e.g., 'webhook-run-job', 'log-webhook-event'). Cannot be changed after creation.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the webhook is enabled. Cannot be changed after creation - requires replacement.",
				Required:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"auth_token": schema.StringAttribute{
				Description: "The authentication token for the webhook. Only available after creation.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"config": schema.SingleNestedBlock{
				Description: "Plugin-specific configuration. Structure varies by event_plugin type.",
				Attributes: map[string]schema.Attribute{
					"log_level": schema.StringAttribute{
						Description: "Log level for log-webhook-event plugin (INFO, DEBUG, WARN, ERROR)",
						Optional:    true,
					},
					"job_id": schema.StringAttribute{
						Description: "Job ID for webhook-run-job plugin",
						Optional:    true,
					},
					"arg_string": schema.StringAttribute{
						Description: "Job option arguments for webhook-run-job plugin, in the form '-opt1 value -opt2 \"other value\"'",
						Optional:    true,
					},
					"node_filter": schema.StringAttribute{
						Description: "Node filter override for webhook-run-job plugin",
						Optional:    true,
					},
					"as_user": schema.StringAttribute{
						Description: "Execute job as specified user for webhook-run-job plugin",
						Optional:    true,
					},
					"key_storage_path": schema.StringAttribute{
						Description: "Key storage path for advanced-run-job and Enterprise plugins",
						Optional:    true,
					},
					"batch_key": schema.StringAttribute{
						Description: "JSONPath to key containing items to treat as individual events (advanced-run-job, datadog-run-job, pagerduty-run-job)",
						Optional:    true,
					},
					"event_id_key": schema.StringAttribute{
						Description: "JSONPath to key containing value to use as the event ID; works with batches (advanced-run-job, datadog-run-job, pagerduty-run-job, pagerduty-V3-run-job)",
						Optional:    true,
					},
					"return_processing_info": schema.BoolAttribute{
						Description: "Return processing information to the caller of the webhook. Default return is {\"msg\":\"ok\"} (advanced-run-job and Enterprise plugins)",
						Optional:    true,
					},
					"secret": schema.StringAttribute{
						Description: "GitHub webhook secret for authentication (github-webhook plugin only)",
						Optional:    true,
						Sensitive:   true,
					},
					"auto_subscribe": schema.BoolAttribute{
						Description: "Automatically confirm SNS subscription (aws-sns-webhook plugin only)",
						Optional:    true,
					},
				},
			},
			"rules": schema.ListNestedBlock{
				Description: "Action rules for advanced-run-job and Enterprise plugins. Each rule maps incoming webhook events to a job execution.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the action rule",
							Required:    true,
						},
						"job_id": schema.StringAttribute{
							Description: "UUID of the job to run",
							Required:    true,
						},
						"job_name": schema.StringAttribute{
							Description: "Name of the job (computed from job_id)",
							Optional:    true,
							Computed:    true,
						},
						"node_filter": schema.StringAttribute{
							Description: "Node filter override for job execution",
							Optional:    true,
						},
						"user": schema.StringAttribute{
							Description: "User override for job execution",
							Optional:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether this rule is enabled",
							Optional:    true,
							Computed:    true,
						},
						"policy": schema.StringAttribute{
							Description: "Condition matching policy: 'all' (AND) or 'any' (OR)",
							Optional:    true,
							Computed:    true,
						},
					},
					Blocks: map[string]schema.Block{
						"job_options": schema.ListNestedBlock{
							Description: "Job options to pass to the triggered job",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Description: "Option name",
										Required:    true,
									},
									"value": schema.StringAttribute{
										Description: "Option value",
										Required:    true,
									},
								},
							},
						},
						"conditions": schema.ListNestedBlock{
							Description: "Conditions that must be met for this rule to trigger",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"path": schema.StringAttribute{
										Description: "JSONPath to the event field to evaluate",
										Required:    true,
									},
									"condition": schema.StringAttribute{
										Description: "Condition operator: equals, contains, dateTimeAfter, dateTimeBefore, exists, isA",
										Required:    true,
									},
									"value": schema.StringAttribute{
										Description: "Value to compare against",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// configToAPI converts the Terraform config model to the API's JSON format (excluding rules).
func (r *webhookResource) configToAPI(ctx context.Context, config types.Object) (map[string]interface{}, error) {
	if config.IsNull() || config.IsUnknown() {
		return nil, nil
	}

	var configModel webhookConfigModel
	if diags := config.As(ctx, &configModel, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("failed to extract config model")
	}

	apiConfig := make(map[string]interface{})

	if !configModel.LogLevel.IsNull() && !configModel.LogLevel.IsUnknown() {
		apiConfig["logLevel"] = configModel.LogLevel.ValueString()
	}
	if !configModel.JobID.IsNull() && !configModel.JobID.IsUnknown() {
		apiConfig["jobId"] = configModel.JobID.ValueString()
	}
	if !configModel.ArgString.IsNull() && !configModel.ArgString.IsUnknown() {
		apiConfig["argString"] = configModel.ArgString.ValueString()
	}
	if !configModel.NodeFilter.IsNull() && !configModel.NodeFilter.IsUnknown() {
		apiConfig["nodeFilter"] = configModel.NodeFilter.ValueString()
	}
	if !configModel.AsUser.IsNull() && !configModel.AsUser.IsUnknown() {
		apiConfig["asUser"] = configModel.AsUser.ValueString()
	}
	if !configModel.KeyStoragePath.IsNull() && !configModel.KeyStoragePath.IsUnknown() {
		apiConfig["keyStoragePath"] = configModel.KeyStoragePath.ValueString()
	}
	if !configModel.BatchKey.IsNull() && !configModel.BatchKey.IsUnknown() {
		apiConfig["batchKey"] = configModel.BatchKey.ValueString()
	}
	if !configModel.EventIDKey.IsNull() && !configModel.EventIDKey.IsUnknown() {
		apiConfig["eventIdKey"] = configModel.EventIDKey.ValueString()
	}
	if !configModel.ReturnProcessing.IsNull() && !configModel.ReturnProcessing.IsUnknown() {
		apiConfig["returnProcessingInfo"] = configModel.ReturnProcessing.ValueBool()
	}
	if !configModel.Secret.IsNull() && !configModel.Secret.IsUnknown() {
		apiConfig["secret"] = configModel.Secret.ValueString()
	}
	if !configModel.AutoSubscribe.IsNull() && !configModel.AutoSubscribe.IsUnknown() {
		apiConfig["autoSubscribe"] = configModel.AutoSubscribe.ValueBool()
	}

	return apiConfig, nil
}

// rulesToAPI converts top-level rules to the API's config rules format.
func (r *webhookResource) rulesToAPI(ctx context.Context, rules types.List) ([]map[string]interface{}, error) {
	var ruleItems []webhookAdvancedRule
	if diags := rules.ElementsAs(ctx, &ruleItems, false); diags.HasError() {
		return nil, fmt.Errorf("failed to extract rules")
	}

	apiRules := make([]map[string]interface{}, 0, len(ruleItems))
	for _, rule := range ruleItems {
		apiRule := map[string]interface{}{
			"name":    rule.Name.ValueString(),
			"jobId":   rule.JobID.ValueString(),
			"enabled": true,
			"policy":  "all",
		}

		if !rule.JobName.IsNull() && !rule.JobName.IsUnknown() {
			apiRule["jobName"] = rule.JobName.ValueString()
		}
		if !rule.NodeFilter.IsNull() && !rule.NodeFilter.IsUnknown() {
			apiRule["nodeFilter"] = rule.NodeFilter.ValueString()
		}
		if !rule.User.IsNull() && !rule.User.IsUnknown() {
			apiRule["user"] = rule.User.ValueString()
		}
		if !rule.Enabled.IsNull() && !rule.Enabled.IsUnknown() {
			apiRule["enabled"] = rule.Enabled.ValueBool()
		}
		if !rule.Policy.IsNull() && !rule.Policy.IsUnknown() {
			apiRule["policy"] = rule.Policy.ValueString()
		}

		if !rule.JobOptions.IsNull() && !rule.JobOptions.IsUnknown() {
			var jobOptions []webhookJobOption
			if diags := rule.JobOptions.ElementsAs(ctx, &jobOptions, false); diags.HasError() {
				return nil, fmt.Errorf("failed to extract job options")
			}
			apiJobOptions := make([]map[string]interface{}, 0, len(jobOptions))
			for _, opt := range jobOptions {
				apiJobOptions = append(apiJobOptions, map[string]interface{}{
					"name":  opt.Name.ValueString(),
					"value": opt.Value.ValueString(),
				})
			}
			apiRule["jobOptions"] = apiJobOptions
		}

		if !rule.Conditions.IsNull() && !rule.Conditions.IsUnknown() {
			var conditions []webhookCondition
			if diags := rule.Conditions.ElementsAs(ctx, &conditions, false); diags.HasError() {
				return nil, fmt.Errorf("failed to extract conditions")
			}
			apiConditions := make([]map[string]interface{}, 0, len(conditions))
			for _, cond := range conditions {
				apiConditions = append(apiConditions, map[string]interface{}{
					"path":      cond.Path.ValueString(),
					"condition": cond.Condition.ValueString(),
					"value":     cond.Value.ValueString(),
				})
			}
			apiRule["conditions"] = apiConditions
		}

		apiRules = append(apiRules, apiRule)
	}

	return apiRules, nil
}

// apiToConfig converts the API's JSON config format to the Terraform config model (excluding rules).
func (r *webhookResource) apiToConfig(ctx context.Context, apiConfig map[string]interface{}) types.Object {
	configAttrs := map[string]attr.Value{
		"log_level":              types.StringNull(),
		"job_id":                 types.StringNull(),
		"arg_string":             types.StringNull(),
		"node_filter":            types.StringNull(),
		"as_user":                types.StringNull(),
		"key_storage_path":       types.StringNull(),
		"batch_key":              types.StringNull(),
		"event_id_key":           types.StringNull(),
		"return_processing_info": types.BoolNull(),
		"secret":                 types.StringNull(),
		"auto_subscribe":         types.BoolNull(),
	}

	if apiConfig == nil || len(apiConfig) == 0 {
		configObj, _ := types.ObjectValue(webhookConfigAttrTypes(), configAttrs)
		return configObj
	}

	if logLevel, ok := apiConfig["logLevel"].(string); ok {
		configAttrs["log_level"] = types.StringValue(logLevel)
	}
	if jobID, ok := apiConfig["jobId"].(string); ok {
		configAttrs["job_id"] = types.StringValue(jobID)
	}
	if argString, ok := apiConfig["argString"].(string); ok {
		configAttrs["arg_string"] = types.StringValue(argString)
	}
	if nodeFilter, ok := apiConfig["nodeFilter"].(string); ok {
		configAttrs["node_filter"] = types.StringValue(nodeFilter)
	}
	if asUser, ok := apiConfig["asUser"].(string); ok {
		configAttrs["as_user"] = types.StringValue(asUser)
	}
	if keyStoragePath, ok := apiConfig["keyStoragePath"].(string); ok {
		configAttrs["key_storage_path"] = types.StringValue(keyStoragePath)
	}
	if batchKey, ok := apiConfig["batchKey"].(string); ok {
		configAttrs["batch_key"] = types.StringValue(batchKey)
	}
	if eventIDKey, ok := apiConfig["eventIdKey"].(string); ok {
		configAttrs["event_id_key"] = types.StringValue(eventIDKey)
	}
	if returnProcessing, ok := apiConfig["returnProcessingInfo"].(bool); ok {
		configAttrs["return_processing_info"] = types.BoolValue(returnProcessing)
	}
	if secret, ok := apiConfig["secret"].(string); ok {
		configAttrs["secret"] = types.StringValue(secret)
	}
	if autoSubscribe, ok := apiConfig["autoSubscribe"].(bool); ok {
		configAttrs["auto_subscribe"] = types.BoolValue(autoSubscribe)
	}

	configObj, _ := types.ObjectValue(webhookConfigAttrTypes(), configAttrs)
	return configObj
}

// apiToRules extracts rules from an API config map and returns a top-level types.List.
func (r *webhookResource) apiToRules(apiConfig map[string]interface{}) types.List {
	rulesInterface, ok := apiConfig["rules"].([]interface{})
	if !ok || len(rulesInterface) == 0 {
		return types.ListNull(webhookRuleAttrType())
	}

	ruleElements := make([]attr.Value, 0, len(rulesInterface))
	for _, ruleInterface := range rulesInterface {
		ruleMap, ok := ruleInterface.(map[string]interface{})
		if !ok {
			continue
		}

		jobName := getStringFromMap(ruleMap, "jobName")
		jobNameAttr := types.StringNull()
		if jobName != "" {
			jobNameAttr = types.StringValue(jobName)
		}

		nodeFilter := getStringFromMap(ruleMap, "nodeFilter")
		nodeFilterAttr := types.StringNull()
		if nodeFilter != "" {
			nodeFilterAttr = types.StringValue(nodeFilter)
		}

		user := getStringFromMap(ruleMap, "user")
		userAttr := types.StringNull()
		if user != "" {
			userAttr = types.StringValue(user)
		}

		enabledAttr := types.BoolValue(getBoolFromMap(ruleMap, "enabled", true))

		policy := getStringFromMap(ruleMap, "policy")
		policyAttr := types.StringValue("all")
		if policy != "" {
			policyAttr = types.StringValue(policy)
		}

		ruleAttrs := map[string]attr.Value{
			"name":        types.StringValue(getStringFromMap(ruleMap, "name")),
			"job_id":      types.StringValue(getStringFromMap(ruleMap, "jobId")),
			"job_name":    jobNameAttr,
			"node_filter": nodeFilterAttr,
			"user":        userAttr,
			"enabled":     enabledAttr,
			"policy":      policyAttr,
			"job_options": types.ListNull(webhookJobOptionAttrType()),
			"conditions":  types.ListNull(webhookConditionAttrType()),
		}

		if jobOptionsInterface, ok := ruleMap["jobOptions"].([]interface{}); ok && len(jobOptionsInterface) > 0 {
			jobOptionElements := make([]attr.Value, 0, len(jobOptionsInterface))
			for _, optInterface := range jobOptionsInterface {
				optMap, ok := optInterface.(map[string]interface{})
				if !ok {
					continue
				}
				optObj, _ := types.ObjectValue(
					webhookJobOptionAttrTypes(),
					map[string]attr.Value{
						"name":  types.StringValue(getStringFromMap(optMap, "name")),
						"value": types.StringValue(getStringFromMap(optMap, "value")),
					},
				)
				jobOptionElements = append(jobOptionElements, optObj)
			}
			if len(jobOptionElements) > 0 {
				ruleAttrs["job_options"], _ = types.ListValue(webhookJobOptionAttrType(), jobOptionElements)
			}
		}

		if conditionsInterface, ok := ruleMap["conditions"].([]interface{}); ok && len(conditionsInterface) > 0 {
			conditionElements := make([]attr.Value, 0, len(conditionsInterface))
			for _, condInterface := range conditionsInterface {
				condMap, ok := condInterface.(map[string]interface{})
				if !ok {
					continue
				}
				condObj, _ := types.ObjectValue(
					webhookConditionAttrTypes(),
					map[string]attr.Value{
						"path":      types.StringValue(getStringFromMap(condMap, "path")),
						"condition": types.StringValue(getStringFromMap(condMap, "condition")),
						"value":     types.StringValue(getStringFromMap(condMap, "value")),
					},
				)
				conditionElements = append(conditionElements, condObj)
			}
			if len(conditionElements) > 0 {
				ruleAttrs["conditions"], _ = types.ListValue(webhookConditionAttrType(), conditionElements)
			}
		}

		ruleObj, _ := types.ObjectValue(webhookRuleAttrTypes(), ruleAttrs)
		ruleElements = append(ruleElements, ruleObj)
	}

	if len(ruleElements) == 0 {
		return types.ListNull(webhookRuleAttrType())
	}

	rulesList, _ := types.ListValue(webhookRuleAttrType(), ruleElements)
	return rulesList
}

// buildAPIConfig combines the config block and top-level rules into the API config map.
func (r *webhookResource) buildAPIConfig(ctx context.Context, config types.Object, rules types.List) (map[string]interface{}, error) {
	apiConfig := make(map[string]interface{})

	if !config.IsNull() && !config.IsUnknown() {
		configFields, err := r.configToAPI(ctx, config)
		if err != nil {
			return nil, err
		}
		for k, v := range configFields {
			apiConfig[k] = v
		}
	}

	if !rules.IsNull() && !rules.IsUnknown() {
		apiRules, err := r.rulesToAPI(ctx, rules)
		if err != nil {
			return nil, err
		}
		if len(apiRules) > 0 {
			apiConfig["rules"] = apiRules
		}
	}

	return apiConfig, nil
}

// Helper functions for attribute types
func webhookConfigAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"log_level":              types.StringType,
		"job_id":                 types.StringType,
		"arg_string":             types.StringType,
		"node_filter":            types.StringType,
		"as_user":                types.StringType,
		"key_storage_path":       types.StringType,
		"batch_key":              types.StringType,
		"event_id_key":           types.StringType,
		"return_processing_info": types.BoolType,
		"secret":                 types.StringType,
		"auto_subscribe":         types.BoolType,
	}
}

func webhookRuleAttrType() types.ObjectType {
	return types.ObjectType{AttrTypes: webhookRuleAttrTypes()}
}

func webhookRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"job_id":      types.StringType,
		"job_name":    types.StringType,
		"node_filter": types.StringType,
		"user":        types.StringType,
		"enabled":     types.BoolType,
		"policy":      types.StringType,
		"job_options": types.ListType{ElemType: webhookJobOptionAttrType()},
		"conditions":  types.ListType{ElemType: webhookConditionAttrType()},
	}
}

func webhookJobOptionAttrType() types.ObjectType {
	return types.ObjectType{AttrTypes: webhookJobOptionAttrTypes()}
}

func webhookJobOptionAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":  types.StringType,
		"value": types.StringType,
	}
}

func webhookConditionAttrType() types.ObjectType {
	return types.ObjectType{AttrTypes: webhookConditionAttrTypes()}
}

func webhookConditionAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"path":      types.StringType,
		"condition": types.StringType,
		"value":     types.StringType,
	}
}

// Helper functions to safely extract values from maps
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultVal
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := plan.Project.ValueString()
	webhookData := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"user":        plan.User.ValueString(),
		"roles":       plan.Roles.ValueString(),
		"eventPlugin": plan.EventPlugin.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
		"project":     project,
	}

	apiConfig, err := r.buildAPIConfig(ctx, plan.Config, plan.Rules)
	if err != nil {
		resp.Diagnostics.AddError("Error Converting Config", fmt.Sprintf("Could not convert config to API format: %s", err))
		return
	}
	if len(apiConfig) > 0 {
		webhookData["config"] = apiConfig
	}

	apiCtx := r.clients.ctx
	apiResp, httpResp, err := r.clients.V2.WebhookAPI.CreateWebhookDocs(apiCtx, project).Body(webhookData).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Webhook",
			fmt.Sprintf("Could not create webhook in project %s: %s\n\nAPI Response: %+v", project, err, httpResp),
		)
		return
	}

	var createdUUID string
	if uuid, ok := apiResp["uuid"].(string); ok {
		createdUUID = uuid
	} else {
		resp.Diagnostics.AddError("Error Parsing Webhook Response", "Webhook UUID not found in API response")
		return
	}

	webhooksList, _, err := r.clients.V2.WebhookAPI.List(apiCtx, project).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Webhooks",
			fmt.Sprintf("Could not list webhooks after creation in project %s: %s", project, err),
		)
		return
	}

	var webhookID string
	var authToken string
	found := false
	for _, webhook := range webhooksList {
		if uuid, ok := webhook["uuid"].(string); ok && uuid == createdUUID {
			if id, ok := webhook["id"].(float64); ok {
				webhookID = fmt.Sprintf("%.0f", id)
			} else if id, ok := webhook["id"].(int); ok {
				webhookID = fmt.Sprintf("%d", id)
			} else if id, ok := webhook["id"].(string); ok {
				webhookID = id
			}
			if token, ok := webhook["authToken"].(string); ok {
				authToken = token
			}
			found = true
			break
		}
	}

	if !found || webhookID == "" {
		resp.Diagnostics.AddError("Error Finding Webhook", fmt.Sprintf("Could not find created webhook with UUID %s", createdUUID))
		return
	}

	fullWebhook, httpResp, err := r.clients.V2.WebhookAPI.Get(apiCtx, project, webhookID).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Created Webhook",
			fmt.Sprintf("Webhook created but could not read details: %s\n\nAPI Response: %+v", err, httpResp),
		)
		return
	}

	state := webhookResourceModel{
		ID:          types.StringValue(webhookID),
		Project:     plan.Project,
		Name:        types.StringValue(getStringFromMap(fullWebhook, "name")),
		User:        types.StringValue(getStringFromMap(fullWebhook, "user")),
		Roles:       types.StringValue(normalizeRoles(getStringFromMap(fullWebhook, "roles"))),
		EventPlugin: types.StringValue(getStringFromMap(fullWebhook, "eventPlugin")),
		Enabled:     types.BoolValue(getBoolFromMap(fullWebhook, "enabled", true)),
		AuthToken:   types.StringValue(authToken),
	}

	if configInterface, ok := fullWebhook["config"].(map[string]interface{}); ok && len(configInterface) > 0 {
		if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
			state.Config = r.apiToConfig(ctx, configInterface)
		} else {
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
		state.Rules = r.apiToRules(configInterface)
	} else {
		state.Config = types.ObjectNull(webhookConfigAttrTypes())
		state.Rules = types.ListNull(webhookRuleAttrType())
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()
	id := state.ID.ValueString()

	apiCtx := r.clients.ctx
	apiResp, httpResp, err := r.clients.V2.WebhookAPI.Get(apiCtx, project, id).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Webhook",
			fmt.Sprintf("Could not read webhook %s in project %s: %s", id, project, err),
		)
		return
	}

	if name, ok := apiResp["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if user, ok := apiResp["user"].(string); ok {
		state.User = types.StringValue(user)
	}
	if roles, ok := apiResp["roles"].(string); ok {
		state.Roles = types.StringValue(normalizeRoles(roles))
	}
	if eventPlugin, ok := apiResp["eventPlugin"].(string); ok {
		state.EventPlugin = types.StringValue(eventPlugin)
	}
	if enabled, ok := apiResp["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(enabled)
	}

	if configData, ok := apiResp["config"].(map[string]interface{}); ok {
		hasConfigData := len(configData) > 0 &&
			(configData["jobId"] != nil ||
				configData["logLevel"] != nil ||
				configData["argString"] != nil ||
				configData["nodeFilter"] != nil ||
				configData["asUser"] != nil ||
				configData["keyStoragePath"] != nil ||
				configData["batchKey"] != nil ||
				configData["eventIdKey"] != nil ||
				configData["returnProcessingInfo"] != nil ||
				configData["secret"] != nil ||
				configData["autoSubscribe"] != nil)

		if hasConfigData {
			state.Config = r.apiToConfig(ctx, configData)
		} else if state.Config.IsNull() {
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		}

		// Always attempt to read rules from config
		apiRules := r.apiToRules(configData)
		if !apiRules.IsNull() {
			state.Rules = apiRules
		} else if state.Rules.IsNull() {
			state.Rules = types.ListNull(webhookRuleAttrType())
		}
	} else {
		if state.Config.IsNull() {
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
		if state.Rules.IsNull() {
			state.Rules = types.ListNull(webhookRuleAttrType())
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := plan.Project.ValueString()
	webhookData := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"user":        plan.User.ValueString(),
		"roles":       plan.Roles.ValueString(),
		"eventPlugin": plan.EventPlugin.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
		"project":     project,
	}

	apiConfig, err := r.buildAPIConfig(ctx, plan.Config, plan.Rules)
	if err != nil {
		resp.Diagnostics.AddError("Error Converting Config", fmt.Sprintf("Could not convert config to API format: %s", err))
		return
	}
	if len(apiConfig) > 0 {
		webhookData["config"] = apiConfig
	}

	apiCtx := r.clients.ctx
	id := state.ID.ValueString()
	_, httpResp, err := r.clients.V2.WebhookAPI.Save(apiCtx, project, id).Body(webhookData).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Webhook",
			fmt.Sprintf("Could not update webhook %s in project %s: %s\n\nAPI Response: %+v", id, project, err, httpResp),
		)
		return
	}

	apiResp, _, err := r.clients.V2.WebhookAPI.Get(apiCtx, project, id).Execute()
	if err != nil {
		resp.Diagnostics.AddError("Error Reading Webhook After Update", fmt.Sprintf("Could not read webhook %s after update: %s", id, err))
		return
	}

	if name, ok := apiResp["name"].(string); ok {
		plan.Name = types.StringValue(name)
	}
	if user, ok := apiResp["user"].(string); ok {
		plan.User = types.StringValue(user)
	}
	if roles, ok := apiResp["roles"].(string); ok {
		plan.Roles = types.StringValue(normalizeRoles(roles))
	}
	if eventPlugin, ok := apiResp["eventPlugin"].(string); ok {
		plan.EventPlugin = types.StringValue(eventPlugin)
	}
	if enabled, ok := apiResp["enabled"].(bool); ok {
		plan.Enabled = types.BoolValue(enabled)
	}

	if configData, ok := apiResp["config"].(map[string]interface{}); ok && len(configData) > 0 {
		if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
			parsedConfig := r.apiToConfig(ctx, configData)
			if !parsedConfig.IsNull() {
				plan.Config = parsedConfig
			}
		}
		apiRules := r.apiToRules(configData)
		if !apiRules.IsNull() {
			plan.Rules = apiRules
		}
	} else {
		if plan.Config.IsNull() || plan.Config.IsUnknown() {
			plan.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
		if plan.Rules.IsNull() || plan.Rules.IsUnknown() {
			plan.Rules = types.ListNull(webhookRuleAttrType())
		}
	}

	plan.AuthToken = state.AuthToken
	plan.ID = state.ID

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := state.Project.ValueString()
	id := state.ID.ValueString()

	apiCtx := r.clients.ctx
	_, httpResp, err := r.clients.V2.WebhookAPI.Remove(apiCtx, project, id).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Webhook",
			fmt.Sprintf("Could not delete webhook %s in project %s: %s", id, project, err),
		)
		return
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import identifier in format 'project/webhook-id', got: %s", req.ID),
		)
		return
	}

	project := parts[0]
	id := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project"), project)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
