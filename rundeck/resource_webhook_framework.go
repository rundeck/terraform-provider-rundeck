package rundeck

import (
	"context"
	"fmt"
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
}

// webhookConfigModel represents the webhook configuration for different plugin types
type webhookConfigModel struct {
	// Simple plugin fields (log-webhook-event)
	LogLevel types.String `tfsdk:"log_level"`

	// webhook-run-job plugin fields
	JobID      types.String `tfsdk:"job_id"`
	ArgString  types.String `tfsdk:"arg_string"`
	NodeFilter types.String `tfsdk:"node_filter"`
	AsUser     types.String `tfsdk:"as_user"`

	// Advanced-run-job and Enterprise plugin fields
	KeyStoragePath   types.String `tfsdk:"key_storage_path"`
	BatchKey         types.String `tfsdk:"batch_key"`
	EventIDKey       types.String `tfsdk:"event_id_key"`
	ReturnProcessing types.Bool   `tfsdk:"return_processing_info"`
	Rules            types.List   `tfsdk:"rules"` // []webhookAdvancedRule

	// GitHub-specific fields
	Secret types.String `tfsdk:"secret"` // GitHub webhook secret for authentication

	// AWS SNS-specific fields
	AutoSubscribe types.Bool `tfsdk:"auto_subscribe"` // Auto-confirm SNS subscription
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
	JobOptions types.List   `tfsdk:"job_options"` // []webhookJobOption
	Conditions types.List   `tfsdk:"conditions"`  // []webhookCondition
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
					// log-webhook-event plugin fields
					"log_level": schema.StringAttribute{
						Description: "Log level for log-webhook-event plugin (INFO, DEBUG, WARN, ERROR)",
						Optional:    true,
					},
					// webhook-run-job plugin fields
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
					// Advanced-run-job and Enterprise plugin fields
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
					// GitHub-specific fields
					"secret": schema.StringAttribute{
						Description: "GitHub webhook secret for authentication (github-webhook plugin only)",
						Optional:    true,
						Sensitive:   true,
					},
					// AWS SNS-specific fields
					"auto_subscribe": schema.BoolAttribute{
						Description: "Automatically confirm SNS subscription (aws-sns-webhook plugin only)",
						Optional:    true,
					},
				},
				Blocks: map[string]schema.Block{
					"rules": schema.ListNestedBlock{
						Description: "Action rules for advanced-run-job plugin",
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

// configToAPI converts the Terraform config model to the API's JSON format
func (r *webhookResource) configToAPI(ctx context.Context, config types.Object) (map[string]interface{}, error) {
	if config.IsNull() || config.IsUnknown() {
		return nil, nil
	}

	var configModel webhookConfigModel
	if diags := config.As(ctx, &configModel, basetypes.ObjectAsOptions{}); diags.HasError() {
		return nil, fmt.Errorf("failed to extract config model")
	}

	apiConfig := make(map[string]interface{})

	// Handle log-webhook-event plugin fields
	if !configModel.LogLevel.IsNull() && !configModel.LogLevel.IsUnknown() {
		apiConfig["logLevel"] = configModel.LogLevel.ValueString()
	}

	// Handle webhook-run-job plugin fields
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

	// Handle advanced-run-job plugin fields (also used by Enterprise plugins)
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

	// Handle GitHub-specific fields
	if !configModel.Secret.IsNull() && !configModel.Secret.IsUnknown() {
		apiConfig["secret"] = configModel.Secret.ValueString()
	}

	// Handle AWS SNS-specific fields
	if !configModel.AutoSubscribe.IsNull() && !configModel.AutoSubscribe.IsUnknown() {
		apiConfig["autoSubscribe"] = configModel.AutoSubscribe.ValueBool()
	}

	if !configModel.Rules.IsNull() && !configModel.Rules.IsUnknown() {
		var rules []webhookAdvancedRule
		if diags := configModel.Rules.ElementsAs(ctx, &rules, false); diags.HasError() {
			return nil, fmt.Errorf("failed to extract rules")
		}

		apiRules := make([]map[string]interface{}, 0, len(rules))
		for _, rule := range rules {
			apiRule := map[string]interface{}{
				"name":    rule.Name.ValueString(),
				"jobId":   rule.JobID.ValueString(),
				"enabled": true,  // default
				"policy":  "all", // default
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

			// Handle job options
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

			// Handle conditions
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
		apiConfig["rules"] = apiRules
	}

	return apiConfig, nil
}

// apiToConfig converts the API's JSON config format to the Terraform config model
func (r *webhookResource) apiToConfig(ctx context.Context, apiConfig map[string]interface{}) types.Object {
	// Always return a valid object for blocks (never null), even if empty
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
		"rules":                  types.ListNull(webhookRuleAttrType()),
	}

	// If apiConfig is nil or empty, return object with all null values
	if apiConfig == nil || len(apiConfig) == 0 {
		configObj, _ := types.ObjectValue(webhookConfigAttrTypes(), configAttrs)
		return configObj
	}

	// Handle log-webhook-event plugin fields
	if logLevel, ok := apiConfig["logLevel"].(string); ok {
		configAttrs["log_level"] = types.StringValue(logLevel)
	}

	// Handle webhook-run-job plugin fields
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

	// Handle advanced-run-job plugin fields (also used by Enterprise plugins)
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

	// Handle GitHub-specific fields (note: secret is write-only, not returned by API)
	if secret, ok := apiConfig["secret"].(string); ok {
		configAttrs["secret"] = types.StringValue(secret)
	}

	// Handle AWS SNS-specific fields
	if autoSubscribe, ok := apiConfig["autoSubscribe"].(bool); ok {
		configAttrs["auto_subscribe"] = types.BoolValue(autoSubscribe)
	}

	// Handle rules array
	if rulesInterface, ok := apiConfig["rules"].([]interface{}); ok && len(rulesInterface) > 0 {
		ruleElements := make([]attr.Value, 0, len(rulesInterface))

		for _, ruleInterface := range rulesInterface {
			ruleMap, ok := ruleInterface.(map[string]interface{})
			if !ok {
				continue
			}

			// Handle computed fields appropriately
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

			// enabled defaults to true if not present
			enabledAttr := types.BoolValue(getBoolFromMap(ruleMap, "enabled", true))

			// policy defaults to "all" if not present
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

			// Handle job options
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

			// Handle conditions
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

		if len(ruleElements) > 0 {
			rulesList, _ := types.ListValue(webhookRuleAttrType(), ruleElements)
			configAttrs["rules"] = rulesList
		}
	}

	configObj, _ := types.ObjectValue(webhookConfigAttrTypes(), configAttrs)
	return configObj
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
		"rules":                  types.ListType{ElemType: webhookRuleAttrType()},
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

	// Build webhook data
	project := plan.Project.ValueString()
	webhookData := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"user":        plan.User.ValueString(),
		"roles":       plan.Roles.ValueString(),
		"eventPlugin": plan.EventPlugin.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
		"project":     project,
	}

	// Add config if provided
	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		apiConfig, err := r.configToAPI(ctx, plan.Config)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Converting Config",
				fmt.Sprintf("Could not convert config to API format: %s", err),
			)
			return
		}
		webhookData["config"] = apiConfig
	}

	// Create webhook via API (use authenticated context)
	apiCtx := r.clients.ctx
	apiResp, httpResp, err := r.clients.V2.WebhookAPI.CreateWebhookDocs(apiCtx, project).Body(webhookData).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Webhook",
			fmt.Sprintf("Could not create webhook in project %s: %s\n\nAPI Response: %+v", project, err, httpResp),
		)
		return
	}

	// Extract UUID from create response
	var createdUUID string
	if uuid, ok := apiResp["uuid"].(string); ok {
		createdUUID = uuid
	} else {
		resp.Diagnostics.AddError(
			"Error Parsing Webhook Response",
			"Webhook UUID not found in API response",
		)
		return
	}

	// List webhooks to find the integer ID for the UUID we just created
	// This is necessary because Create returns UUID but subsequent operations need integer ID
	webhooksList, _, err := r.clients.V2.WebhookAPI.List(apiCtx, project).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Webhooks",
			fmt.Sprintf("Could not list webhooks after creation in project %s: %s", project, err),
		)
		return
	}

	// Find our webhook by UUID
	var webhookID string
	var authToken string
	found := false
	for _, webhook := range webhooksList {
		if uuid, ok := webhook["uuid"].(string); ok && uuid == createdUUID {
			// Extract integer ID (may be float64 from JSON)
			if id, ok := webhook["id"].(float64); ok {
				webhookID = fmt.Sprintf("%.0f", id)
			} else if id, ok := webhook["id"].(int); ok {
				webhookID = fmt.Sprintf("%d", id)
			} else if id, ok := webhook["id"].(string); ok {
				webhookID = id
			}

			// Extract auth token
			if token, ok := webhook["authToken"].(string); ok {
				authToken = token
			}
			found = true
			break
		}
	}

	if !found || webhookID == "" {
		resp.Diagnostics.AddError(
			"Error Finding Webhook",
			fmt.Sprintf("Could not find created webhook with UUID %s", createdUUID),
		)
		return
	}

	// Now fetch the full webhook details to populate computed fields
	fullWebhook, httpResp, err := r.clients.V2.WebhookAPI.Get(apiCtx, project, webhookID).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Created Webhook",
			fmt.Sprintf("Webhook created but could not read details: %s\n\nAPI Response: %+v", err, httpResp),
		)
		return
	}

	// Build state from API response
	state := webhookResourceModel{
		ID:          types.StringValue(webhookID),
		Project:     plan.Project,
		Name:        types.StringValue(getStringFromMap(fullWebhook, "name")),
		User:        types.StringValue(getStringFromMap(fullWebhook, "user")),
		Roles:       types.StringValue(getStringFromMap(fullWebhook, "roles")),
		EventPlugin: types.StringValue(getStringFromMap(fullWebhook, "eventPlugin")),
		Enabled:     types.BoolValue(getBoolFromMap(fullWebhook, "enabled", true)),
		AuthToken:   types.StringValue(authToken), // From list response, not returned by Get
	}

	// Handle config - only add to state if it was in the plan or has meaningful data
	if configInterface, ok := fullWebhook["config"].(map[string]interface{}); ok && len(configInterface) > 0 {
		// Check if plan had config
		if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
			// Config was in plan, so populate from API
			state.Config = r.apiToConfig(ctx, configInterface)
		} else {
			// Config not in plan, don't add empty config to state
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
	} else {
		// No config in API response
		if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
			// Config was in plan but API returned nothing, set to null
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		} else {
			// No config in plan and none in API, leave as null
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
	}

	// Set state
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

	// Get webhook from API (use authenticated context)
	apiCtx := r.clients.ctx
	apiResp, httpResp, err := r.clients.V2.WebhookAPI.Get(apiCtx, project, id).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			// Webhook no longer exists
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Webhook",
			fmt.Sprintf("Could not read webhook %s in project %s: %s", id, project, err),
		)
		return
	}

	// Update state from API response
	if name, ok := apiResp["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if user, ok := apiResp["user"].(string); ok {
		state.User = types.StringValue(user)
	}
	if roles, ok := apiResp["roles"].(string); ok {
		state.Roles = types.StringValue(roles)
	}
	if eventPlugin, ok := apiResp["eventPlugin"].(string); ok {
		state.EventPlugin = types.StringValue(eventPlugin)
	}
	if enabled, ok := apiResp["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(enabled)
	}

	// Parse config - only set if it has meaningful data
	if configData, ok := apiResp["config"].(map[string]interface{}); ok {
		// Check if config has any non-empty values
		hasData := false
		if len(configData) > 0 {
			hasData = configData["jobId"] != nil ||
				configData["logLevel"] != nil ||
				configData["argString"] != nil ||
				configData["nodeFilter"] != nil ||
				configData["asUser"] != nil ||
				configData["keyStoragePath"] != nil ||
				configData["batchKey"] != nil ||
				configData["eventIdKey"] != nil ||
				configData["returnProcessingInfo"] != nil ||
				configData["secret"] != nil ||
				configData["autoSubscribe"] != nil

			// Check rules separately to avoid panic
			if rules, ok := configData["rules"].([]interface{}); ok && len(rules) > 0 {
				hasData = true
			}
		}

		if hasData {
			// API returned meaningful config data, update state
			state.Config = r.apiToConfig(ctx, configData)
		} else {
			// API returned empty config
			// Due to known API limitation (Save doesn't return updated config),
			// preserve existing config if it was previously set
			if state.Config.IsNull() {
				// Config wasn't set before and API doesn't have it, keep null
				state.Config = types.ObjectNull(webhookConfigAttrTypes())
			}
			// else: preserve existing state.Config (API didn't return it due to known limitation)
		}
	} else {
		// No config in API response
		if state.Config.IsNull() {
			state.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
		// else: preserve existing state.Config
	}

	// Note: authToken is not returned by the Read API, so we preserve the existing value in state

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

	// Build update data
	project := plan.Project.ValueString()
	webhookData := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"user":        plan.User.ValueString(),
		"roles":       plan.Roles.ValueString(),
		"eventPlugin": plan.EventPlugin.ValueString(),
		"enabled":     plan.Enabled.ValueBool(),
		"project":     project,
	}

	// Add config if provided
	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		apiConfig, err := r.configToAPI(ctx, plan.Config)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Converting Config",
				fmt.Sprintf("Could not convert config to API format: %s", err),
			)
			return
		}
		webhookData["config"] = apiConfig
	}

	// Update webhook via API (use authenticated context)
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

	// Read back from API to get actual state after update
	apiResp, _, err := r.clients.V2.WebhookAPI.Get(apiCtx, project, id).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Webhook After Update",
			fmt.Sprintf("Could not read webhook %s after update: %s", id, err),
		)
		return
	}

	// Update plan with values from API
	if name, ok := apiResp["name"].(string); ok {
		plan.Name = types.StringValue(name)
	}
	if user, ok := apiResp["user"].(string); ok {
		plan.User = types.StringValue(user)
	}
	if roles, ok := apiResp["roles"].(string); ok {
		plan.Roles = types.StringValue(roles)
	}
	if eventPlugin, ok := apiResp["eventPlugin"].(string); ok {
		plan.EventPlugin = types.StringValue(eventPlugin)
	}
	if enabled, ok := apiResp["enabled"].(bool); ok {
		plan.Enabled = types.BoolValue(enabled)
	}

	// Parse config if present - handle cases where API may not return all config fields
	if configData, ok := apiResp["config"].(map[string]interface{}); ok && len(configData) > 0 {
		// Check if plan had config
		if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
			// Config was in plan, convert from API
			parsedConfig := r.apiToConfig(ctx, configData)

			// For simple plugins (log-webhook-event), the API may not return config
			// In that case, preserve the plan's config
			if !parsedConfig.IsNull() {
				plan.Config = parsedConfig
			}
			// If API returned empty/null config but plan had config, keep plan's config
			// This handles the known API limitation where Save doesn't return updated config
		}
	} else {
		// No config in API response
		// If plan had config, keep it (API limitation - doesn't return updated config reliably)
		// If plan didn't have config, set to null
		if plan.Config.IsNull() || plan.Config.IsUnknown() {
			plan.Config = types.ObjectNull(webhookConfigAttrTypes())
		}
		// else: keep plan.Config as-is since API didn't return it
	}

	// Preserve the auth token from state (not returned by API)
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

	// Delete webhook via API (use authenticated context)
	apiCtx := r.clients.ctx
	_, httpResp, err := r.clients.V2.WebhookAPI.Remove(apiCtx, project, id).Execute()
	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			// Already deleted, no error
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
	// Split "project/id"
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
