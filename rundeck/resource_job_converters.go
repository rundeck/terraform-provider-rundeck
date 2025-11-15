package rundeck

import (
	"context"
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

		// Webhook fields
		if v, ok := attrs["format"].(types.String); ok && !v.IsNull() {
			notifMap["format"] = v.ValueString()
		}
		if v, ok := attrs["http_method"].(types.String); ok && !v.IsNull() {
			notifMap["httpMethod"] = v.ValueString()
		}
		if v, ok := attrs["webhook_urls"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			var urls []string
			diags.Append(v.ElementsAs(ctx, &urls, false)...)
			if len(urls) > 0 {
				notifMap["urls"] = urls
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
