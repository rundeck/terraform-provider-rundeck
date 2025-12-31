package rundeck

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// convertCommandsToJSON converts Framework command list to JSON array
func convertCommandsToJSON(ctx context.Context, commandsList types.List) ([]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if commandsList.IsNull() || commandsList.IsUnknown() {
		return nil, diags
	}

	var commands []types.Object
	diags.Append(commandsList.ElementsAs(ctx, &commands, false)...)
	if diags.HasError() {
		return nil, diags
	}

	result := make([]interface{}, 0, len(commands))
	for _, cmdObj := range commands {
		cmdMap := make(map[string]interface{})
		attrs := cmdObj.Attributes()

		// Simple string fields
		if v, ok := attrs["description"].(types.String); ok && !v.IsNull() {
			cmdMap["description"] = v.ValueString()
		}
		if v, ok := attrs["shell_command"].(types.String); ok && !v.IsNull() {
			cmdMap["exec"] = v.ValueString()
		}
		if v, ok := attrs["inline_script"].(types.String); ok && !v.IsNull() {
			cmdMap["script"] = v.ValueString()
		}
		if v, ok := attrs["script_url"].(types.String); ok && !v.IsNull() {
			cmdMap["scripturl"] = v.ValueString()
		}
		if v, ok := attrs["script_file"].(types.String); ok && !v.IsNull() {
			cmdMap["scriptfile"] = v.ValueString()
		}
		if v, ok := attrs["script_file_args"].(types.String); ok && !v.IsNull() {
			cmdMap["args"] = v.ValueString()
		}
		if v, ok := attrs["file_extension"].(types.String); ok && !v.IsNull() {
			cmdMap["fileExtension"] = v.ValueString()
		}

		// Boolean fields
		if v, ok := attrs["expand_token_in_script_file"].(types.Bool); ok && !v.IsNull() {
			cmdMap["expandTokenInScriptFile"] = v.ValueBool()
		}
		if v, ok := attrs["keep_going_on_success"].(types.Bool); ok && !v.IsNull() {
			cmdMap["keepgoingOnSuccess"] = v.ValueBool()
		}

		// Handle nested script_interpreter
		// Note: Rundeck expects TWO separate fields:
		// 1. "interpreterArgsQuoted" (boolean) at command level
		// 2. "scriptInterpreter" (string) at command level - just the invocation string
		if v, ok := attrs["script_interpreter"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var interpreters []types.Object
			diags.Append(v.ElementsAs(ctx, &interpreters, false)...)
			if len(interpreters) > 0 {
				interpAttrs := interpreters[0].Attributes()

				// Set interpreterArgsQuoted as a boolean at command level
				if aq, ok := interpAttrs["args_quoted"].(types.Bool); ok && !aq.IsNull() {
					cmdMap["interpreterArgsQuoted"] = aq.ValueBool()
				}

				// Set scriptInterpreter as a string (just the invocation) at command level
				if inv, ok := interpAttrs["invocation_string"].(types.String); ok && !inv.IsNull() {
					cmdMap["scriptInterpreter"] = inv.ValueString()
				}
			}
		}

		// Handle nested job reference
		if v, ok := attrs["job"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var jobs []types.Object
			diags.Append(v.ElementsAs(ctx, &jobs, false)...)
			if len(jobs) > 0 {
				jobAttrs := jobs[0].Attributes()
				jobMap := make(map[string]interface{})

				// UUID reference (preferred, immutable)
				if uuid, ok := jobAttrs["uuid"].(types.String); ok && !uuid.IsNull() {
					jobMap["uuid"] = uuid.ValueString()
				}
				// Name-based reference (backward compatible)
				if name, ok := jobAttrs["name"].(types.String); ok && !name.IsNull() {
					jobMap["name"] = name.ValueString()
				}
				if group, ok := jobAttrs["group_name"].(types.String); ok && !group.IsNull() {
					jobMap["group"] = group.ValueString()
				}
				if project, ok := jobAttrs["project_name"].(types.String); ok && !project.IsNull() {
					jobMap["project"] = project.ValueString()
				}
				if rfn, ok := jobAttrs["run_for_each_node"].(types.Bool); ok && !rfn.IsNull() {
					jobMap["runForEachNode"] = rfn.ValueBool()
				}
				if ns, ok := jobAttrs["node_step"].(types.Bool); ok && !ns.IsNull() {
					// API expects string "true" or "false" for nodeStep
					if ns.ValueBool() {
						jobMap["nodeStep"] = "true"
					} else {
						jobMap["nodeStep"] = "false"
					}
				}
				if args, ok := jobAttrs["args"].(types.String); ok && !args.IsNull() {
					jobMap["args"] = args.ValueString()
				}
				if io, ok := jobAttrs["import_options"].(types.Bool); ok && !io.IsNull() {
					jobMap["importOptions"] = io.ValueBool()
				}
				if cn, ok := jobAttrs["child_nodes"].(types.Bool); ok && !cn.IsNull() {
					jobMap["childNodes"] = cn.ValueBool()
				}
				if fod, ok := jobAttrs["fail_on_disable"].(types.Bool); ok && !fod.IsNull() {
					jobMap["failOnDisable"] = fod.ValueBool()
				}
				if ign, ok := jobAttrs["ignore_notifications"].(types.Bool); ok && !ign.IsNull() {
					jobMap["ignoreNotifications"] = ign.ValueBool()
				}

				// Handle node_filters if present
				if nf, ok := jobAttrs["node_filters"].(types.List); ok && !nf.IsNull() && !nf.IsUnknown() {
					var nodeFilters []types.Object
					diags.Append(nf.ElementsAs(ctx, &nodeFilters, false)...)
					if len(nodeFilters) > 0 {
						nfAttrs := nodeFilters[0].Attributes()
						nfMap := make(map[string]interface{})
						if filter, ok := nfAttrs["filter"].(types.String); ok && !filter.IsNull() {
							nfMap["filter"] = filter.ValueString()
						}
						if exclude, ok := nfAttrs["exclude_filter"].(types.String); ok && !exclude.IsNull() {
							nfMap["excludeFilter"] = exclude.ValueString()
						}
						if prec, ok := nfAttrs["exclude_precedence"].(types.Bool); ok && !prec.IsNull() {
							nfMap["excludePrecedence"] = prec.ValueBool()
						}

						// Handle dispatch configuration
						if disp, ok := nfAttrs["dispatch"].(types.List); ok && !disp.IsNull() && !disp.IsUnknown() {
							var dispatches []types.Object
							diags.Append(disp.ElementsAs(ctx, &dispatches, false)...)
							if len(dispatches) > 0 {
								dispAttrs := dispatches[0].Attributes()
								dispMap := make(map[string]interface{})

								if tc, ok := dispAttrs["thread_count"].(types.Int64); ok && !tc.IsNull() {
									dispMap["threadcount"] = int(tc.ValueInt64())
								}
								if kg, ok := dispAttrs["keep_going"].(types.Bool); ok && !kg.IsNull() {
									dispMap["keepgoing"] = kg.ValueBool()
								}
								if ra, ok := dispAttrs["rank_attribute"].(types.String); ok && !ra.IsNull() {
									dispMap["rankAttribute"] = ra.ValueString()
								}
								if ro, ok := dispAttrs["rank_order"].(types.String); ok && !ro.IsNull() {
									dispMap["rankOrder"] = ro.ValueString()
								}

								if len(dispMap) > 0 {
									nfMap["dispatch"] = dispMap
								}
							}
						}

						jobMap["nodefilters"] = nfMap
					}
				}

				cmdMap["jobref"] = jobMap
			}
		}

		// Handle step_plugin (workflow step)
		// API expects: type and configuration at command level, nodeStep: false
		if v, ok := attrs["step_plugin"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var plugins []types.Object
			diags.Append(v.ElementsAs(ctx, &plugins, false)...)
			if len(plugins) > 0 {
				pluginAttrs := plugins[0].Attributes()

				// Set type at command level
				if ptype, ok := pluginAttrs["type"].(types.String); ok && !ptype.IsNull() {
					cmdMap["type"] = ptype.ValueString()
				}

				// Set configuration at command level
				if config, ok := pluginAttrs["config"].(types.Map); ok && !config.IsNull() {
					var configMap map[string]string
					diags.Append(config.ElementsAs(ctx, &configMap, false)...)
					cmdMap["configuration"] = configMap
				}

				// Workflow steps run once (not per node)
				cmdMap["nodeStep"] = false
			}
		}

		// Handle node_step_plugin
		// API expects: type and configuration at command level, nodeStep: true
		if v, ok := attrs["node_step_plugin"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var plugins []types.Object
			diags.Append(v.ElementsAs(ctx, &plugins, false)...)
			if len(plugins) > 0 {
				pluginAttrs := plugins[0].Attributes()

				// Set type at command level
				if ptype, ok := pluginAttrs["type"].(types.String); ok && !ptype.IsNull() {
					cmdMap["type"] = ptype.ValueString()
				}

				// Set configuration at command level
				if config, ok := pluginAttrs["config"].(types.Map); ok && !config.IsNull() {
					var configMap map[string]string
					diags.Append(config.ElementsAs(ctx, &configMap, false)...)
					cmdMap["configuration"] = configMap
				}

				// Node steps run on each node
				cmdMap["nodeStep"] = true
			}
		}

		// Handle error_handler (simplified - doesn't recurse infinitely)
		if v, ok := attrs["error_handler"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var handlers []types.Object
			diags.Append(v.ElementsAs(ctx, &handlers, false)...)
			if len(handlers) > 0 {
				handlerAttrs := handlers[0].Attributes()
				handlerMap := make(map[string]interface{})

				// String fields
				if desc, ok := handlerAttrs["description"].(types.String); ok && !desc.IsNull() {
					handlerMap["description"] = desc.ValueString()
				}
				if sc, ok := handlerAttrs["shell_command"].(types.String); ok && !sc.IsNull() {
					handlerMap["exec"] = sc.ValueString()
				}
				if script, ok := handlerAttrs["inline_script"].(types.String); ok && !script.IsNull() {
					handlerMap["script"] = script.ValueString()
				}
				if scriptUrl, ok := handlerAttrs["script_url"].(types.String); ok && !scriptUrl.IsNull() {
					handlerMap["scripturl"] = scriptUrl.ValueString()
				}
				if scriptFile, ok := handlerAttrs["script_file"].(types.String); ok && !scriptFile.IsNull() {
					handlerMap["scriptfile"] = scriptFile.ValueString()
				}
				if args, ok := handlerAttrs["script_file_args"].(types.String); ok && !args.IsNull() {
					handlerMap["args"] = args.ValueString()
				}
				if ext, ok := handlerAttrs["file_extension"].(types.String); ok && !ext.IsNull() {
					handlerMap["fileExtension"] = ext.ValueString()
				}

				// Boolean fields
				if expand, ok := handlerAttrs["expand_token_in_script_file"].(types.Bool); ok && !expand.IsNull() {
					handlerMap["expandTokenInScriptFile"] = expand.ValueBool()
				}
				if keepGoing, ok := handlerAttrs["keep_going_on_success"].(types.Bool); ok && !keepGoing.IsNull() {
					handlerMap["keepgoingOnSuccess"] = keepGoing.ValueBool()
				}

				// Handle nested job reference in error_handler
				if jobList, ok := handlerAttrs["job"].(types.List); ok && !jobList.IsNull() && !jobList.IsUnknown() {
					var jobs []types.Object
					diags.Append(jobList.ElementsAs(ctx, &jobs, false)...)
					if len(jobs) > 0 {
						jobAttrs := jobs[0].Attributes()
						jobMap := make(map[string]interface{})

						// UUID reference (preferred, immutable)
						if uuid, ok := jobAttrs["uuid"].(types.String); ok && !uuid.IsNull() {
							jobMap["uuid"] = uuid.ValueString()
						}
						// Name-based reference (backward compatible)
						if name, ok := jobAttrs["name"].(types.String); ok && !name.IsNull() {
							jobMap["name"] = name.ValueString()
						}
						if group, ok := jobAttrs["group_name"].(types.String); ok && !group.IsNull() {
							jobMap["group"] = group.ValueString()
						}
						if project, ok := jobAttrs["project_name"].(types.String); ok && !project.IsNull() {
							jobMap["project"] = project.ValueString()
						}
						if rfn, ok := jobAttrs["run_for_each_node"].(types.Bool); ok && !rfn.IsNull() {
							jobMap["runForEachNode"] = rfn.ValueBool()
						}

						// Determine if parent command is a node step or workflow step
						// Error handlers must match the parent command type
						isParentNodeStep := false
						if _, hasNodeStepPlugin := attrs["node_step_plugin"]; hasNodeStepPlugin {
							isParentNodeStep = true
						} else if _, hasStepPlugin := attrs["step_plugin"]; hasStepPlugin {
							isParentNodeStep = false // step_plugin is workflow step
						} else {
							// shell_command, inline_script, etc. are workflow steps (not node steps)
							isParentNodeStep = false
						}

						if ns, ok := jobAttrs["node_step"].(types.Bool); ok && !ns.IsNull() {
							// API expects string "true" or "false" for nodeStep
							if ns.ValueBool() {
								jobMap["nodeStep"] = "true"
							} else {
								jobMap["nodeStep"] = "false"
							}
						} else {
							// If node_step not specified, infer from parent command type
							// Rundeck requires error handler job references to match parent command type
							if isParentNodeStep {
								jobMap["nodeStep"] = "true"
							} else {
								jobMap["nodeStep"] = "false"
							}
						}
						if args, ok := jobAttrs["args"].(types.String); ok && !args.IsNull() {
							jobMap["args"] = args.ValueString()
						}
						if io, ok := jobAttrs["import_options"].(types.Bool); ok && !io.IsNull() {
							jobMap["importOptions"] = io.ValueBool()
						}
						if cn, ok := jobAttrs["child_nodes"].(types.Bool); ok && !cn.IsNull() {
							jobMap["childNodes"] = cn.ValueBool()
						}
						if fod, ok := jobAttrs["fail_on_disable"].(types.Bool); ok && !fod.IsNull() {
							jobMap["failOnDisable"] = fod.ValueBool()
						}
						if ign, ok := jobAttrs["ignore_notifications"].(types.Bool); ok && !ign.IsNull() {
							jobMap["ignoreNotifications"] = ign.ValueBool()
						}

						// Handle node_filters if present
						if nf, ok := jobAttrs["node_filters"].(types.List); ok && !nf.IsNull() && !nf.IsUnknown() {
							var nodeFilters []types.Object
							diags.Append(nf.ElementsAs(ctx, &nodeFilters, false)...)
							if len(nodeFilters) > 0 {
								nfAttrs := nodeFilters[0].Attributes()
								nfMap := make(map[string]interface{})
								if filter, ok := nfAttrs["filter"].(types.String); ok && !filter.IsNull() {
									nfMap["filter"] = filter.ValueString()
								}
								if exclude, ok := nfAttrs["exclude_filter"].(types.String); ok && !exclude.IsNull() {
									nfMap["excludeFilter"] = exclude.ValueString()
								}
								if prec, ok := nfAttrs["exclude_precedence"].(types.Bool); ok && !prec.IsNull() {
									nfMap["excludePrecedence"] = prec.ValueBool()
								}

								// Handle dispatch configuration
								if disp, ok := nfAttrs["dispatch"].(types.List); ok && !disp.IsNull() && !disp.IsUnknown() {
									var dispatches []types.Object
									diags.Append(disp.ElementsAs(ctx, &dispatches, false)...)
									if len(dispatches) > 0 {
										dispAttrs := dispatches[0].Attributes()
										dispMap := make(map[string]interface{})

										if tc, ok := dispAttrs["thread_count"].(types.Int64); ok && !tc.IsNull() {
											dispMap["threadcount"] = int(tc.ValueInt64())
										}
										if kg, ok := dispAttrs["keep_going"].(types.Bool); ok && !kg.IsNull() {
											dispMap["keepgoing"] = kg.ValueBool()
										}
										if ra, ok := dispAttrs["rank_attribute"].(types.String); ok && !ra.IsNull() {
											dispMap["rankAttribute"] = ra.ValueString()
										}
										if ro, ok := dispAttrs["rank_order"].(types.String); ok && !ro.IsNull() {
											dispMap["rankOrder"] = ro.ValueString()
										}

										if len(dispMap) > 0 {
											nfMap["dispatch"] = dispMap
										}
									}
								}

								if len(nfMap) > 0 {
									jobMap["nodefilters"] = nfMap
								}
							}
						}

						if len(jobMap) > 0 {
							handlerMap["jobref"] = jobMap
						}
					}
				}

				cmdMap["errorhandler"] = handlerMap
			}
		}

		// Handle command-level plugins (e.g., log_filter_plugin)
		if v, ok := attrs["plugins"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			pluginsMap, pluginDiags := convertCommandPluginsToJSON(ctx, v)
			diags.Append(pluginDiags...)
			if len(pluginsMap) > 0 {
				cmdMap["plugins"] = pluginsMap
			}
		}

		result = append(result, cmdMap)
	}

	return result, diags
}

// convertOptionsToJSON converts Framework option list to JSON array
func convertOptionsToJSON(ctx context.Context, optionsList types.List) ([]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if optionsList.IsNull() || optionsList.IsUnknown() {
		return nil, diags
	}

	var options []types.Object
	diags.Append(optionsList.ElementsAs(ctx, &options, false)...)
	if diags.HasError() {
		return nil, diags
	}

	result := make([]interface{}, 0, len(options))
	for _, optObj := range options {
		attrs := optObj.Attributes()

		optMap := make(map[string]interface{})

		// Name is required
		if name, ok := attrs["name"].(types.String); ok && !name.IsNull() {
			optMap["name"] = name.ValueString()
		} else {
			continue // Skip if no name
		}

		// Map Terraform field names to Rundeck JSON field names
		if v, ok := attrs["label"].(types.String); ok && !v.IsNull() {
			optMap["label"] = v.ValueString()
		}
		// Rundeck uses "value" for the default value, not "defaultValue"
		if v, ok := attrs["default_value"].(types.String); ok && !v.IsNull() {
			optMap["value"] = v.ValueString()
		}
		if v, ok := attrs["description"].(types.String); ok && !v.IsNull() {
			optMap["description"] = v.ValueString()
		}
		if v, ok := attrs["validation_regex"].(types.String); ok && !v.IsNull() {
			optMap["regex"] = v.ValueString()
		}
		if v, ok := attrs["value_choices_url"].(types.String); ok && !v.IsNull() {
			optMap["valuesUrl"] = v.ValueString()
		}
		if v, ok := attrs["multi_value_delimiter"].(types.String); ok && !v.IsNull() {
			optMap["delimiter"] = v.ValueString()
		}
		if v, ok := attrs["storage_path"].(types.String); ok && !v.IsNull() {
			optMap["storagePath"] = v.ValueString()
		}
		if v, ok := attrs["type"].(types.String); ok && !v.IsNull() {
			optMap["type"] = v.ValueString()
		}
		if v, ok := attrs["date_format"].(types.String); ok && !v.IsNull() {
			optMap["dateFormat"] = v.ValueString()
		}

		// Boolean fields
		if v, ok := attrs["required"].(types.Bool); ok && !v.IsNull() {
			optMap["required"] = v.ValueBool()
		}
		if v, ok := attrs["sort_values"].(types.Bool); ok && !v.IsNull() {
			optMap["sortValues"] = v.ValueBool()
		}
		if v, ok := attrs["require_predefined_choice"].(types.Bool); ok && !v.IsNull() {
			optMap["enforcedValues"] = v.ValueBool()
		}
		if v, ok := attrs["allow_multiple_values"].(types.Bool); ok && !v.IsNull() {
			optMap["multivalued"] = v.ValueBool()
		}
		if v, ok := attrs["obscure_input"].(types.Bool); ok && !v.IsNull() {
			optMap["secure"] = v.ValueBool()
		}
		if v, ok := attrs["exposed_to_scripts"].(types.Bool); ok && !v.IsNull() {
			optMap["valueExposed"] = v.ValueBool()
		}
		if v, ok := attrs["hidden"].(types.Bool); ok && !v.IsNull() {
			optMap["hidden"] = v.ValueBool()
		}
		if v, ok := attrs["is_date"].(types.Bool); ok && !v.IsNull() {
			optMap["isDate"] = v.ValueBool()
		}

		// value_choices list
		if v, ok := attrs["value_choices"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var choices []string
			diags.Append(v.ElementsAs(ctx, &choices, false)...)
			if len(choices) > 0 {
				optMap["values"] = choices
			}
		}

		result = append(result, optMap)
	}

	return result, diags
}

// convertNotificationsToJSON converts Framework notification list to JSON structure
func convertNotificationsToJSON(ctx context.Context, notificationsList types.List) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if notificationsList.IsNull() || notificationsList.IsUnknown() {
		return nil, diags
	}

	var notifications []types.Object
	diags.Append(notificationsList.ElementsAs(ctx, &notifications, false)...)
	if diags.HasError() {
		return nil, diags
	}

	// Sort notifications alphabetically by type to match the order we read them back
	// This prevents plan drift when users define notifications in a different order
	sort.Slice(notifications, func(i, j int) bool {
		var typeI, typeJ string

		// Safely extract type from notification i
		if attrI, ok := notifications[i].Attributes()["type"]; ok {
			if s, ok := attrI.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				typeI = s.ValueString()
			}
		}

		// Safely extract type from notification j
		if attrJ, ok := notifications[j].Attributes()["type"]; ok {
			if s, ok := attrJ.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
				typeJ = s.ValueString()
			}
		}

		// Treat notifications with invalid or missing types as last in sort order
		if typeI == "" {
			typeI = "\uffff" // Unicode max character - sorts last
		}
		if typeJ == "" {
			typeJ = "\uffff"
		}

		return typeI < typeJ
	})

	// Rundeck expects: { "onsuccess": [ {...}, {...} ], "onfailure": [ {...} ] }
	// Each notification type maps to an ARRAY of notification targets
	result := make(map[string]interface{})

	for _, notifObj := range notifications {
		attrs := notifObj.Attributes()

		var notifType string
		if ntype, ok := attrs["type"].(types.String); ok && !ntype.IsNull() {
			notifType = ntype.ValueString()
			// Convert Terraform underscore format to Rundeck format
			// on_success -> onsuccess, on_failure -> onfailure, etc.
			notifType = strings.ReplaceAll(notifType, "_", "")
		} else {
			continue
		}

		notifMap := make(map[string]interface{})

		// Determine what kind of notification this is (webhook, email, or plugin)

		// Check for webhook (webhook_urls indicates this is a webhook notification)
		if v, ok := attrs["webhook_urls"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var urls []string
			diags.Append(v.ElementsAs(ctx, &urls, false)...)
			if len(urls) > 0 {
				// Webhook fields go at TOP LEVEL of notification object (not nested)
				notifMap["urls"] = strings.Join(urls, ",")

				if format, ok := attrs["format"].(types.String); ok && !format.IsNull() {
					notifMap["format"] = format.ValueString()
				}
				if httpMethod, ok := attrs["http_method"].(types.String); ok && !httpMethod.IsNull() {
					notifMap["httpMethod"] = httpMethod.ValueString()
				}
			}
		}

		// Check for email configuration
		if v, ok := attrs["email"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var emails []types.Object
			diags.Append(v.ElementsAs(ctx, &emails, false)...)
			if len(emails) > 0 {
				emailAttrs := emails[0].Attributes()
				emailMap := make(map[string]interface{})

				if al, ok := emailAttrs["attach_log"].(types.Bool); ok && !al.IsNull() {
					emailMap["attachLog"] = al.ValueBool()
				}
				if subj, ok := emailAttrs["subject"].(types.String); ok && !subj.IsNull() {
					emailMap["subject"] = subj.ValueString()
				}
				if recip, ok := emailAttrs["recipients"].(types.List); ok && !recip.IsNull() {
					var recipients []string
					diags.Append(recip.ElementsAs(ctx, &recipients, false)...)
					// Rundeck expects a comma-separated string, not an array
					emailMap["recipients"] = strings.Join(recipients, ",")
				}

				notifMap["email"] = emailMap
			}
		}

		// Check for plugin configuration
		// Rundeck expects: "plugin": [ {"type": "...", "configuration": {...}}, {...} ]
		if v, ok := attrs["plugin"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var plugins []types.Object
			diags.Append(v.ElementsAs(ctx, &plugins, false)...)
			if len(plugins) > 0 {
				pluginArray := make([]interface{}, 0, len(plugins))

				for _, pluginObj := range plugins {
					pluginAttrs := pluginObj.Attributes()
					pluginMap := make(map[string]interface{})

					if ptype, ok := pluginAttrs["type"].(types.String); ok && !ptype.IsNull() {
						pluginMap["type"] = ptype.ValueString()
					}
					if config, ok := pluginAttrs["config"].(types.Map); ok && !config.IsNull() {
						var configMap map[string]string
						diags.Append(config.ElementsAs(ctx, &configMap, false)...)
						if len(configMap) > 0 {
							pluginMap["configuration"] = configMap
						}
					}

					pluginArray = append(pluginArray, pluginMap)
				}

				notifMap["plugin"] = pluginArray
			}
		}

		// Add this notification to the array for this notification type
		// Initialize array if this is the first notification of this type
		if _, exists := result[notifType]; !exists {
			result[notifType] = []interface{}{}
		}

		// Append to the array for this notification type
		if arr, ok := result[notifType].([]interface{}); ok {
			result[notifType] = append(arr, notifMap)
		}
	}

	return result, diags
}

// Helper to create empty list value
func createEmptyListValue(ctx context.Context, elemType attr.Type) types.List {
	listVal, _ := types.ListValue(elemType, []attr.Value{})
	return listVal
}

// Helper to create object value from map
func createObjectValue(ctx context.Context, attrTypes map[string]attr.Type, values map[string]attr.Value) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(attrTypes, values)
}

// convertExecutionLifecyclePluginsToJSON converts execution_lifecycle_plugin blocks to JSON
// Rundeck expects: {"ExecutionLifecycle": {"pluginType": {config}, "anotherPlugin": {config}}}
// NOT an array with "type" field
func convertExecutionLifecyclePluginsToJSON(ctx context.Context, pluginsList types.List) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if pluginsList.IsNull() || pluginsList.IsUnknown() {
		return nil, diags
	}

	var plugins []types.Object
	diags.Append(pluginsList.ElementsAs(ctx, &plugins, false)...)
	if diags.HasError() {
		return nil, diags
	}

	if len(plugins) == 0 {
		return nil, diags
	}

	// Build ExecutionLifecycle plugin structure as a MAP (not array)
	// The plugin type becomes the key, and the config becomes the value
	executionLifecycle := make(map[string]interface{})
	pluginMap := make(map[string]interface{})

	for _, pluginObj := range plugins {
		attrs := pluginObj.Attributes()

		// Get plugin type (required) - this becomes the MAP KEY
		var pluginType string
		if v, ok := attrs["type"].(types.String); ok && !v.IsNull() {
			pluginType = v.ValueString()
		} else {
			continue // Skip if no type
		}

		// Get config map (optional) - this becomes the MAP VALUE
		configMap := make(map[string]string)
		if v, ok := attrs["config"].(types.Map); ok && !v.IsNull() {
			for key, val := range v.Elements() {
				if strVal, ok := val.(types.String); ok && !strVal.IsNull() {
					configMap[key] = strVal.ValueString()
				}
			}
		}

		// Store as map[pluginType] = configMap
		// Even if config is empty, store empty map (not nil)
		pluginMap[pluginType] = configMap
	}

	executionLifecycle["ExecutionLifecycle"] = pluginMap
	return executionLifecycle, diags
}

// convertProjectSchedulesToJSON converts project_schedule blocks to JSON format
// Returns an array of schedule objects for the "schedules" field at job root level
func convertProjectSchedulesToJSON(ctx context.Context, schedulesList types.List) ([]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if schedulesList.IsNull() || schedulesList.IsUnknown() {
		return nil, diags
	}

	var schedules []types.Object
	diags.Append(schedulesList.ElementsAs(ctx, &schedules, false)...)
	if diags.HasError() {
		return nil, diags
	}

	if len(schedules) == 0 {
		return nil, diags
	}

	// Build project schedule array
	scheduleArray := make([]interface{}, 0, len(schedules))

	for _, scheduleObj := range schedules {
		attrs := scheduleObj.Attributes()
		schedule := make(map[string]interface{})

		// Get schedule name (required)
		if v, ok := attrs["name"].(types.String); ok && !v.IsNull() {
			schedule["name"] = v.ValueString()
		}

		// Get job_options (optional) - maps to "jobParams" in Rundeck API
		if v, ok := attrs["job_options"].(types.String); ok && !v.IsNull() && v.ValueString() != "" {
			schedule["jobParams"] = v.ValueString()
		} else {
			schedule["jobParams"] = nil
		}

		scheduleArray = append(scheduleArray, schedule)
	}

	return scheduleArray, diags
}

// convertCommandPluginsToJSON converts command-level plugins blocks to JSON
func convertCommandPluginsToJSON(ctx context.Context, pluginsList types.List) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if pluginsList.IsNull() || pluginsList.IsUnknown() {
		return nil, diags
	}

	var plugins []types.Object
	diags.Append(pluginsList.ElementsAs(ctx, &plugins, false)...)
	if diags.HasError() {
		return nil, diags
	}

	if len(plugins) == 0 {
		return nil, diags
	}

	// Each plugin object contains nested blocks like log_filter_plugin
	result := make(map[string]interface{})

	for _, pluginObj := range plugins {
		attrs := pluginObj.Attributes()

		// Check for log_filter_plugin blocks
		if v, ok := attrs["log_filter_plugin"].(types.List); ok && !v.IsNull() {
			var logFilters []types.Object
			diags.Append(v.ElementsAs(ctx, &logFilters, false)...)
			if diags.HasError() {
				continue
			}

			logFilterArray := make([]map[string]interface{}, 0, len(logFilters))
			for _, lfObj := range logFilters {
				lfAttrs := lfObj.Attributes()
				logFilter := make(map[string]interface{})

				// Get type (required)
				if typeVal, ok := lfAttrs["type"].(types.String); ok && !typeVal.IsNull() {
					logFilter["type"] = typeVal.ValueString()
				}

				// Get config (optional)
				if configVal, ok := lfAttrs["config"].(types.Map); ok && !configVal.IsNull() {
					configMap := make(map[string]string)
					for key, val := range configVal.Elements() {
						if strVal, ok := val.(types.String); ok && !strVal.IsNull() {
							configMap[key] = strVal.ValueString()
						}
					}
					if len(configMap) > 0 {
						logFilter["config"] = configMap
					}
				}

				logFilterArray = append(logFilterArray, logFilter)
			}

			if len(logFilterArray) > 0 {
				result["LogFilter"] = logFilterArray
			}
		}
	}

	return result, diags
}

// convertOrchestratorToJSON converts orchestrator block to JSON structure
// Rundeck expects: {"type": "maxPercentage", "configuration": {"percent": "80"}}
func convertOrchestratorToJSON(ctx context.Context, orchestratorList types.List) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if orchestratorList.IsNull() || orchestratorList.IsUnknown() {
		return nil, diags
	}

	var orchestrators []types.Object
	diags.Append(orchestratorList.ElementsAs(ctx, &orchestrators, false)...)
	if diags.HasError() {
		return nil, diags
	}

	if len(orchestrators) == 0 {
		return nil, diags
	}

	// Only use the first orchestrator block (job can only have one)
	attrs := orchestrators[0].Attributes()

	result := make(map[string]interface{})
	config := make(map[string]interface{})

	// Get orchestrator type (required)
	if v, ok := attrs["type"].(types.String); ok && !v.IsNull() {
		result["type"] = v.ValueString()
	}

	// Get configuration fields (optional, depends on type)
	if v, ok := attrs["count"].(types.Int64); ok && !v.IsNull() {
		config["count"] = fmt.Sprintf("%d", v.ValueInt64())
	}
	if v, ok := attrs["percent"].(types.Int64); ok && !v.IsNull() {
		config["percent"] = fmt.Sprintf("%d", v.ValueInt64())
	}
	if v, ok := attrs["attribute"].(types.String); ok && !v.IsNull() {
		config["attribute"] = v.ValueString()
	}
	if v, ok := attrs["sort"].(types.String); ok && !v.IsNull() {
		config["sort"] = v.ValueString()
	}

	// Add configuration if any fields were set
	if len(config) > 0 {
		result["configuration"] = config
	}

	return result, diags
}

// convertLogLimitToJobFields converts log_limit block to separate job-level fields
// Returns three values: loglimit, loglimitAction, loglimitStatus (all as strings)
func convertLogLimitToJobFields(ctx context.Context, logLimitList types.List) (loglimit, action, status *string, diags diag.Diagnostics) {
	if logLimitList.IsNull() || logLimitList.IsUnknown() {
		return nil, nil, nil, diags
	}

	var logLimits []types.Object
	diags.Append(logLimitList.ElementsAs(ctx, &logLimits, false)...)
	if diags.HasError() {
		return nil, nil, nil, diags
	}

	if len(logLimits) == 0 {
		return nil, nil, nil, diags
	}

	// Only use the first log_limit block (job can only have one)
	attrs := logLimits[0].Attributes()

	if v, ok := attrs["output"].(types.String); ok && !v.IsNull() {
		val := v.ValueString()
		loglimit = &val
	}
	if v, ok := attrs["action"].(types.String); ok && !v.IsNull() {
		val := v.ValueString()
		action = &val
	}
	if v, ok := attrs["status"].(types.String); ok && !v.IsNull() {
		val := v.ValueString()
		status = &val
	}

	return loglimit, action, status, diags
}

// convertGlobalLogFiltersToJSON converts global_log_filter blocks to pluginConfig.LogFilter
// Returns a map with LogFilter array for sequence-level pluginConfig
func convertGlobalLogFiltersToJSON(ctx context.Context, filtersList types.List) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if filtersList.IsNull() || filtersList.IsUnknown() {
		return nil, diags
	}

	var filters []types.Object
	diags.Append(filtersList.ElementsAs(ctx, &filters, false)...)
	if diags.HasError() {
		return nil, diags
	}

	if len(filters) == 0 {
		return nil, diags
	}

	// Build LogFilter array
	logFilterArray := make([]interface{}, 0, len(filters))

	for _, filterObj := range filters {
		attrs := filterObj.Attributes()
		filter := make(map[string]interface{})

		// Get filter type (required)
		if v, ok := attrs["type"].(types.String); ok && !v.IsNull() {
			filter["type"] = v.ValueString()
		} else {
			continue // Skip if no type
		}

		// Get config map (optional)
		if v, ok := attrs["config"].(types.Map); ok && !v.IsNull() {
			configElements := v.Elements()
			if len(configElements) > 0 {
				configMap := make(map[string]interface{})
				for key, val := range configElements {
					if strVal, ok := val.(types.String); ok && !strVal.IsNull() {
						configMap[key] = strVal.ValueString()
					}
				}
				if len(configMap) > 0 {
					filter["config"] = configMap
				}
			}
		}

		logFilterArray = append(logFilterArray, filter)
	}

	if len(logFilterArray) == 0 {
		return nil, diags
	}

	// Return in the format Rundeck expects for sequence.pluginConfig
	result := map[string]interface{}{
		"LogFilter": logFilterArray,
	}

	return result, diags
}

// normalizeCronSchedule normalizes a cron expression for Rundeck compatibility
// Rundeck ignores day-of-month, so we normalize it to "?" to avoid drift
func normalizeCronSchedule(cronExpr string) string {
	parts := strings.Fields(cronExpr)
	if len(parts) != 7 {
		return cronExpr // Return as-is if invalid format
	}

	// Replace day-of-month (position 3) with "?" since Rundeck doesn't use it
	parts[3] = "?"
	return strings.Join(parts, " ")
}

// convertCronToScheduleObject converts a Quartz cron expression to Rundeck's schedule object format
// Cron format: "seconds minutes hours day-of-month month day-of-week year"
// Example: "* 0/40 * ? * * *" or "0 0 12 ? * * *"
// Note: day-of-month is ignored by Rundeck and will be normalized to "?"
func convertCronToScheduleObject(cronExpr string) (map[string]interface{}, error) {
	parts := strings.Fields(cronExpr)
	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid cron expression: expected 7 fields (seconds minutes hours day-of-month month day-of-week year), got %d", len(parts))
	}

	// Parse: seconds minutes hours day-of-month month day-of-week year
	seconds := parts[0]
	minutes := parts[1]
	hours := parts[2]
	// dayOfMonth := parts[3]  // Not used in Rundeck's format - always normalized to "?"
	month := parts[4]
	dayOfWeek := parts[5]
	year := parts[6]

	schedule := map[string]interface{}{
		"time": map[string]interface{}{
			"seconds": seconds,
			"minute":  minutes,
			"hour":    hours,
		},
		"month": month,
		"weekday": map[string]interface{}{
			"day": dayOfWeek,
		},
		"year": year,
	}

	return schedule, nil
}

// convertScheduleObjectToCron converts Rundeck's schedule object back to Quartz cron string
// Input: {"month":"*","time":{"hour":"*","minute":"0/40","seconds":"*"},"weekday":{"day":"*"},"year":"*"}
// Output: "* 0/40 * ? * * *"
func convertScheduleObjectToCron(scheduleObj interface{}) (string, error) {
	schedMap, ok := scheduleObj.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("schedule is not a map")
	}

	// Extract time fields
	timeMap, ok := schedMap["time"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("schedule.time is not a map")
	}

	seconds, _ := timeMap["seconds"].(string)
	minutes, _ := timeMap["minute"].(string)
	hours, _ := timeMap["hour"].(string)

	// Extract month
	month, _ := schedMap["month"].(string)

	// Extract weekday
	weekdayMap, ok := schedMap["weekday"].(map[string]interface{})
	var dayOfWeek string
	if ok {
		dayOfWeek, _ = weekdayMap["day"].(string)
	}

	// Extract year
	year, _ := schedMap["year"].(string)

	// Build cron string: seconds minutes hours day-of-month month day-of-week year
	// Rundeck uses ? for day-of-month when day-of-week is specified
	cronStr := fmt.Sprintf("%s %s %s ? %s %s %s", seconds, minutes, hours, month, dayOfWeek, year)

	return cronStr, nil
}

// convertOrchestratorFromJSON converts Rundeck orchestrator JSON to Terraform state list
func convertOrchestratorFromJSON(ctx context.Context, orchestratorObj interface{}) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if orchestratorObj == nil {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":      types.StringType,
				"count":     types.Int64Type,
				"percent":   types.Int64Type,
				"attribute": types.StringType,
				"sort":      types.StringType,
			},
		}), diags
	}

	orchMap, ok := orchestratorObj.(map[string]interface{})
	if !ok {
		return types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}), diags
	}

	orchAttrs := make(map[string]attr.Value)

	// Type (required)
	if orchType, ok := orchMap["type"].(string); ok {
		orchAttrs["type"] = types.StringValue(orchType)
	} else {
		orchAttrs["type"] = types.StringNull()
	}

	// Configuration fields (optional, depends on type)
	orchAttrs["count"] = types.Int64Null()
	orchAttrs["percent"] = types.Int64Null()
	orchAttrs["attribute"] = types.StringNull()
	orchAttrs["sort"] = types.StringNull()

	if config, ok := orchMap["configuration"].(map[string]interface{}); ok {
		if count, ok := config["count"].(string); ok {
			if countInt, err := strconv.ParseInt(count, 10, 64); err == nil {
				orchAttrs["count"] = types.Int64Value(countInt)
			}
		}
		if percent, ok := config["percent"].(string); ok {
			if percentInt, err := strconv.ParseInt(percent, 10, 64); err == nil {
				orchAttrs["percent"] = types.Int64Value(percentInt)
			}
		}
		if attribute, ok := config["attribute"].(string); ok {
			orchAttrs["attribute"] = types.StringValue(attribute)
		}
		if sort, ok := config["sort"].(string); ok {
			orchAttrs["sort"] = types.StringValue(sort)
		}
	}

	orchObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":      types.StringType,
			"count":     types.Int64Type,
			"percent":   types.Int64Type,
			"attribute": types.StringType,
			"sort":      types.StringType,
		},
		orchAttrs,
	)

	return types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":      types.StringType,
				"count":     types.Int64Type,
				"percent":   types.Int64Type,
				"attribute": types.StringType,
				"sort":      types.StringType,
			},
		},
		[]attr.Value{orchObj},
	), diags
}

// convertLogLimitFromJSON converts Rundeck log limit fields to Terraform state list
func convertLogLimitFromJSON(ctx context.Context, loglimit, loglimitAction, loglimitStatus *string) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if loglimit == nil || *loglimit == "" {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"output": types.StringType,
				"action": types.StringType,
				"status": types.StringType,
			},
		}), diags
	}

	logLimitAttrs := make(map[string]attr.Value)
	logLimitAttrs["output"] = types.StringValue(*loglimit)

	if loglimitAction != nil {
		logLimitAttrs["action"] = types.StringValue(*loglimitAction)
	} else {
		logLimitAttrs["action"] = types.StringNull()
	}

	if loglimitStatus != nil {
		logLimitAttrs["status"] = types.StringValue(*loglimitStatus)
	} else {
		logLimitAttrs["status"] = types.StringNull()
	}

	logLimitObj := types.ObjectValueMust(
		map[string]attr.Type{
			"output": types.StringType,
			"action": types.StringType,
			"status": types.StringType,
		},
		logLimitAttrs,
	)

	return types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"output": types.StringType,
				"action": types.StringType,
				"status": types.StringType,
			},
		},
		[]attr.Value{logLimitObj},
	), diags
}

// convertNotificationsFromJSON converts Rundeck notification JSON to Terraform state list
func convertNotificationsFromJSON(ctx context.Context, notificationsObj interface{}) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Define the notification object type schema - must match resource schema exactly
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

	if notificationsObj == nil {
		return types.ListNull(notificationObjectType), diags
	}

	notifMap, ok := notificationsObj.(map[string]interface{})
	if !ok {
		return types.ListNull(notificationObjectType), diags
	}

	var notifList []attr.Value

	// Sort notification types alphabetically for deterministic ordering
	// Go map iteration is non-deterministic, but Terraform lists are ordered
	// We use alphabetical sorting since we can't preserve user's input order from the API
	notifTypes := make([]string, 0, len(notifMap))
	for notifType := range notifMap {
		notifTypes = append(notifTypes, notifType)
	}
	sort.Strings(notifTypes)

	// Parse each notification type (onsuccess, onfailure, onstart, onavgduration, onretryablefailure)
	// Rundeck Read/Export format: { "onsuccess": { "email": {...}, "webhook": {...} }, "onfailure": { "email": {...} } }
	// Each type maps to an OBJECT with notification targets as keys
	for _, notifType := range notifTypes {
		notifData := notifMap[notifType]

		// Convert Rundeck format back to Terraform format
		// onsuccess -> on_success, onfailure -> on_failure
		terraformType := notifType
		if notifType == "onsuccess" {
			terraformType = "on_success"
		} else if notifType == "onfailure" {
			terraformType = "on_failure"
		} else if notifType == "onstart" {
			terraformType = "on_start"
		} else if notifType == "onavgduration" {
			terraformType = "on_avg_duration"
		} else if notifType == "onretryablefailure" {
			terraformType = "on_retryable_failure"
		}

		// Each notification type maps to an OBJECT with target types as keys
		// Format: { "email": {...}, "urls": "...", "plugin": [...] }
		targetMap, ok := notifData.(map[string]interface{})
		if !ok {
			continue
		}

		// Determine notification types by checking what fields are present in the target map
		// 1. Webhook: has "urls" key (Rundeck export uses "urls", not a "webhook" wrapper)
		// 2. Email: has "email" key
		// 3. Plugin: has "plugin" key

		// Check for webhook (urls, format, httpMethod at top level in the targetMap)
		if urls, hasUrls := targetMap["urls"].(string); hasUrls {
			notifAttrs := make(map[string]attr.Value)
			notifAttrs["type"] = types.StringValue(terraformType)

			urlList := strings.Split(urls, ",")
			urlValues := make([]attr.Value, len(urlList))
			for i, url := range urlList {
				urlValues[i] = types.StringValue(strings.TrimSpace(url))
			}
			notifAttrs["webhook_urls"] = types.ListValueMust(types.StringType, urlValues)

			// Parse format and http_method for webhook (at top level)
			if format, ok := targetMap["format"].(string); ok {
				notifAttrs["format"] = types.StringValue(format)
			} else {
				notifAttrs["format"] = types.StringNull()
			}

			if httpMethod, ok := targetMap["httpMethod"].(string); ok {
				notifAttrs["http_method"] = types.StringValue(httpMethod)
			} else {
				notifAttrs["http_method"] = types.StringNull()
			}

			// Null out other targets
			notifAttrs["email"] = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"recipients": types.ListType{ElemType: types.StringType},
					"subject":    types.StringType,
					"attach_log": types.BoolType,
				},
			})
			notifAttrs["plugin"] = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":   types.StringType,
					"config": types.MapType{ElemType: types.StringType},
				},
			})

			notifObj := types.ObjectValueMust(
				map[string]attr.Type{
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
				notifAttrs,
			)
			notifList = append(notifList, notifObj)
		}

		// Check for email notification
		if email, ok := targetMap["email"].(map[string]interface{}); ok {
			notifAttrs := make(map[string]attr.Value)
			notifAttrs["type"] = types.StringValue(terraformType)

			// Null out webhook fields
			notifAttrs["webhook_urls"] = types.ListNull(types.StringType)
			notifAttrs["format"] = types.StringNull()
			notifAttrs["http_method"] = types.StringNull()

			emailAttrs := make(map[string]attr.Value)

			if recipients, ok := email["recipients"].(string); ok {
				recipList := strings.Split(recipients, ",")
				recipValues := make([]attr.Value, len(recipList))
				for i, recip := range recipList {
					recipValues[i] = types.StringValue(strings.TrimSpace(recip))
				}
				emailAttrs["recipients"] = types.ListValueMust(types.StringType, recipValues)
			} else {
				emailAttrs["recipients"] = types.ListNull(types.StringType)
			}

			if subject, ok := email["subject"].(string); ok {
				emailAttrs["subject"] = types.StringValue(subject)
			} else {
				emailAttrs["subject"] = types.StringNull()
			}

			// attach_log - only set if present in API response
			// If not present, leave as null to avoid drift when user doesn't specify it
			if attachLog, ok := email["attachLog"].(bool); ok {
				emailAttrs["attach_log"] = types.BoolValue(attachLog)
			} else {
				emailAttrs["attach_log"] = types.BoolNull()
			}

			emailObj := types.ObjectValueMust(
				map[string]attr.Type{
					"recipients": types.ListType{ElemType: types.StringType},
					"subject":    types.StringType,
					"attach_log": types.BoolType,
				},
				emailAttrs,
			)
			notifAttrs["email"] = types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"recipients": types.ListType{ElemType: types.StringType},
						"subject":    types.StringType,
						"attach_log": types.BoolType,
					},
				},
				[]attr.Value{emailObj},
			)

			// Null out plugin
			notifAttrs["plugin"] = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type":   types.StringType,
					"config": types.MapType{ElemType: types.StringType},
				},
			})

			notifObj := types.ObjectValueMust(
				map[string]attr.Type{
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
				notifAttrs,
			)
			notifList = append(notifList, notifObj)
		}

		// Check for plugin notification (plugin is an ARRAY, can contain multiple plugins)
		if pluginArray, ok := targetMap["plugin"].([]interface{}); ok {
			// For Rundeck API, multiple plugins in one notification target are in an array
			// For Terraform, we represent each plugin as a separate entry in the plugin list

			// Create ONE Terraform notification block with ALL plugins from this target
			notifAttrs := make(map[string]attr.Value)
			notifAttrs["type"] = types.StringValue(terraformType)

			// Null out webhook and email fields
			notifAttrs["webhook_urls"] = types.ListNull(types.StringType)
			notifAttrs["format"] = types.StringNull()
			notifAttrs["http_method"] = types.StringNull()
			notifAttrs["email"] = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"recipients": types.ListType{ElemType: types.StringType},
					"subject":    types.StringType,
					"attach_log": types.BoolType,
				},
			})

			// Parse all plugins in the array
			var pluginObjs []attr.Value
			for _, p := range pluginArray {
				pluginMap, ok := p.(map[string]interface{})
				if !ok {
					continue
				}

				pluginAttrs := make(map[string]attr.Value)

				if pluginType, ok := pluginMap["type"].(string); ok {
					pluginAttrs["type"] = types.StringValue(pluginType)
				} else {
					pluginAttrs["type"] = types.StringNull()
				}

				if config, ok := pluginMap["configuration"].(map[string]interface{}); ok {
					configMap := make(map[string]attr.Value)
					for k, v := range config {
						if strVal, ok := v.(string); ok {
							configMap[k] = types.StringValue(strVal)
						}
					}
					pluginAttrs["config"] = types.MapValueMust(types.StringType, configMap)
				} else {
					pluginAttrs["config"] = types.MapNull(types.StringType)
				}

				pluginObj := types.ObjectValueMust(
					map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					},
					pluginAttrs,
				)
				pluginObjs = append(pluginObjs, pluginObj)
			}

			if len(pluginObjs) > 0 {
				notifAttrs["plugin"] = types.ListValueMust(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"type":   types.StringType,
							"config": types.MapType{ElemType: types.StringType},
						},
					},
					pluginObjs,
				)

				notifObj := types.ObjectValueMust(
					map[string]attr.Type{
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
					notifAttrs,
				)
				notifList = append(notifList, notifObj)
			}
		}
	}

	if len(notifList) == 0 {
		return types.ListNull(notificationObjectType), diags
	}

	return types.ListValueMust(notificationObjectType, notifList), diags
}

// convertOptionsFromJSON converts Rundeck options JSON array to Terraform state list
func convertOptionsFromJSON(ctx context.Context, optionsArray []interface{}) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(optionsArray) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}), diags
	}

	var optionList []attr.Value

	for _, optData := range optionsArray {
		optMap, ok := optData.(map[string]interface{})
		if !ok {
			continue
		}

		optAttrs := make(map[string]attr.Value)

		// Required fields
		if name, ok := optMap["name"].(string); ok {
			optAttrs["name"] = types.StringValue(name)
		} else {
			optAttrs["name"] = types.StringNull()
		}

		// Optional string fields
		optAttrs["default_value"] = types.StringNull()
		if defaultValue, ok := optMap["value"].(string); ok {
			optAttrs["default_value"] = types.StringValue(defaultValue)
		}

		optAttrs["description"] = types.StringNull()
		if description, ok := optMap["description"].(string); ok {
			optAttrs["description"] = types.StringValue(description)
		}

		optAttrs["label"] = types.StringNull()
		if label, ok := optMap["label"].(string); ok {
			optAttrs["label"] = types.StringValue(label)
		}

		optAttrs["value_choices_url"] = types.StringNull()
		if valuesUrl, ok := optMap["valuesUrl"].(string); ok {
			optAttrs["value_choices_url"] = types.StringValue(valuesUrl)
		}

		optAttrs["validation_regex"] = types.StringNull()
		if regex, ok := optMap["regex"].(string); ok {
			optAttrs["validation_regex"] = types.StringValue(regex)
		}

		optAttrs["multi_value_delimiter"] = types.StringNull()
		if delimiter, ok := optMap["delimiter"].(string); ok {
			optAttrs["multi_value_delimiter"] = types.StringValue(delimiter)
		}

		// API field is "secure" not "obscureInput"
		if secure, ok := optMap["secure"].(bool); ok {
			optAttrs["obscure_input"] = types.BoolValue(secure)
		} else {
			optAttrs["obscure_input"] = types.BoolNull()
		}

		optAttrs["storage_path"] = types.StringNull()
		if storagePath, ok := optMap["storagePath"].(string); ok {
			optAttrs["storage_path"] = types.StringValue(storagePath)
		}

		optAttrs["type"] = types.StringNull()
		if optType, ok := optMap["type"].(string); ok {
			optAttrs["type"] = types.StringValue(optType)
		}

		// Boolean fields - use correct API field names (match TO JSON converter)
		if required, ok := optMap["required"].(bool); ok {
			optAttrs["required"] = types.BoolValue(required)
		} else {
			optAttrs["required"] = types.BoolNull()
		}

		if multiValued, ok := optMap["multivalued"].(bool); ok {
			optAttrs["allow_multiple_values"] = types.BoolValue(multiValued)
		} else {
			optAttrs["allow_multiple_values"] = types.BoolNull()
		}

		// API field is "enforcedValues" not "enforced"
		// Debug: check what fields are actually in optMap
		if enforced, ok := optMap["enforcedValues"].(bool); ok {
			optAttrs["require_predefined_choice"] = types.BoolValue(enforced)
		} else if enforced, ok := optMap["enforced"].(bool); ok {
			// Try alternate field name
			optAttrs["require_predefined_choice"] = types.BoolValue(enforced)
		} else {
			// If value_choices is specified, enforcedValues defaults to true
			// Check if values or valuesList exists to infer the default
			if _, hasValues := optMap["values"]; hasValues {
				optAttrs["require_predefined_choice"] = types.BoolValue(true)
			} else if _, hasValuesList := optMap["valuesList"]; hasValuesList {
				optAttrs["require_predefined_choice"] = types.BoolValue(true)
			} else {
				optAttrs["require_predefined_choice"] = types.BoolNull()
			}
		}

		if isDate, ok := optMap["isDate"].(bool); ok {
			optAttrs["is_date"] = types.BoolValue(isDate)
		} else {
			optAttrs["is_date"] = types.BoolNull()
		}

		// API field is "valueExposed" not "exposedToScripts"
		if exposed, ok := optMap["valueExposed"].(bool); ok {
			optAttrs["exposed_to_scripts"] = types.BoolValue(exposed)
		} else {
			optAttrs["exposed_to_scripts"] = types.BoolNull()
		}

		if hidden, ok := optMap["hidden"].(bool); ok {
			optAttrs["hidden"] = types.BoolValue(hidden)
		} else {
			optAttrs["hidden"] = types.BoolNull()
		}

		if sortVals, ok := optMap["sortValues"].(bool); ok {
			optAttrs["sort_values"] = types.BoolValue(sortVals)
		} else {
			optAttrs["sort_values"] = types.BoolNull()
		}

		if dateFormat, ok := optMap["dateFormat"].(string); ok {
			optAttrs["date_format"] = types.StringValue(dateFormat)
		} else {
			optAttrs["date_format"] = types.StringNull()
		}

		// value_choices list
		if values, ok := optMap["values"].([]interface{}); ok && len(values) > 0 {
			choiceValues := make([]attr.Value, len(values))
			for i, v := range values {
				if strVal, ok := v.(string); ok {
					choiceValues[i] = types.StringValue(strVal)
				}
			}
			optAttrs["value_choices"] = types.ListValueMust(types.StringType, choiceValues)
		} else {
			optAttrs["value_choices"] = types.ListNull(types.StringType)
		}

		optionObj := types.ObjectValueMust(
			map[string]attr.Type{
				"name":                      types.StringType,
				"default_value":             types.StringType,
				"description":               types.StringType,
				"label":                     types.StringType,
				"value_choices":             types.ListType{ElemType: types.StringType},
				"value_choices_url":         types.StringType,
				"required":                  types.BoolType,
				"allow_multiple_values":     types.BoolType,
				"multi_value_delimiter":     types.StringType,
				"require_predefined_choice": types.BoolType,
				"validation_regex":          types.StringType,
				"obscure_input":             types.BoolType,
				"storage_path":              types.StringType,
				"type":                      types.StringType,
				"is_date":                   types.BoolType,
				"exposed_to_scripts":        types.BoolType,
				"hidden":                    types.BoolType,
				"sort_values":               types.BoolType,
				"date_format":               types.StringType,
			},
			optAttrs,
		)

		optionList = append(optionList, optionObj)
	}

	if len(optionList) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}), diags
	}

	return types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":                      types.StringType,
				"default_value":             types.StringType,
				"description":               types.StringType,
				"label":                     types.StringType,
				"value_choices":             types.ListType{ElemType: types.StringType},
				"value_choices_url":         types.StringType,
				"required":                  types.BoolType,
				"allow_multiple_values":     types.BoolType,
				"multi_value_delimiter":     types.StringType,
				"require_predefined_choice": types.BoolType,
				"validation_regex":          types.StringType,
				"obscure_input":             types.BoolType,
				"storage_path":              types.StringType,
				"type":                      types.StringType,
				"is_date":                   types.BoolType,
				"exposed_to_scripts":        types.BoolType,
				"hidden":                    types.BoolType,
				"sort_values":               types.BoolType,
				"date_format":               types.StringType,
			},
		},
		optionList,
	), diags
}

// convertCommandsFromJSON converts API command array to Terraform state
func convertCommandsFromJSON(ctx context.Context, commands []interface{}) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(commands) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}), diags
	}

	commandList := make([]attr.Value, 0, len(commands))

	for _, cmdInterface := range commands {
		cmd, ok := cmdInterface.(map[string]interface{})
		if !ok {
			continue
		}

		cmdAttrs := map[string]attr.Value{
			"description":                 types.StringNull(),
			"shell_command":               types.StringNull(),
			"inline_script":               types.StringNull(),
			"script_url":                  types.StringNull(),
			"script_file":                 types.StringNull(),
			"script_file_args":            types.StringNull(),
			"file_extension":              types.StringNull(),
			"expand_token_in_script_file": types.BoolNull(),
			"keep_going_on_success":       types.BoolNull(),
			// Note: script_interpreter, plugins, job, etc. are only set if they exist in the API response
			// to avoid needing complex type definitions for all possible nested structures
		}

		// Simple string fields
		if v, ok := cmd["description"].(string); ok && v != "" {
			cmdAttrs["description"] = types.StringValue(v)
		}
		if v, ok := cmd["exec"].(string); ok && v != "" {
			cmdAttrs["shell_command"] = types.StringValue(v)
		}
		if v, ok := cmd["script"].(string); ok && v != "" {
			cmdAttrs["inline_script"] = types.StringValue(v)
		}
		if v, ok := cmd["scripturl"].(string); ok && v != "" {
			cmdAttrs["script_url"] = types.StringValue(v)
		}
		if v, ok := cmd["scriptfile"].(string); ok && v != "" {
			cmdAttrs["script_file"] = types.StringValue(v)
		}
		if v, ok := cmd["args"].(string); ok && v != "" {
			cmdAttrs["script_file_args"] = types.StringValue(v)
		}
		if v, ok := cmd["fileExtension"].(string); ok && v != "" {
			cmdAttrs["file_extension"] = types.StringValue(v)
		}

		// Boolean fields
		if v, ok := cmd["expandTokenInScriptFile"].(bool); ok {
			cmdAttrs["expand_token_in_script_file"] = types.BoolValue(v)
		}
		if v, ok := cmd["keepgoingOnSuccess"].(bool); ok {
			cmdAttrs["keep_going_on_success"] = types.BoolValue(v)
		}

		// Handle scriptInterpreter
		// Rundeck stores this as TWO separate fields at command level:
		// 1. "interpreterArgsQuoted" (boolean)
		// 2. "scriptInterpreter" (string - just the invocation)
		scriptInterp := cmd["scriptInterpreter"]
		interpreterArgsQuoted := cmd["interpreterArgsQuoted"]

		if scriptInterp != nil || interpreterArgsQuoted != nil {
			interpAttrs := map[string]attr.Value{
				"invocation_string": types.StringNull(),
				"args_quoted":       types.BoolNull(),
			}

			// Get the invocation string
			if interpStr, ok := scriptInterp.(string); ok && interpStr != "" {
				interpAttrs["invocation_string"] = types.StringValue(interpStr)
			}

			// Get the args_quoted boolean
			if aq, ok := interpreterArgsQuoted.(bool); ok {
				interpAttrs["args_quoted"] = types.BoolValue(aq)
			}

			// Only create the object if we found at least one value
			if !interpAttrs["invocation_string"].IsNull() || !interpAttrs["args_quoted"].IsNull() {
				interpObj, interpDiags := types.ObjectValue(
					map[string]attr.Type{
						"invocation_string": types.StringType,
						"args_quoted":       types.BoolType,
					},
					interpAttrs,
				)
				diags.Append(interpDiags...)

				cmdAttrs["script_interpreter"] = types.ListValueMust(
					types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"invocation_string": types.StringType,
							"args_quoted":       types.BoolType,
						},
					},
					[]attr.Value{interpObj},
				)
			}
		}

		// Handle error_handler
		if handler, ok := cmd["errorhandler"].(map[string]interface{}); ok {
			handlerAttrs := map[string]attr.Value{
				"description":                 types.StringNull(),
				"shell_command":               types.StringNull(),
				"inline_script":               types.StringNull(),
				"script_url":                  types.StringNull(),
				"script_file":                 types.StringNull(),
				"script_file_args":            types.StringNull(),
				"file_extension":              types.StringNull(),
				"expand_token_in_script_file": types.BoolNull(),
				"keep_going_on_success":       types.BoolNull(),
			}

			// String fields
			if v, ok := handler["description"].(string); ok && v != "" {
				handlerAttrs["description"] = types.StringValue(v)
			}
			if v, ok := handler["exec"].(string); ok && v != "" {
				handlerAttrs["shell_command"] = types.StringValue(v)
			}
			if v, ok := handler["script"].(string); ok && v != "" {
				handlerAttrs["inline_script"] = types.StringValue(v)
			}
			if v, ok := handler["scripturl"].(string); ok && v != "" {
				handlerAttrs["script_url"] = types.StringValue(v)
			}
			if v, ok := handler["scriptfile"].(string); ok && v != "" {
				handlerAttrs["script_file"] = types.StringValue(v)
			}
			if v, ok := handler["args"].(string); ok && v != "" {
				handlerAttrs["script_file_args"] = types.StringValue(v)
			}
			if v, ok := handler["fileExtension"].(string); ok && v != "" {
				handlerAttrs["file_extension"] = types.StringValue(v)
			}

			// Boolean fields
			if v, ok := handler["expandTokenInScriptFile"].(bool); ok {
				handlerAttrs["expand_token_in_script_file"] = types.BoolValue(v)
			}
			if v, ok := handler["keepgoingOnSuccess"].(bool); ok {
				handlerAttrs["keep_going_on_success"] = types.BoolValue(v)
			}

			// Handle job reference in error_handler
			if jobref, ok := handler["jobref"].(map[string]interface{}); ok {
				jobAttrs := map[string]attr.Value{
					"uuid":                 types.StringNull(),
					"name":                 types.StringNull(),
					"group_name":           types.StringNull(),
					"project_name":         types.StringNull(),
					"run_for_each_node":    types.BoolNull(),
					"node_step":            types.BoolNull(),
					"args":                 types.StringNull(),
					"import_options":       types.BoolNull(),
					"child_nodes":          types.BoolNull(),
					"fail_on_disable":      types.BoolNull(),
					"ignore_notifications": types.BoolNull(),
				}

				// String fields
				if uuid, ok := jobref["uuid"].(string); ok && uuid != "" {
					jobAttrs["uuid"] = types.StringValue(uuid)
				}
				if name, ok := jobref["name"].(string); ok && name != "" {
					jobAttrs["name"] = types.StringValue(name)
				}
				if group, ok := jobref["group"].(string); ok && group != "" {
					jobAttrs["group_name"] = types.StringValue(group)
				}
				if project, ok := jobref["project"].(string); ok && project != "" {
					jobAttrs["project_name"] = types.StringValue(project)
				}
				if args, ok := jobref["args"].(string); ok && args != "" {
					jobAttrs["args"] = types.StringValue(args)
				}

				// Boolean fields
				if rfn, ok := jobref["runForEachNode"].(bool); ok {
					jobAttrs["run_for_each_node"] = types.BoolValue(rfn)
				}
				if io, ok := jobref["importOptions"].(bool); ok {
					jobAttrs["import_options"] = types.BoolValue(io)
				}
				if cn, ok := jobref["childNodes"].(bool); ok {
					jobAttrs["child_nodes"] = types.BoolValue(cn)
				}
				if fod, ok := jobref["failOnDisable"].(bool); ok {
					jobAttrs["fail_on_disable"] = types.BoolValue(fod)
				}
				if ign, ok := jobref["ignoreNotifications"].(bool); ok {
					jobAttrs["ignore_notifications"] = types.BoolValue(ign)
				}

				// nodeStep field (API uses string "true"/"false")
				if ns, ok := jobref["nodeStep"].(string); ok && ns != "" {
					jobAttrs["node_step"] = types.BoolValue(ns == "true")
				}

				// Handle node_filters if present
				if nodefilters, ok := jobref["nodefilters"].(map[string]interface{}); ok {
					nfAttrs := map[string]attr.Value{
						"filter":             types.StringNull(),
						"exclude_filter":     types.StringNull(),
						"exclude_precedence": types.BoolNull(),
					}

					if filter, ok := nodefilters["filter"].(string); ok && filter != "" {
						nfAttrs["filter"] = types.StringValue(filter)
					}
					if exclude, ok := nodefilters["excludeFilter"].(string); ok && exclude != "" {
						nfAttrs["exclude_filter"] = types.StringValue(exclude)
					}
					if prec, ok := nodefilters["excludePrecedence"].(bool); ok {
						nfAttrs["exclude_precedence"] = types.BoolValue(prec)
					}

					// Handle dispatch configuration
					if dispatch, ok := nodefilters["dispatch"].(map[string]interface{}); ok {
						dispAttrs := map[string]attr.Value{
							"thread_count":   types.Int64Null(),
							"keep_going":     types.BoolNull(),
							"rank_attribute": types.StringNull(),
							"rank_order":     types.StringNull(),
						}

						// threadcount can be int or float64 from JSON
						if tc, ok := dispatch["threadcount"].(float64); ok {
							dispAttrs["thread_count"] = types.Int64Value(int64(tc))
						} else if tc, ok := dispatch["threadcount"].(int); ok {
							dispAttrs["thread_count"] = types.Int64Value(int64(tc))
						}

						if kg, ok := dispatch["keepgoing"].(bool); ok {
							dispAttrs["keep_going"] = types.BoolValue(kg)
						}
						if ra, ok := dispatch["rankAttribute"].(string); ok && ra != "" {
							dispAttrs["rank_attribute"] = types.StringValue(ra)
						}
						if ro, ok := dispatch["rankOrder"].(string); ok && ro != "" {
							dispAttrs["rank_order"] = types.StringValue(ro)
						}

						dispObj, dispDiags := types.ObjectValue(
							map[string]attr.Type{
								"thread_count":   types.Int64Type,
								"keep_going":     types.BoolType,
								"rank_attribute": types.StringType,
								"rank_order":     types.StringType,
							},
							dispAttrs,
						)
						diags.Append(dispDiags...)
						if !dispObj.IsNull() {
							nfAttrs["dispatch"] = types.ListValueMust(
								types.ObjectType{AttrTypes: map[string]attr.Type{
									"thread_count":   types.Int64Type,
									"keep_going":     types.BoolType,
									"rank_attribute": types.StringType,
									"rank_order":     types.StringType,
								}},
								[]attr.Value{dispObj},
							)
						}
					}

					nfObj, nfDiags := types.ObjectValue(
						map[string]attr.Type{
							"filter":             types.StringType,
							"exclude_filter":     types.StringType,
							"exclude_precedence": types.BoolType,
							"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
								"thread_count":   types.Int64Type,
								"keep_going":     types.BoolType,
								"rank_attribute": types.StringType,
								"rank_order":     types.StringType,
							}}},
						},
						nfAttrs,
					)
					diags.Append(nfDiags...)
					if !nfObj.IsNull() {
						jobAttrs["node_filters"] = types.ListValueMust(
							types.ObjectType{AttrTypes: map[string]attr.Type{
								"filter":             types.StringType,
								"exclude_filter":     types.StringType,
								"exclude_precedence": types.BoolType,
								"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
									"thread_count":   types.Int64Type,
									"keep_going":     types.BoolType,
									"rank_attribute": types.StringType,
									"rank_order":     types.StringType,
								}}},
							}},
							[]attr.Value{nfObj},
						)
					}
				}

				// Only add job block if there are actual job reference fields
				hasJobRef := false
				for _, v := range jobAttrs {
					if !v.IsNull() {
						hasJobRef = true
						break
					}
				}
				if hasJobRef {
					jobObj, jobDiags := types.ObjectValue(
						map[string]attr.Type{
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
							"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
								"filter":             types.StringType,
								"exclude_filter":     types.StringType,
								"exclude_precedence": types.BoolType,
								"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
									"thread_count":   types.Int64Type,
									"keep_going":     types.BoolType,
									"rank_attribute": types.StringType,
									"rank_order":     types.StringType,
								}}},
							}}},
						},
						jobAttrs,
					)
					diags.Append(jobDiags...)
					if !jobObj.IsNull() {
						handlerAttrs["job"] = types.ListValueMust(
							types.ObjectType{AttrTypes: map[string]attr.Type{
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
								"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
									"filter":             types.StringType,
									"exclude_filter":     types.StringType,
									"exclude_precedence": types.BoolType,
									"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
										"thread_count":   types.Int64Type,
										"keep_going":     types.BoolType,
										"rank_attribute": types.StringType,
										"rank_order":     types.StringType,
									}}},
								}}},
							}},
							[]attr.Value{jobObj},
						)
					}
				}
			}

			// Build error_handler object type with job block included
			errorHandlerAttrTypes := map[string]attr.Type{
				"description":                 types.StringType,
				"shell_command":               types.StringType,
				"inline_script":               types.StringType,
				"script_url":                  types.StringType,
				"script_file":                 types.StringType,
				"script_file_args":            types.StringType,
				"file_extension":              types.StringType,
				"expand_token_in_script_file": types.BoolType,
				"keep_going_on_success":       types.BoolType,
				"job": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
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
					"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					}}},
				}}},
			}

			// Initialize job as null if not present
			if _, exists := handlerAttrs["job"]; !exists {
				handlerAttrs["job"] = types.ListNull(
					types.ObjectType{AttrTypes: map[string]attr.Type{
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
						"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"filter":             types.StringType,
							"exclude_filter":     types.StringType,
							"exclude_precedence": types.BoolType,
							"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
								"thread_count":   types.Int64Type,
								"keep_going":     types.BoolType,
								"rank_attribute": types.StringType,
								"rank_order":     types.StringType,
							}}},
						}}},
					}},
				)
			}

			handlerObj, handlerDiags := types.ObjectValue(errorHandlerAttrTypes, handlerAttrs)
			diags.Append(handlerDiags...)
			cmdAttrs["error_handler"] = types.ListValueMust(
				types.ObjectType{AttrTypes: errorHandlerAttrTypes},
				[]attr.Value{handlerObj},
			)
		}

		// Handle step_plugin and node_step_plugin
		// API format: type and configuration at command level, with nodeStep boolean
		if cmdType, ok := cmd["type"].(string); ok && cmdType != "" {
			if configuration, ok := cmd["configuration"].(map[string]interface{}); ok {
				// Check if this is a node step or workflow step
				isNodeStep := false
				if ns, ok := cmd["nodeStep"].(bool); ok {
					isNodeStep = ns
				}

				// Build plugin attributes
				pluginAttrs := map[string]attr.Value{
					"type":   types.StringValue(cmdType),
					"config": types.MapNull(types.StringType),
				}

				// Convert configuration map
				if len(configuration) > 0 {
					configMap := make(map[string]attr.Value)
					for k, v := range configuration {
						if strVal, ok := v.(string); ok {
							configMap[k] = types.StringValue(strVal)
						}
					}
					if len(configMap) > 0 {
						pluginAttrs["config"] = types.MapValueMust(types.StringType, configMap)
					}
				}

				pluginObj, pluginDiags := types.ObjectValue(
					map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					},
					pluginAttrs,
				)
				diags.Append(pluginDiags...)

				// Set as step_plugin or node_step_plugin based on nodeStep flag
				if isNodeStep {
					cmdAttrs["node_step_plugin"] = types.ListValueMust(
						types.ObjectType{AttrTypes: map[string]attr.Type{
							"type":   types.StringType,
							"config": types.MapType{ElemType: types.StringType},
						}},
						[]attr.Value{pluginObj},
					)
				} else {
					cmdAttrs["step_plugin"] = types.ListValueMust(
						types.ObjectType{AttrTypes: map[string]attr.Type{
							"type":   types.StringType,
							"config": types.MapType{ElemType: types.StringType},
						}},
						[]attr.Value{pluginObj},
					)
				}
			}
		}

		// Handle job references
		if jobref, ok := cmd["jobref"].(map[string]interface{}); ok {
			jobAttrs := map[string]attr.Value{
				"uuid":                 types.StringNull(),
				"name":                 types.StringNull(),
				"group_name":           types.StringNull(),
				"project_name":         types.StringNull(),
				"run_for_each_node":    types.BoolNull(),
				"node_step":            types.BoolNull(),
				"args":                 types.StringNull(),
				"import_options":       types.BoolNull(),
				"child_nodes":          types.BoolNull(),
				"fail_on_disable":      types.BoolNull(),
				"ignore_notifications": types.BoolNull(),
			}

			// String fields
			if uuid, ok := jobref["uuid"].(string); ok && uuid != "" {
				jobAttrs["uuid"] = types.StringValue(uuid)
			}
			if name, ok := jobref["name"].(string); ok && name != "" {
				jobAttrs["name"] = types.StringValue(name)
			}
			if group, ok := jobref["group"].(string); ok && group != "" {
				jobAttrs["group_name"] = types.StringValue(group)
			}
			if project, ok := jobref["project"].(string); ok && project != "" {
				jobAttrs["project_name"] = types.StringValue(project)
			}
			if args, ok := jobref["args"].(string); ok && args != "" {
				jobAttrs["args"] = types.StringValue(args)
			}

			// Boolean fields
			if rfn, ok := jobref["runForEachNode"].(bool); ok {
				jobAttrs["run_for_each_node"] = types.BoolValue(rfn)
			}
			if io, ok := jobref["importOptions"].(bool); ok {
				jobAttrs["import_options"] = types.BoolValue(io)
			}
			if cn, ok := jobref["childNodes"].(bool); ok {
				jobAttrs["child_nodes"] = types.BoolValue(cn)
			}
			if fod, ok := jobref["failOnDisable"].(bool); ok {
				jobAttrs["fail_on_disable"] = types.BoolValue(fod)
			}
			if ign, ok := jobref["ignoreNotifications"].(bool); ok {
				jobAttrs["ignore_notifications"] = types.BoolValue(ign)
			}

			// nodeStep field (API can use string "true"/"false" or boolean)
			if ns, ok := jobref["nodeStep"].(string); ok && ns != "" {
				jobAttrs["node_step"] = types.BoolValue(ns == "true")
			} else if ns, ok := jobref["nodeStep"].(bool); ok {
				jobAttrs["node_step"] = types.BoolValue(ns)
			}

			// Handle node_filters if present
			if nodefilters, ok := jobref["nodefilters"].(map[string]interface{}); ok {
				nfAttrs := map[string]attr.Value{
					"filter":             types.StringNull(),
					"exclude_filter":     types.StringNull(),
					"exclude_precedence": types.BoolNull(),
				}

				if filter, ok := nodefilters["filter"].(string); ok && filter != "" {
					nfAttrs["filter"] = types.StringValue(filter)
				}
				if exclude, ok := nodefilters["excludeFilter"].(string); ok && exclude != "" {
					nfAttrs["exclude_filter"] = types.StringValue(exclude)
				}
				if prec, ok := nodefilters["excludePrecedence"].(bool); ok {
					nfAttrs["exclude_precedence"] = types.BoolValue(prec)
				}

				// Handle dispatch configuration
				if dispatch, ok := nodefilters["dispatch"].(map[string]interface{}); ok {
					dispAttrs := map[string]attr.Value{
						"thread_count":   types.Int64Null(),
						"keep_going":     types.BoolNull(),
						"rank_attribute": types.StringNull(),
						"rank_order":     types.StringNull(),
					}

					// threadcount can be int or float64 from JSON
					if tc, ok := dispatch["threadcount"].(float64); ok {
						dispAttrs["thread_count"] = types.Int64Value(int64(tc))
					} else if tc, ok := dispatch["threadcount"].(int); ok {
						dispAttrs["thread_count"] = types.Int64Value(int64(tc))
					}

					if kg, ok := dispatch["keepgoing"].(bool); ok {
						dispAttrs["keep_going"] = types.BoolValue(kg)
					}
					if ra, ok := dispatch["rankAttribute"].(string); ok && ra != "" {
						dispAttrs["rank_attribute"] = types.StringValue(ra)
					}
					if ro, ok := dispatch["rankOrder"].(string); ok && ro != "" {
						dispAttrs["rank_order"] = types.StringValue(ro)
					}

					dispObj, dispDiags := types.ObjectValue(
						map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						},
						dispAttrs,
					)
					diags.Append(dispDiags...)

					nfAttrs["dispatch"] = types.ListValueMust(
						types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}},
						[]attr.Value{dispObj},
					)
				} else {
					// No dispatch config
					nfAttrs["dispatch"] = types.ListNull(
						types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}},
					)
				}

				nfObj, nfDiags := types.ObjectValue(
					map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					},
					nfAttrs,
				)
				diags.Append(nfDiags...)

				jobAttrs["node_filters"] = types.ListValueMust(
					types.ObjectType{AttrTypes: map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					}},
					[]attr.Value{nfObj},
				)
			} else {
				// No node_filters
				jobAttrs["node_filters"] = types.ListNull(
					types.ObjectType{AttrTypes: map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					}},
				)
			}

			jobObj, jobDiags := types.ObjectValue(
				map[string]attr.Type{
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
					"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					}}},
				},
				jobAttrs,
			)
			diags.Append(jobDiags...)

			cmdAttrs["job"] = types.ListValueMust(
				types.ObjectType{AttrTypes: map[string]attr.Type{
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
					"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					}}},
				}},
				[]attr.Value{jobObj},
			)
		}

		// Handle command-level plugins (log filters)
		if plugins, ok := cmd["plugins"].(map[string]interface{}); ok {
			if logFilters, ok := plugins["LogFilter"].([]interface{}); ok && len(logFilters) > 0 {
				logFilterValues := make([]attr.Value, 0, len(logFilters))

				for _, lf := range logFilters {
					if logFilter, ok := lf.(map[string]interface{}); ok {
						lfAttrs := map[string]attr.Value{
							"type":   types.StringNull(),
							"config": types.MapNull(types.StringType),
						}

						// Get type
						if lfType, ok := logFilter["type"].(string); ok && lfType != "" {
							lfAttrs["type"] = types.StringValue(lfType)
						}

						// Get config
						if config, ok := logFilter["config"].(map[string]interface{}); ok && len(config) > 0 {
							configMap := make(map[string]attr.Value)
							for k, v := range config {
								if strVal, ok := v.(string); ok {
									configMap[k] = types.StringValue(strVal)
								}
							}
							if len(configMap) > 0 {
								lfAttrs["config"] = types.MapValueMust(types.StringType, configMap)
							}
						}

						lfObj, lfDiags := types.ObjectValue(
							map[string]attr.Type{
								"type":   types.StringType,
								"config": types.MapType{ElemType: types.StringType},
							},
							lfAttrs,
						)
						diags.Append(lfDiags...)
						logFilterValues = append(logFilterValues, lfObj)
					}
				}

				if len(logFilterValues) > 0 {
					// Create the plugins block with log_filter_plugin list
					pluginsAttrs := map[string]attr.Value{
						"log_filter_plugin": types.ListValueMust(
							types.ObjectType{AttrTypes: map[string]attr.Type{
								"type":   types.StringType,
								"config": types.MapType{ElemType: types.StringType},
							}},
							logFilterValues,
						),
					}

					pluginsObj, pluginsDiags := types.ObjectValue(
						map[string]attr.Type{
							"log_filter_plugin": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
								"type":   types.StringType,
								"config": types.MapType{ElemType: types.StringType},
							}}},
						},
						pluginsAttrs,
					)
					diags.Append(pluginsDiags...)

					cmdAttrs["plugins"] = types.ListValueMust(
						types.ObjectType{AttrTypes: map[string]attr.Type{
							"log_filter_plugin": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
								"type":   types.StringType,
								"config": types.MapType{ElemType: types.StringType},
							}}},
						}},
						[]attr.Value{pluginsObj},
					)
				}
			}
		}

		// Initialize any missing nested block fields as null to match schema
		// We need to use the exact types from commandObjectType to avoid type mismatch errors
		if _, exists := cmdAttrs["script_interpreter"]; !exists {
			cmdAttrs["script_interpreter"] = types.ListNull(
				types.ObjectType{AttrTypes: map[string]attr.Type{
					"args_quoted":       types.BoolType,
					"invocation_string": types.StringType,
				}},
			)
		}
		if _, exists := cmdAttrs["plugins"]; !exists {
			cmdAttrs["plugins"] = types.ListNull(
				types.ObjectType{AttrTypes: map[string]attr.Type{
					"log_filter_plugin": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					}}},
				}},
			)
		}
		if _, exists := cmdAttrs["job"]; !exists {
			cmdAttrs["job"] = types.ListNull(
				types.ObjectType{AttrTypes: map[string]attr.Type{
					"name":                 types.StringType,
					"group_name":           types.StringType,
					"project_name":         types.StringType,
					"uuid":                 types.StringType,
					"args":                 types.StringType,
					"run_for_each_node":    types.BoolType,
					"node_step":            types.BoolType,
					"child_nodes":          types.BoolType,
					"import_options":       types.BoolType,
					"fail_on_disable":      types.BoolType,
					"ignore_notifications": types.BoolType,
					"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"filter":             types.StringType,
						"exclude_filter":     types.StringType,
						"exclude_precedence": types.BoolType,
						"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"thread_count":   types.Int64Type,
							"keep_going":     types.BoolType,
							"rank_attribute": types.StringType,
							"rank_order":     types.StringType,
						}}},
					}}},
				}},
			)
		}
		if _, exists := cmdAttrs["step_plugin"]; !exists {
			cmdAttrs["step_plugin"] = types.ListNull(
				types.ObjectType{AttrTypes: map[string]attr.Type{
					"type":   types.StringType,
					"config": types.MapType{ElemType: types.StringType},
				}},
			)
		}
		if _, exists := cmdAttrs["node_step_plugin"]; !exists {
			cmdAttrs["node_step_plugin"] = types.ListNull(
				types.ObjectType{AttrTypes: map[string]attr.Type{
					"type":   types.StringType,
					"config": types.MapType{ElemType: types.StringType},
				}},
			)
		}
		if _, exists := cmdAttrs["error_handler"]; !exists {
			cmdAttrs["error_handler"] = types.ListNull(
				types.ObjectType{AttrTypes: map[string]attr.Type{
					"description":                 types.StringType,
					"shell_command":               types.StringType,
					"inline_script":               types.StringType,
					"script_url":                  types.StringType,
					"script_file":                 types.StringType,
					"script_file_args":            types.StringType,
					"file_extension":              types.StringType,
					"expand_token_in_script_file": types.BoolType,
					"keep_going_on_success":       types.BoolType,
					"script_interpreter": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"args_quoted":       types.BoolType,
						"invocation_string": types.StringType,
					}}},
					"job": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
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
						"node_filters": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
							"filter":             types.StringType,
							"exclude_filter":     types.StringType,
							"exclude_precedence": types.BoolType,
							"dispatch": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
								"thread_count":   types.Int64Type,
								"keep_going":     types.BoolType,
								"rank_attribute": types.StringType,
								"rank_order":     types.StringType,
							}}},
						}}},
					}}},
					"step_plugin": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					}}},
					"node_step_plugin": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					}}},
				}},
			)
		}

		// Use the commandObjectType defined in resource_job_command_schema.go
		cmdObj, cmdDiags := types.ObjectValue(commandObjectType.AttrTypes, cmdAttrs)
		diags.Append(cmdDiags...)

		commandList = append(commandList, cmdObj)
	}

	if len(commandList) == 0 {
		return types.ListNull(commandObjectType), diags
	}

	return types.ListValueMust(commandObjectType, commandList), diags
}
