package rundeck

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
								Description: "UUID of the job to reference (immutable, preferred). Can reference another rundeck_job's id attribute.",
							},
							"name": schema.StringAttribute{
								Optional:    true,
								Description: "Name of the job to reference. Required if uuid is not specified.",
							},
							"group_name": schema.StringAttribute{
								Optional:    true,
								Description: "Group path of the job. Used with name-based references.",
							},
							"project_name": schema.StringAttribute{
								Optional:    true,
								Description: "Project containing the job. Used with name-based references.",
							},
							"run_for_each_node": schema.BoolAttribute{
								Optional: true,
							},
							"node_step": schema.BoolAttribute{
								Optional:    true,
								Description: "Run the referenced job as a node step (once per node)",
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
								Optional:    true,
								Description: "Continue workflow even if error handler succeeds",
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
											Description: "UUID of the job to reference (immutable, preferred). Can reference another rundeck_job's id attribute.",
										},
										"name": schema.StringAttribute{
											Optional:    true,
											Description: "Name of the job to reference. Required if uuid is not specified.",
										},
										"group_name": schema.StringAttribute{
											Optional:    true,
											Description: "Group path of the job. Used with name-based references.",
										},
										"project_name": schema.StringAttribute{
											Optional:    true,
											Description: "Project containing the job. Used with name-based references.",
										},
										"run_for_each_node": schema.BoolAttribute{
											Optional: true,
										},
										"node_step": schema.BoolAttribute{
											Optional:    true,
											Description: "Run the referenced job as a node step (once per node)",
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
// Changed to ListAttribute with CustomType for semantic equality support
// This enables semantic equality so notification order doesn't cause plan drift
func jobNotificationNestedBlock() schema.ListAttribute {
	// Define the notification object type - must match convertNotificationsFromJSON
	notificationObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":         types.StringType,
			"webhook_urls": types.ListType{ElemType: types.StringType},
			"format":       types.StringType,
			"http_method":  types.StringType,
			"email": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"recipients": types.ListType{ElemType: types.StringType},
						"subject":    types.StringType,
						"attach_log": types.BoolType,
					},
				},
			},
			"plugin": types.ListType{
				ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					},
				},
			},
		},
	}

	// Create custom list type with semantic equality
	customListType := NotificationListType{ListType: basetypes.ListType{ElemType: notificationObjectType}}

	return schema.ListAttribute{
		Description: "Job notifications",
		ElementType: notificationObjectType,
		Optional:    true,
		CustomType:  customListType,
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
		"plugins":                     types.ListType{ElemType: types.ObjectType{}},
		"script_interpreter":          types.ListType{ElemType: types.ObjectType{}},
		"job":                         types.ListType{ElemType: types.ObjectType{}},
		"step_plugin":                 types.ListType{ElemType: types.ObjectType{}},
		"node_step_plugin":            types.ListType{ElemType: types.ObjectType{}},
		"error_handler":               types.ListType{ElemType: types.ObjectType{}},
	},
}
