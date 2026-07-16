package rundeck

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Command nested block schema
func jobCommandNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Workflow command steps",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"description": schema.StringAttribute{
					Optional: true,
				},
				"shell_command": schema.StringAttribute{
					Optional:    true,
					Description: "A shell command to execute",
				},
				"inline_script": schema.StringAttribute{
					Optional:    true,
					Description: "A script to execute inline",
				},
				"script_url": schema.StringAttribute{
					Optional:    true,
					Description: "URL of a script to download and execute",
				},
				"script_file": schema.StringAttribute{
					Optional:    true,
					Description: "Path to a script file to execute",
				},
				"script_file_args": schema.StringAttribute{
					Optional:    true,
					Description: "Arguments to pass to the script file",
				},
				"expand_token_in_script_file": schema.BoolAttribute{
					Optional: true,
				},
				"file_extension": schema.StringAttribute{
					Optional:    true,
					Description: "File extension for temporary script file",
				},
				"keep_going_on_success": schema.BoolAttribute{
					Optional: true,
				},
			},
			Blocks: map[string]schema.Block{
				"plugins": schema.ListNestedBlock{
					Description: "Command-level plugins (e.g., log filters)",
					NestedObject: schema.NestedBlockObject{
						Blocks: map[string]schema.Block{
							"log_filter_plugin": schema.ListNestedBlock{
								Description: "Log filter plugin configuration",
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required: true,
										},
										"config": schema.MapAttribute{
											Optional:    true,
											ElementType: types.StringType,
										},
									},
								},
							},
						},
					},
				},
				"script_interpreter": schema.ListNestedBlock{
					Description: "Script interpreter configuration",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"invocation_string": schema.StringAttribute{
								Optional: true,
							},
							"args_quoted": schema.BoolAttribute{
								Optional: true,
							},
						},
					},
				},
				"job": schema.ListNestedBlock{
					Description: "Reference to another job to execute. Either uuid or name must be specified.",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "UUID of the job to reference (immutable, preferred). Can reference another rundeck_job's id attribute.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"name": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Name of the job to reference. Required if uuid is not specified.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"group_name": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Group path of the job. Used with name-based references.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"project_name": schema.StringAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Project containing the job. Used with name-based references. Also populated from the API for uuid-based references, since Rundeck always resolves a job reference to a specific project.",
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.UseStateForUnknown(),
								},
							},
							"run_for_each_node": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Whether the referenced job runs once per node (node step). Alias of node_step; both map to the API's nodeStep flag, and run_for_each_node takes precedence if both are set.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"node_step": schema.BoolAttribute{
								Optional:    true,
								Computed:    true,
								Description: "Run the referenced job as a node step (once per node). Alias of run_for_each_node.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
							"args": schema.StringAttribute{
								Optional: true,
							},
							"import_options": schema.BoolAttribute{
								Optional: true,
							},
							"child_nodes": schema.BoolAttribute{
								Optional: true,
							},
							"fail_on_disable": schema.BoolAttribute{
								Optional: true,
							},
							"ignore_notifications": schema.BoolAttribute{
								Optional: true,
							},
						},
						Blocks: map[string]schema.Block{
							"node_filters": schema.ListNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"filter": schema.StringAttribute{
											Optional: true,
										},
										"exclude_filter": schema.StringAttribute{
											Optional: true,
										},
										"exclude_precedence": schema.BoolAttribute{
											Optional: true,
										},
									},
									Blocks: map[string]schema.Block{
										"dispatch": schema.ListNestedBlock{
											Description: "Dispatch configuration for node execution",
											NestedObject: schema.NestedBlockObject{
												Attributes: map[string]schema.Attribute{
													"thread_count": schema.Int64Attribute{
														Optional:    true,
														Description: "Number of threads to use for parallel execution",
													},
													"keep_going": schema.BoolAttribute{
														Optional:    true,
														Description: "Continue execution on remaining nodes after a failure",
													},
													"rank_attribute": schema.StringAttribute{
														Optional:    true,
														Description: "Node attribute to use for ranking",
													},
													"rank_order": schema.StringAttribute{
														Optional:    true,
														Description: "Rank order: ascending or descending",
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
				"step_plugin": schema.ListNestedBlock{
					Description: "Workflow step plugin",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Required: true,
							},
							"config": schema.MapAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
				"node_step_plugin": schema.ListNestedBlock{
					Description: "Node step plugin",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Required: true,
							},
							"config": schema.MapAttribute{
								Optional:    true,
								ElementType: types.StringType,
							},
						},
					},
				},
				"error_handler": schema.ListNestedBlock{
					Description: "Error handler for this command",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"description": schema.StringAttribute{
								Optional: true,
							},
							"shell_command": schema.StringAttribute{
								Optional: true,
							},
							"inline_script": schema.StringAttribute{
								Optional: true,
							},
							"script_url": schema.StringAttribute{
								Optional: true,
							},
							"script_file": schema.StringAttribute{
								Optional: true,
							},
							"script_file_args": schema.StringAttribute{
								Optional: true,
							},
							"expand_token_in_script_file": schema.BoolAttribute{
								Optional: true,
							},
							"file_extension": schema.StringAttribute{
								Optional: true,
							},
							"keep_going_on_success": schema.BoolAttribute{
								Optional: true,
								Computed: true,
								Description: "Continue workflow even if error handler succeeds. Rundeck's API omits this " +
									"field entirely when false, so it is Computed to avoid drift/inconsistent-apply " +
									"errors on that default.",
								PlanModifiers: []planmodifier.Bool{
									boolplanmodifier.UseStateForUnknown(),
								},
							},
						},
						Blocks: map[string]schema.Block{
							"script_interpreter": schema.ListNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"invocation_string": schema.StringAttribute{
											Optional: true,
										},
										"args_quoted": schema.BoolAttribute{
											Optional: true,
										},
									},
								},
							},
							"job": schema.ListNestedBlock{
								Description: "Reference to another job to execute as error handler. Either uuid or name must be specified.",
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"uuid": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "UUID of the job to reference (immutable, preferred). Can reference another rundeck_job's id attribute.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"name": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Name of the job to reference. Required if uuid is not specified.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"group_name": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Group path of the job. Used with name-based references.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"project_name": schema.StringAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Project containing the job. Used with name-based references. Also populated from the API for uuid-based references, since Rundeck always resolves a job reference to a specific project.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"run_for_each_node": schema.BoolAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Whether the referenced job runs once per node (node step). Alias of node_step; both map to the API's nodeStep flag, and run_for_each_node takes precedence if both are set.",
											PlanModifiers: []planmodifier.Bool{
												boolplanmodifier.UseStateForUnknown(),
											},
										},
										"node_step": schema.BoolAttribute{
											Optional:    true,
											Computed:    true,
											Description: "Run the referenced job as a node step (once per node). Alias of run_for_each_node.",
											PlanModifiers: []planmodifier.Bool{
												boolplanmodifier.UseStateForUnknown(),
											},
										},
										"args": schema.StringAttribute{
											Optional: true,
										},
										"import_options": schema.BoolAttribute{
											Optional: true,
										},
										"child_nodes": schema.BoolAttribute{
											Optional: true,
										},
										"fail_on_disable": schema.BoolAttribute{
											Optional: true,
										},
										"ignore_notifications": schema.BoolAttribute{
											Optional: true,
										},
									},
									Blocks: map[string]schema.Block{
										"node_filters": schema.ListNestedBlock{
											NestedObject: schema.NestedBlockObject{
												Attributes: map[string]schema.Attribute{
													"filter": schema.StringAttribute{
														Optional: true,
													},
													"exclude_filter": schema.StringAttribute{
														Optional: true,
													},
													"exclude_precedence": schema.BoolAttribute{
														Optional: true,
													},
												},
												Blocks: map[string]schema.Block{
													"dispatch": schema.ListNestedBlock{
														NestedObject: schema.NestedBlockObject{
															Attributes: map[string]schema.Attribute{
																"thread_count": schema.Int64Attribute{
																	Optional: true,
																},
																"keep_going": schema.BoolAttribute{
																	Optional: true,
																},
																"rank_attribute": schema.StringAttribute{
																	Optional: true,
																},
																"rank_order": schema.StringAttribute{
																	Optional: true,
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
							"step_plugin": schema.ListNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required: true,
										},
										"config": schema.MapAttribute{
											Optional:    true,
											ElementType: types.StringType,
										},
									},
								},
							},
							"node_step_plugin": schema.ListNestedBlock{
								NestedObject: schema.NestedBlockObject{
									Attributes: map[string]schema.Attribute{
										"type": schema.StringAttribute{
											Required: true,
										},
										"config": schema.MapAttribute{
											Optional:    true,
											ElementType: types.StringType,
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

// Option nested block schema
func jobOptionNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Job options/parameters",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Required: true,
				},
				"label": schema.StringAttribute{
					Optional: true,
				},
				"default_value": schema.StringAttribute{
					Optional: true,
				},
				"value_choices": schema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
				},
				"value_choices_url": schema.StringAttribute{
					Optional: true,
				},
				"sort_values": schema.BoolAttribute{
					Optional: true,
				},
				"require_predefined_choice": schema.BoolAttribute{
					Optional: true,
					Computed: true,
				},
				"validation_regex": schema.StringAttribute{
					Optional: true,
				},
				"description": schema.StringAttribute{
					Optional: true,
				},
				"required": schema.BoolAttribute{
					Optional: true,
				},
				"allow_multiple_values": schema.BoolAttribute{
					Optional: true,
				},
				"multi_value_delimiter": schema.StringAttribute{
					Optional: true,
				},
				"obscure_input": schema.BoolAttribute{
					Optional: true,
				},
				"exposed_to_scripts": schema.BoolAttribute{
					Optional: true,
				},
				"storage_path": schema.StringAttribute{
					Optional: true,
				},
				"hidden": schema.BoolAttribute{
					Optional: true,
				},
				"type": schema.StringAttribute{
					Optional: true,
				},
				"is_date": schema.BoolAttribute{
					Optional: true,
				},
				"date_format": schema.StringAttribute{
					Optional: true,
				},
			},
		},
	}
}

// Notification nested block schema
// Using ListNestedBlock - notifications must be defined in alphabetical order by type
// to prevent plan drift (Rundeck API returns them sorted alphabetically)
func jobNotificationNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Job notifications",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required:    true,
					Description: "Notification type (on_success, on_failure, on_start, on_avg_duration, on_retryable_failure)",
				},
				"webhook_urls": schema.ListAttribute{
					ElementType: types.StringType,
					Optional:    true,
					Description: "Webhook URLs for webhook notifications",
				},
				"format": schema.StringAttribute{
					Optional:    true,
					Description: "Format for webhook notifications (json, xml, form)",
				},
				"http_method": schema.StringAttribute{
					Optional:    true,
					Description: "HTTP method for webhook notifications (GET, POST, PUT, DELETE)",
				},
			},
			Blocks: map[string]schema.Block{
				"email": schema.ListNestedBlock{
					Description: "Email notification configuration",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"recipients": schema.ListAttribute{
								ElementType: types.StringType,
								Required:    true,
								Description: "Email recipients",
							},
							"subject": schema.StringAttribute{
								Optional:    true,
								Description: "Email subject",
							},
							"attach_log": schema.BoolAttribute{
								Optional:    true,
								Description: "Attach execution log to email",
							},
						},
					},
				},
				"plugin": schema.ListNestedBlock{
					Description: "Plugin notification configuration",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"type": schema.StringAttribute{
								Required:    true,
								Description: "Plugin type",
							},
							"config": schema.MapAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Description: "Plugin configuration",
							},
						},
					},
				},
			},
		},
	}
}

// Log limit nested block schema
func jobLogLimitNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Log output limit configuration",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"output": schema.StringAttribute{
					Required:    true,
					Description: "Maximum log output (e.g. '100', '100/node', '100MB')",
				},
				"action": schema.StringAttribute{
					Required:    true,
					Description: "Action when limit reached: halt or truncate",
				},
				"status": schema.StringAttribute{
					Required:    true,
					Description: "Job status when limit reached: failed, aborted, or custom",
				},
			},
		},
	}
}

// Orchestrator nested block schema
func jobOrchestratorNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Orchestrator configuration for node execution ordering",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required:    true,
					Description: "Orchestrator type: subset, rankTiered, maxPercentage, orchestrator-highest-lowest-attribute",
				},
				"count": schema.Int64Attribute{
					Optional:    true,
					Description: "Number of nodes for subset orchestrator",
				},
				"percent": schema.Int64Attribute{
					Optional:    true,
					Description: "Percentage of nodes for maxPercentage orchestrator",
				},
				"attribute": schema.StringAttribute{
					Optional:    true,
					Description: "Node attribute for ranking",
				},
				"sort": schema.StringAttribute{
					Optional:    true,
					Description: "Sort order: highest or lowest",
				},
			},
		},
	}
}

// Global log filter nested block schema
func jobGlobalLogFilterNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Global log filter plugins",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required: true,
				},
				"config": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
				},
			},
		},
	}
}

// Project schedule nested block schema
func jobProjectScheduleNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Project-level schedule configuration",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					Required: true,
				},
				"job_options": schema.StringAttribute{
					Optional: true,
				},
			},
		},
	}
}

// Execution lifecycle plugin nested block schema
func jobExecutionLifecyclePluginNestedBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description: "Execution lifecycle plugins",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required: true,
				},
				"config": schema.MapAttribute{
					Optional:    true,
					ElementType: types.StringType,
				},
			},
		},
	}
}

// nodeFilterDispatchObjectType mirrors the "dispatch" nested block on a job
// reference's node_filters.
var nodeFilterDispatchObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"thread_count":   types.Int64Type,
		"keep_going":     types.BoolType,
		"rank_attribute": types.StringType,
		"rank_order":     types.StringType,
	},
}

// nodeFilterObjectType mirrors the "node_filters" nested block on a job reference.
var nodeFilterObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"filter":             types.StringType,
		"exclude_filter":     types.StringType,
		"exclude_precedence": types.BoolType,
		"dispatch":           types.ListType{ElemType: nodeFilterDispatchObjectType},
	},
}

// jobRefObjectType mirrors the "job" nested block (a reference to another job),
// used both directly under a command and under its error_handler.
var jobRefObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"uuid":                 types.StringType,
		"name":                 types.StringType,
		"group_name":           types.StringType,
		"project_name":         types.StringType,
		"run_for_each_node":    types.BoolType,
		"node_step":            types.BoolType,
		"args":                 types.StringType,
		"import_options":       types.BoolType,
		"child_nodes":          types.BoolType,
		"fail_on_disable":      types.BoolType,
		"ignore_notifications": types.BoolType,
		"node_filters":         types.ListType{ElemType: nodeFilterObjectType},
	},
}

// scriptInterpreterObjectType mirrors the "script_interpreter" nested block.
var scriptInterpreterObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"invocation_string": types.StringType,
		"args_quoted":       types.BoolType,
	},
}

// stepPluginObjectType mirrors the "step_plugin"/"node_step_plugin" nested blocks.
var stepPluginObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":   types.StringType,
		"config": types.MapType{ElemType: types.StringType},
	},
}

// logFilterPluginObjectType mirrors the "log_filter_plugin" nested block under
// a command's "plugins" block.
var logFilterPluginObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":   types.StringType,
		"config": types.MapType{ElemType: types.StringType},
	},
}

// commandPluginsObjectType mirrors a command's "plugins" nested block.
var commandPluginsObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"log_filter_plugin": types.ListType{ElemType: logFilterPluginObjectType},
	},
}

// errorHandlerObjectType mirrors the "error_handler" nested block on a command.
var errorHandlerObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"description":                 types.StringType,
		"shell_command":               types.StringType,
		"inline_script":               types.StringType,
		"script_url":                  types.StringType,
		"script_file":                 types.StringType,
		"script_file_args":            types.StringType,
		"expand_token_in_script_file": types.BoolType,
		"file_extension":              types.StringType,
		"keep_going_on_success":       types.BoolType,
		"script_interpreter":          types.ListType{ElemType: scriptInterpreterObjectType},
		"job":                         types.ListType{ElemType: jobRefObjectType},
		"step_plugin":                 types.ListType{ElemType: stepPluginObjectType},
		"node_step_plugin":            types.ListType{ElemType: stepPluginObjectType},
	},
}

// Command model type
var commandObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"description":                 types.StringType,
		"shell_command":               types.StringType,
		"inline_script":               types.StringType,
		"script_url":                  types.StringType,
		"script_file":                 types.StringType,
		"script_file_args":            types.StringType,
		"expand_token_in_script_file": types.BoolType,
		"file_extension":              types.StringType,
		"keep_going_on_success":       types.BoolType,
		"plugins":                     types.ListType{ElemType: commandPluginsObjectType},
		"script_interpreter":          types.ListType{ElemType: scriptInterpreterObjectType},
		"job":                         types.ListType{ElemType: jobRefObjectType},
		"step_plugin":                 types.ListType{ElemType: stepPluginObjectType},
		"node_step_plugin":            types.ListType{ElemType: stepPluginObjectType},
		"error_handler":               types.ListType{ElemType: errorHandlerObjectType},
	},
}
