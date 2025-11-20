package rundeck

import (
	"context"
	"fmt"
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
			cmdMap["scriptargs"] = v.ValueString()
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
		if v, ok := attrs["script_interpreter"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var interpreters []types.Object
			diags.Append(v.ElementsAs(ctx, &interpreters, false)...)
			if len(interpreters) > 0 {
				interpAttrs := interpreters[0].Attributes()
				interpMap := make(map[string]interface{})
				if inv, ok := interpAttrs["invocation_string"].(types.String); ok && !inv.IsNull() {
					interpMap["invocationString"] = inv.ValueString()
				}
				if aq, ok := interpAttrs["args_quoted"].(types.Bool); ok && !aq.IsNull() {
					interpMap["argsQuoted"] = aq.ValueBool()
				}
				cmdMap["scriptInterpreter"] = interpMap
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
						jobMap["nodefilters"] = nfMap
					}
				}

				cmdMap["jobref"] = jobMap
			}
		}

		// Handle step_plugin
		if v, ok := attrs["step_plugin"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var plugins []types.Object
			diags.Append(v.ElementsAs(ctx, &plugins, false)...)
			if len(plugins) > 0 {
				pluginAttrs := plugins[0].Attributes()
				pluginMap := make(map[string]interface{})

				if ptype, ok := pluginAttrs["type"].(types.String); ok && !ptype.IsNull() {
					pluginMap["type"] = ptype.ValueString()
				}
				if config, ok := pluginAttrs["config"].(types.Map); ok && !config.IsNull() {
					var configMap map[string]string
					diags.Append(config.ElementsAs(ctx, &configMap, false)...)
					pluginMap["config"] = configMap
				}

				cmdMap["plugin"] = pluginMap
			}
		}

		// Handle node_step_plugin
		if v, ok := attrs["node_step_plugin"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var plugins []types.Object
			diags.Append(v.ElementsAs(ctx, &plugins, false)...)
			if len(plugins) > 0 {
				pluginAttrs := plugins[0].Attributes()
				pluginMap := make(map[string]interface{})

				if ptype, ok := pluginAttrs["type"].(types.String); ok && !ptype.IsNull() {
					pluginMap["type"] = ptype.ValueString()
				}
				if config, ok := pluginAttrs["config"].(types.Map); ok && !config.IsNull() {
					var configMap map[string]string
					diags.Append(config.ElementsAs(ctx, &configMap, false)...)
					pluginMap["config"] = configMap
				}

				cmdMap["nodeStep"] = pluginMap
			}
		}

		// Handle error_handler (simplified - doesn't recurse infinitely)
		if v, ok := attrs["error_handler"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var handlers []types.Object
			diags.Append(v.ElementsAs(ctx, &handlers, false)...)
			if len(handlers) > 0 {
				handlerAttrs := handlers[0].Attributes()
				handlerMap := make(map[string]interface{})

				if desc, ok := handlerAttrs["description"].(types.String); ok && !desc.IsNull() {
					handlerMap["description"] = desc.ValueString()
				}
				if sc, ok := handlerAttrs["shell_command"].(types.String); ok && !sc.IsNull() {
					handlerMap["exec"] = sc.ValueString()
				}
				if script, ok := handlerAttrs["inline_script"].(types.String); ok && !script.IsNull() {
					handlerMap["script"] = script.ValueString()
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

		// Format and HTTP method are top-level webhook notification properties
		if v, ok := attrs["format"].(types.String); ok && !v.IsNull() {
			notifMap["format"] = v.ValueString()
		}
		if v, ok := attrs["http_method"].(types.String); ok && !v.IsNull() {
			notifMap["httpMethod"] = v.ValueString()
		}

		// Webhook URLs need to be nested in a "webhook" object
		if v, ok := attrs["webhook_urls"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var urls []string
			diags.Append(v.ElementsAs(ctx, &urls, false)...)
			if len(urls) > 0 {
				// Rundeck expects webhook URLs as a comma-separated string in a webhook object
				webhookMap := make(map[string]interface{})
				webhookMap["urls"] = strings.Join(urls, ",")
				notifMap["webhook"] = webhookMap
			}
		}

		// Email configuration
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

		// Plugin configuration
		if v, ok := attrs["plugin"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var plugins []types.Object
			diags.Append(v.ElementsAs(ctx, &plugins, false)...)
			if len(plugins) > 0 {
				pluginAttrs := plugins[0].Attributes()
				pluginMap := make(map[string]interface{})

				if ptype, ok := pluginAttrs["type"].(types.String); ok && !ptype.IsNull() {
					pluginMap["type"] = ptype.ValueString()
				}
				if config, ok := pluginAttrs["config"].(types.Map); ok && !config.IsNull() {
					var configMap map[string]string
					diags.Append(config.ElementsAs(ctx, &configMap, false)...)
					pluginMap["config"] = configMap
				}

				notifMap["plugin"] = pluginMap
			}
		}

		result[notifType] = notifMap
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
