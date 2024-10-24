package rundeck

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckJob() *schema.Resource {
	return &schema.Resource{
		Create: CreateJob,
		Update: UpdateJob,
		Delete: DeleteJob,
		Exists: JobExists,
		Read:   ReadJob,
		Importer: &schema.ResourceImporter{
			State: resourceJobImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Required: true,
			},

			"execution_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"log_level": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "INFO",
			},

			"allow_concurrent_executions": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"retry": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"retry_delay": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"max_thread_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"continue_on_error": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"continue_next_node_on_error": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"rank_order": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "ascending",
			},

			"rank_attribute": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"success_on_empty_node_filter": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"preserve_options_order": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"command_ordering_strategy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "node-first",
			},

			"node_filter_query": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"node_filter_exclude_query": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"node_filter_exclude_precedence": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"timeout": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"orchestrator": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Option of `subset`, `rankTiered`, `maxPercentage`, `orchestrator-highest-lowest-attribute`",
						},
						"count": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Value for the subset orchestrator",
						},
						"percent": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Value for the maxPercentage orchestrator",
						},
						"attribute": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The Node Attribute that shoud be used to rank nodes in High/Low Orchestrator.",
						},
						"sort": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Option of `highest` or `lowest` for High/Low Orchestrator",
						},
					},
				},
			},

			"schedule": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"schedule_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"nodes_selected_by_default": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"time_zone": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"notification": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Option of `on_success`, `on_failure`, `on_start`",
						},
						"email": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"attach_log": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"recipients": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"subject": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"webhook_urls": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"plugin": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceRundeckJobPluginResource(),
						},
					},
				},
			},

			"option": {
				// This is a list because order is important when preserve_options_order is
				// set. When it's not set the order is unimportant but preserved by Rundeck/
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"label": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"default_value": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"value_choices": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"value_choices_url": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"require_predefined_choice": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"validation_regex": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"description": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"required": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"allow_multiple_values": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"multi_value_delimiter": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"obscure_input": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"exposed_to_scripts": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"storage_path": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"hidden": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"is_date": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"date_format": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"global_log_filter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobFilter(),
			},

			"command": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     resourceRundeckJobCommand(),
			},
		},
	}
}

// Attention - Changes made to this function should be repeated in resourceRundeckJobCommandErrorHandler below!
func resourceRundeckJobCommand() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"shell_command": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"inline_script": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_url": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file_args": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_interpreter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandScriptInterpreter(),
			},

			"job": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandJob(),
			},
			"step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},
			"plugins": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginsResource(),
			},
			"node_step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},
			"keep_going_on_success": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"error_handler": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandErrorHandler(),
			},
		},
	}
}

// Terraform schemas do not support recursion. The Error Handler is a command within a command, but we're breaking it
// out and repeating it verbatim except for an inner errorHandler field.
func resourceRundeckJobCommandErrorHandler() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"shell_command": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"inline_script": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_url": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_file_args": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"script_interpreter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandScriptInterpreter(),
			},

			"job": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobCommandJob(),
			},
			"step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},

			"plugins": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginsResource(),
			},

			"node_step_plugin": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceRundeckJobPluginResource(),
			},

			"keep_going_on_success": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobCommandScriptInterpreter() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"invocation_string": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"args_quoted": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobCommandJob() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"group_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"run_for_each_node": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"args": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"child_nodes": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"fail_on_disable": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"ignore_notifications": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"import_options": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"node_filters": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"exclude_precedence": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"filter": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"exclude_filter": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceRundeckJobPluginResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceRundeckJobPluginsResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"log_filter_plugin": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     resourceRundeckJobPluginResource(),
			},
		},
	}
}

func resourceRundeckJobFilter() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"config": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func CreateJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)

	job, err := jobFromResourceData(d)
	if err != nil {
		return err
	}

	jobSummary, err := importJob(client, job, "create")
	if err != nil {
		return err
	}

	d.SetId(jobSummary.ID)

	return ReadJob(d, meta)
}

func UpdateJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)

	job, err := jobFromResourceData(d)
	if err != nil {
		return err
	}

	jobSummary, err := importJob(client, job, "update")
	if err != nil {
		return err
	}

	d.SetId(jobSummary.ID)

	return ReadJob(d, meta)
}

func DeleteJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	_, err := client.JobDelete(ctx, d.Id())
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func JobExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	resp, err := client.JobGet(ctx, d.Id(), "")
	if err != nil {
		var notFound *NotFoundError
		if err == notFound {
			return false, nil
		}
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("error checking if job exists: (%v)", err)
}

func ReadJob(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)

	job, err := GetJob(client, d.Id())
	if err != nil {
		return err
	}

	return jobToResourceData(job, d)
}

func jobFromResourceData(d *schema.ResourceData) (*JobDetail, error) {
	job := &JobDetail{
		ID:                        d.Id(),
		Name:                      d.Get("name").(string),
		GroupName:                 d.Get("group_name").(string),
		ProjectName:               d.Get("project_name").(string),
		Description:               d.Get("description").(string),
		ExecutionEnabled:          d.Get("execution_enabled").(bool),
		Timeout:                   d.Get("timeout").(string),
		ScheduleEnabled:           d.Get("schedule_enabled").(bool),
		NodesSelectedByDefault:    d.Get("nodes_selected_by_default").(bool),
		TimeZone:                  d.Get("time_zone").(string),
		LogLevel:                  d.Get("log_level").(string),
		AllowConcurrentExecutions: d.Get("allow_concurrent_executions").(bool),
		Retry: &Retry{
			Value: d.Get("retry").(string),
			Delay: d.Get("retry_delay").(string),
		},
		Dispatch: &JobDispatch{
			MaxThreadCount:          d.Get("max_thread_count").(int),
			ContinueNextNodeOnError: d.Get("continue_next_node_on_error").(bool),
			RankAttribute:           d.Get("rank_attribute").(string),
			RankOrder:               d.Get("rank_order").(string),
		},
	}

	successOnEmpty := d.Get("success_on_empty_node_filter")
	if successOnEmpty != nil {
		job.Dispatch.SuccessOnEmptyNodeFilter = successOnEmpty.(bool)
	}

	orchList := d.Get("orchestrator").([]interface{})
	if len(orchList) > 1 {
		return nil, fmt.Errorf("rundeck command may have no more than one orchestrator")
	}
	for _, orch := range orchList {
		orchMap := orch.(map[string]interface{})
		job.Orchestrator = &JobOrchestrator{
			Type:   orchMap["type"].(string),
			Config: JobOrchestratorConfig{},
		}
		orchType := orchMap["type"].(string)
		if orchType == "orchestrator-highest-lowest-attribute" {
			orchAttr := orchMap["attribute"]
			if orchAttr != nil {
				job.Orchestrator.Config.Attribute = orchAttr.(string)
			} else {
				return nil, fmt.Errorf("high Low Orchestrator must include an attribute to sort against")
			}
			orchSort := orchMap["sort"]
			if orchSort != nil {
				job.Orchestrator.Config.Sort = orchSort.(string)
			} else {
				return nil, fmt.Errorf("high low orchestrator must include sort direction of `high` or `low`")
			}
		}
		if orchType == "subset" {
			orchCount := orchMap["count"]
			if orchCount != nil {
				job.Orchestrator.Config.Count = orchCount.(int)
			} else {
				return nil, fmt.Errorf("subset Orchestrator requires count setting")
			}
		}
		if orchType == "maxPercentage" {
			orchPct := orchMap["percent"]
			if orchPct != nil {
				job.Orchestrator.Config.Percent = orchPct.(int)
			} else {
				return nil, fmt.Errorf("max Percentage Orchestrator requires a percent integer configuration")
			}
		}
	}

	sequence := &JobCommandSequence{
		ContinueOnError:  d.Get("continue_on_error").(bool),
		OrderingStrategy: d.Get("command_ordering_strategy").(string),
		Commands:         []JobCommand{},
	}

	logFilterConfigs := d.Get("global_log_filter").([]interface{})
	if len(logFilterConfigs) > 0 {
		globalLogFilters := &[]JobLogFilter{}
		for _, logFilterI := range logFilterConfigs {
			logFilterMap := logFilterI.(map[string]interface{})
			configI := logFilterMap["config"].(map[string]interface{})
			config := &JobLogFilterConfig{}
			for key, value := range configI {
				(*config)[key] = value.(string)
			}
			logFilter := &JobLogFilter{
				Type:   logFilterMap["type"].(string),
				Config: config,
			}

			*globalLogFilters =
				append(*globalLogFilters, *logFilter)
		}
		sequence.GlobalLogFilters = globalLogFilters
	}
	commandConfigs := d.Get("command").([]interface{})
	for _, commandI := range commandConfigs {
		command, err := commandFromResourceData(commandI)
		if err != nil {
			return nil, err
		}
		sequence.Commands = append(sequence.Commands, *command)
	}
	job.CommandSequence = sequence

	optionConfigsI := d.Get("option").([]interface{})
	if len(optionConfigsI) > 0 {
		optionsConfig := &JobOptions{
			PreserveOrder: d.Get("preserve_options_order").(bool),
			Options:       []JobOption{},
		}
		for _, optionI := range optionConfigsI {
			optionMap := optionI.(map[string]interface{})
			option := JobOption{
				Name:                    optionMap["name"].(string),
				Label:                   optionMap["label"].(string),
				DefaultValue:            optionMap["default_value"].(string),
				ValueChoices:            JobValueChoices([]string{}),
				ValueChoicesURL:         optionMap["value_choices_url"].(string),
				RequirePredefinedChoice: optionMap["require_predefined_choice"].(bool),
				ValidationRegex:         optionMap["validation_regex"].(string),
				Description:             optionMap["description"].(string),
				IsRequired:              optionMap["required"].(bool),
				AllowsMultipleValues:    optionMap["allow_multiple_values"].(bool),
				MultiValueDelimiter:     optionMap["multi_value_delimiter"].(string),
				ObscureInput:            optionMap["obscure_input"].(bool),
				ValueIsExposedToScripts: optionMap["exposed_to_scripts"].(bool),
				StoragePath:             optionMap["storage_path"].(string),
				Hidden:                  optionMap["hidden"].(bool),
				IsDate:                  optionMap["is_date"].(bool),
				DateFormat:              optionMap["date_format"].(string),
			}
			if option.StoragePath != "" && !option.ObscureInput {
				return nil, fmt.Errorf("argument \"obscure_input\" must be set to `true` when \"storage_path\" is not empty")
			}
			if option.ValueIsExposedToScripts && !option.ObscureInput {
				return nil, fmt.Errorf("argument \"obscure_input\" must be set to `true` when \"exposed_to_scripts\" is set to true")
			}
			if option.DateFormat != "" && option.IsDate == false {
				return nil, fmt.Errorf("Argument \"is_date\" must be set to `true` when \"date_format\" is not empty.")
			}

			for _, iv := range optionMap["value_choices"].([]interface{}) {
				if iv == nil {
					return nil, fmt.Errorf("argument \"value_choices\" can not have empty values; try \"required\"")
				}
				option.ValueChoices = append(option.ValueChoices, iv.(string))
			}

			optionsConfig.Options = append(optionsConfig.Options, option)
		}
		job.OptionsConfig = optionsConfig
	}

	job.NodeFilter = &JobNodeFilter{
		ExcludePrecedence: d.Get("node_filter_exclude_precedence").(bool),
	}
	if nodeFilterQuery := d.Get("node_filter_query").(string); nodeFilterQuery != "" {
		job.NodeFilter.Query = nodeFilterQuery
	}
	if nodeFilterExcludeQuery := d.Get("node_filter_exclude_query").(string); nodeFilterExcludeQuery != "" {
		job.NodeFilter.ExcludeQuery = nodeFilterExcludeQuery
	}

	if err := JobScheduleFromResourceData(d, job); err != nil {
		return nil, err
	}

	notificationsConfigI := d.Get("notification").([]interface{})
	if len(notificationsConfigI) > 0 {
		if len(notificationsConfigI) <= 3 {
			jobNotification := JobNotification{}
			// test if unique
			for _, notificationI := range notificationsConfigI {
				notification := Notification{}
				notificationMap := notificationI.(map[string]interface{})
				jobType := notificationMap["type"].(string)

				// Get email notification data
				notificationEmailsI := notificationMap["email"].([]interface{})
				if len(notificationEmailsI) > 0 {
					notificationEmailI := notificationEmailsI[0].(map[string]interface{})
					email := EmailNotification{
						AttachLog:  notificationEmailI["attach_log"].(bool),
						Recipients: NotificationEmails([]string{}),
						Subject:    notificationEmailI["subject"].(string),
					}
					for _, iv := range notificationEmailI["recipients"].([]interface{}) {
						email.Recipients = append(email.Recipients, iv.(string))
					}
					notification.Email = &email
				}

				// Webhook notification
				webHookUrls := notificationMap["webhook_urls"].([]interface{})
				if len(webHookUrls) > 0 {
					webHook := &WebHookNotification{
						Urls: NotificationUrls([]string{}),
					}
					for _, iv := range webHookUrls {
						webHook.Urls = append(webHook.Urls, iv.(string))
					}
					notification.WebHook = webHook
				}

				// plugin Notification
				notificationPluginsI := notificationMap["plugin"].([]interface{})
				if len(notificationPluginsI) > 1 {
					return nil, fmt.Errorf("rundeck command may have no more than one notification plugin")
				}
				if len(notificationPluginsI) > 0 {
					notificationPluginMap := notificationPluginsI[0].(map[string]interface{})
					configI := notificationPluginMap["config"].(map[string]interface{})
					config := map[string]string{}
					for k, v := range configI {
						config[k] = v.(string)
					}
					notification.Plugin = &JobPlugin{
						Type:   notificationPluginMap["type"].(string),
						Config: config,
					}
				}

				switch jobType {
				case "on_success":
					if jobNotification.OnSuccess != nil {
						return nil, fmt.Errorf("a block with %s already exists", jobType)
					}
					jobNotification.OnSuccess = &notification
					job.Notification = &jobNotification
				case "on_failure":
					if jobNotification.OnFailure != nil {
						return nil, fmt.Errorf("a block with %s already exists", jobType)
					}
					jobNotification.OnFailure = &notification
					job.Notification = &jobNotification
				case "on_start":
					if jobNotification.OnStart != nil {
						return nil, fmt.Errorf("a block with %s already exists", jobType)
					}
					jobNotification.OnStart = &notification
					job.Notification = &jobNotification
				default:
					return nil, fmt.Errorf("the notification type is not one of `on_success`, `on_failure`, `on_start`")
				}
			}
		} else {
			return nil, fmt.Errorf("can only have up to three notfication blocks, `on_success`, `on_failure`, `on_start`")
		}
	}
	return job, nil
}

func jobToResourceData(job *JobDetail, d *schema.ResourceData) error {

	d.SetId(job.ID)
	if err := d.Set("name", job.Name); err != nil {
		return err
	}
	if err := d.Set("group_name", job.GroupName); err != nil {
		return err
	}

	// The project name is not consistently returned in all rundeck versions,
	// so we'll only update it if it's set. Jobs can't move between projects
	// anyway, so this is harmless.
	if job.ProjectName != "" {
		if err := d.Set("project_name", job.ProjectName); err != nil {
			return err
		}
	}

	if err := d.Set("description", job.Description); err != nil {
		return err
	}
	if err := d.Set("execution_enabled", job.ExecutionEnabled); err != nil {
		return err
	}
	if err := d.Set("schedule_enabled", job.ScheduleEnabled); err != nil {
		return err
	}
	if err := d.Set("nodes_selected_by_default", job.NodesSelectedByDefault); err != nil {
		return err
	}
	if err := d.Set("time_zone", job.TimeZone); err != nil {
		return err
	}
	if err := d.Set("log_level", job.LogLevel); err != nil {
		return err
	}
	if err := d.Set("allow_concurrent_executions", job.AllowConcurrentExecutions); err != nil {
		return err
	}
	if err := d.Set("timeout", job.Timeout); err != nil {
		return err
	}
	if job.Retry != nil {
		if err := d.Set("retry", job.Retry.Value); err != nil {
			return err
		}
		if err := d.Set("retry_delay", job.Retry.Delay); err != nil {
			return err
		}
	}
	if job.Dispatch != nil {
		if err := d.Set("max_thread_count", job.Dispatch.MaxThreadCount); err != nil {
			return err
		}
		if err := d.Set("continue_next_node_on_error", job.Dispatch.ContinueNextNodeOnError); err != nil {
			return err
		}
		if err := d.Set("rank_attribute", job.Dispatch.RankAttribute); err != nil {
			return err
		}
		if err := d.Set("rank_order", job.Dispatch.RankOrder); err != nil {
			return err
		}
		if err := d.Set("success_on_empty_node_filter", job.Dispatch.SuccessOnEmptyNodeFilter); err != nil {
			return err
		}
	} else {
		if err := d.Set("max_thread_count", 1); err != nil {
			return err
		}
		if err := d.Set("continue_next_node_on_error", false); err != nil {
			return err
		}
		if err := d.Set("rank_attribute", nil); err != nil {
			return err
		}
		if err := d.Set("rank_order", "ascending"); err != nil {
			return err
		}
	}

	if job.NodeFilter != nil {
		if err := d.Set("node_filter_query", job.NodeFilter.Query); err != nil {
			return err
		}
		if err := d.Set("node_filter_exclude_query", job.NodeFilter.ExcludeQuery); err != nil {
			return err
		}
		if err := d.Set("node_filter_exclude_precedence", job.NodeFilter.ExcludePrecedence); err != nil {
			return err
		}
	} else {
		if err := d.Set("node_filter_query", nil); err != nil {
			return err
		}
		if err := d.Set("node_filter_exclude_query", nil); err != nil {
			return err
		}
		if err := d.Set("node_filter_exclude_precedence", nil); err != nil {
			return err
		}
	}

	optionConfigsI := make([]interface{}, 0)
	if job.OptionsConfig != nil {
		if err := d.Set("preserve_options_order", job.OptionsConfig.PreserveOrder); err != nil {
			return err
		}
		for _, option := range job.OptionsConfig.Options {
			optionConfigI := map[string]interface{}{
				"name":                      option.Name,
				"label":                     option.Label,
				"default_value":             option.DefaultValue,
				"value_choices":             option.ValueChoices,
				"value_choices_url":         option.ValueChoicesURL,
				"require_predefined_choice": option.RequirePredefinedChoice,
				"validation_regex":          option.ValidationRegex,
				"description":               option.Description,
				"required":                  option.IsRequired,
				"allow_multiple_values":     option.AllowsMultipleValues,
				"multi_value_delimiter":     option.MultiValueDelimiter,
				"obscure_input":             option.ObscureInput,
				"exposed_to_scripts":        option.ValueIsExposedToScripts,
				"storage_path":              option.StoragePath,
				"is_date":                   option.IsDate,
				"date_format":               option.DateFormat,
				"hidden":                    option.Hidden,
			}
			optionConfigsI = append(optionConfigsI, optionConfigI)
		}
	}
	if err := d.Set("option", optionConfigsI); err != nil {
		return err
	}

	if job.CommandSequence != nil {
		if err := d.Set("command_ordering_strategy", job.CommandSequence.OrderingStrategy); err != nil {
			return err
		}
		if err := d.Set("continue_on_error", job.CommandSequence.ContinueOnError); err != nil {
			return err
		}

		if job.CommandSequence.GlobalLogFilters != nil && len(*job.CommandSequence.GlobalLogFilters) > 0 {
			globalLogFilterConfigsI := make([]interface{}, 0)
			for _, logFilter := range *job.CommandSequence.GlobalLogFilters {
				logFilterI := map[string]interface{}{
					"type":   logFilter.Type,
					"config": map[string]string(*logFilter.Config),
				}
				globalLogFilterConfigsI = append(globalLogFilterConfigsI, logFilterI)
			}
			if err := d.Set("global_log_filter", globalLogFilterConfigsI); err != nil {
				return err
			}
		}

		commandConfigsI := make([]interface{}, 0)
		for i := range job.CommandSequence.Commands {
			commandConfigI, err := commandToResourceData(&job.CommandSequence.Commands[i])
			if err != nil {
				return err
			}
			commandConfigsI = append(commandConfigsI, commandConfigI)
		}
		if err := d.Set("command", commandConfigsI); err != nil {
			return err
		}
	}

	if job.Schedule != nil {
		cronSpec, err := scheduleToCronSpec(job.Schedule)
		if err != nil {
			return err
		}
		if err := d.Set("schedule", cronSpec); err != nil {
			return err
		}
	}
	notificationConfigsI := make([]interface{}, 0)
	if job.Notification != nil {
		if job.Notification.OnSuccess != nil {
			notificationConfigI := readNotification(job.Notification.OnSuccess, "on_success")
			notificationConfigsI = append(notificationConfigsI, notificationConfigI)
		}
		if job.Notification.OnFailure != nil {
			notificationConfigI := readNotification(job.Notification.OnFailure, "on_failure")
			notificationConfigsI = append(notificationConfigsI, notificationConfigI)
		}
		if job.Notification.OnStart != nil {
			notificationConfigI := readNotification(job.Notification.OnStart, "on_start")
			notificationConfigsI = append(notificationConfigsI, notificationConfigI)
		}
	}

	if err := d.Set("notification", notificationConfigsI); err != nil {
		return err
	}

	return nil
}

func JobScheduleFromResourceData(d *schema.ResourceData, job *JobDetail) error {
	const scheduleKey = "schedule"
	cronSpec := d.Get(scheduleKey).(string)
	if cronSpec != "" {
		schedule := strings.Split(cronSpec, " ")
		if len(schedule) != 7 {
			return fmt.Errorf("the Rundeck schedule must be formatted like a cron expression, as defined here: http://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/tutorial-lesson-06.html")
		}
		job.Schedule = &JobSchedule{
			Time: JobScheduleTime{
				Seconds: schedule[0],
				Minute:  schedule[1],
				Hour:    schedule[2],
			},
			Month: JobScheduleMonth{
				Day:   schedule[3],
				Month: schedule[4],
			},
			WeekDay: JobScheduleWeekDay{
				Day: schedule[5],
			},
			Year: JobScheduleYear{
				Year: schedule[6],
			},
		}
		// Day-of-month and Day-of-week can both be asterisks, but otherwise one, and only one, must be a '?'
		if job.Schedule.Month.Day == job.Schedule.WeekDay.Day {
			if job.Schedule.Month.Day != "*" {
				return fmt.Errorf("invalid '%s' specification %s - one of day-of-month (4th item) or day-of-week (6th) must be '?'", scheduleKey, cronSpec)
			}
		} else if job.Schedule.Month.Day != "?" && job.Schedule.WeekDay.Day != "?" {
			return fmt.Errorf("invalid '%s' specification %s - one of day-of-month (4th item) or day-of-week (6th) must be '?'", scheduleKey, cronSpec)
		}
	}
	return nil
}

func scheduleToCronSpec(schedule *JobSchedule) (string, error) {
	if schedule.Month.Day == "" {
		if schedule.WeekDay.Day == "*" || schedule.WeekDay.Day == "" {
			schedule.Month.Day = "*"
		} else {
			schedule.Month.Day = "?"
		}
	}
	if schedule.WeekDay.Day == "" {
		if schedule.Month.Day == "*" {
			schedule.WeekDay.Day = "*"
		} else {
			schedule.WeekDay.Day = "?"
		}
	}
	cronSpec := make([]string, 0)
	cronSpec = append(cronSpec, schedule.Time.Seconds)
	cronSpec = append(cronSpec, schedule.Time.Minute)
	cronSpec = append(cronSpec, schedule.Time.Hour)
	cronSpec = append(cronSpec, schedule.Month.Day)
	cronSpec = append(cronSpec, schedule.Month.Month)
	cronSpec = append(cronSpec, schedule.WeekDay.Day)
	cronSpec = append(cronSpec, schedule.Year.Year)
	return strings.Join(cronSpec, " "), nil
}

func commandFromResourceData(commandI interface{}) (*JobCommand, error) {
	commandMap := commandI.(map[string]interface{})
	command := &JobCommand{
		Description:        commandMap["description"].(string),
		ShellCommand:       commandMap["shell_command"].(string),
		Script:             commandMap["inline_script"].(string),
		ScriptUrl:          commandMap["script_url"].(string),
		ScriptFile:         commandMap["script_file"].(string),
		ScriptFileArgs:     commandMap["script_file_args"].(string),
		KeepGoingOnSuccess: commandMap["keep_going_on_success"].(bool),
	}

	// Because of the lack of schema recursion, the inner command has a separate schema without an error_handler
	// field, but is otherwise identical. The 'exists' checks allow this function to apply to both 'command' and
	// 'errorHandler' schemas.
	if errorHandlersI, exists := commandMap["error_handler"].([]interface{}); exists {
		if len(errorHandlersI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one error handler")
		}
		if len(errorHandlersI) > 0 {
			errorHandlerMap := errorHandlersI[0].(map[string]interface{})
			errorHandler, err := commandFromResourceData(errorHandlerMap)
			if err != nil {
				return nil, err
			}
			command.ErrorHandler = errorHandler
		}
	}

	scriptInterpretersI := commandMap["script_interpreter"].([]interface{})
	if len(scriptInterpretersI) > 1 {
		return nil, fmt.Errorf("rundeck command may have no more than one script interpreter")
	}
	if len(scriptInterpretersI) > 0 {
		scriptInterpreterMap := scriptInterpretersI[0].(map[string]interface{})
		command.ScriptInterpreter = &JobCommandScriptInterpreter{
			InvocationString: scriptInterpreterMap["invocation_string"].(string),
			ArgsQuoted:       scriptInterpreterMap["args_quoted"].(bool),
		}
	}

	var err error
	if command.Job, err = jobCommandJobRefFromResourceData("job", commandMap); err != nil {
		return nil, err
	}
	if command.StepPlugin, err = singlePluginFromResourceData("step_plugin", commandMap); err != nil {
		return nil, err
	}
	if command.Plugins, err = pluginsFromResourceData("plugins", commandMap); err != nil {
		return nil, err
	}
	if command.NodeStepPlugin, err = singlePluginFromResourceData("node_step_plugin", commandMap); err != nil {
		return nil, err
	}

	return command, nil
}

func jobCommandJobRefFromResourceData(key string, commandMap map[string]interface{}) (*JobCommandJobRef, error) {
	jobRefsI := commandMap[key].([]interface{})
	if len(jobRefsI) > 1 {
		return nil, fmt.Errorf("rundeck command may have no more than one %s", key)
	}
	if len(jobRefsI) == 0 {
		return nil, nil
	}
	jobRefMap := jobRefsI[0].(map[string]interface{})
	jobRef := &JobCommandJobRef{
		Name:                jobRefMap["name"].(string),
		GroupName:           jobRefMap["group_name"].(string),
		Project:             jobRefMap["project_name"].(string),
		RunForEachNode:      jobRefMap["run_for_each_node"].(bool),
		Arguments:           JobCommandJobRefArguments(jobRefMap["args"].(string)),
		ChildNodes:          jobRefMap["child_nodes"].(bool),
		FailOnDisable:       jobRefMap["fail_on_disable"].(bool),
		ImportOptions:       jobRefMap["import_options"].(bool),
		IgnoreNotifications: jobRefMap["ignore_notifications"].(bool),
	}
	nodeFiltersI := jobRefMap["node_filters"].([]interface{})
	if len(nodeFiltersI) > 1 {
		return nil, fmt.Errorf("rundeck command job reference may have no more than one node filter")
	}
	if len(nodeFiltersI) > 0 {
		nodeFilterMap := nodeFiltersI[0].(map[string]interface{})
		jobRef.NodeFilter = &JobNodeFilter{
			Query:             nodeFilterMap["filter"].(string),
			ExcludeQuery:      nodeFilterMap["exclude_filter"].(string),
			ExcludePrecedence: nodeFilterMap["exclude_precedence"].(bool),
		}
	}
	return jobRef, nil
}

func pluginsFromResourceData(key string, commandMap map[string]interface{}) (*JobPlugins, error) {
	pluginsList := commandMap[key].([]interface{})
	if len(pluginsList) == 0 {
		return nil, nil
	}
	if len(pluginsList) > 1 {
		return nil, fmt.Errorf("rundeck command job reference may have no more than one plugins section")
	}
	pluginsI := pluginsList[0].(map[string]interface{})
	var plugins = new(JobPlugins)
	for _, pluginI := range pluginsI["log_filter_plugin"].([]interface{}) {
		pluginMap := pluginI.(map[string]interface{})
		configI := pluginMap["config"].(map[string]interface{})
		var config = JobLogFilterConfig{}
		for key, value := range configI {
			config[key] = value.(string)
		}
		plugin := &JobLogFilter{
			Type:   pluginMap["type"].(string),
			Config: &config,
		}
		plugins.LogFilterPlugins = append(plugins.LogFilterPlugins, *plugin)
	}
	return plugins, nil
}

func singlePluginFromResourceData(key string, commandMap map[string]interface{}) (*JobPlugin, error) {
	stepPluginsI := commandMap[key].([]interface{})
	if len(stepPluginsI) > 1 {
		return nil, fmt.Errorf("rundeck command may have no more than one %s", key)
	}
	if len(stepPluginsI) == 0 {
		return nil, nil
	}
	stepPluginMap := stepPluginsI[0].(map[string]interface{})
	configI := stepPluginMap["config"].(map[string]interface{})
	config := map[string]string{}
	for key, value := range configI {
		config[key] = value.(string)
	}
	result := &JobPlugin{
		Type:   stepPluginMap["type"].(string),
		Config: config,
	}
	return result, nil
}

func commandToResourceData(command *JobCommand) (map[string]interface{}, error) {
	commandConfigI := map[string]interface{}{
		"description":           command.Description,
		"shell_command":         command.ShellCommand,
		"inline_script":         command.Script,
		"script_url":            command.ScriptUrl,
		"script_file":           command.ScriptFile,
		"script_file_args":      command.ScriptFileArgs,
		"keep_going_on_success": command.KeepGoingOnSuccess,
	}

	if command.ErrorHandler != nil {
		errorHandlerI, err := commandToResourceData(command.ErrorHandler)
		if err != nil {
			return nil, err
		}
		commandConfigI["error_handler"] = []interface{}{
			errorHandlerI,
		}
	}

	if command.ScriptInterpreter != nil {
		commandConfigI["script_interpreter"] = []interface{}{
			map[string]interface{}{
				"invocation_string": command.ScriptInterpreter.InvocationString,
				"args_quoted":       command.ScriptInterpreter.ArgsQuoted,
			},
		}
	}

	if command.Job != nil {
		jobRefConfigI := map[string]interface{}{
			"name":                 command.Job.Name,
			"group_name":           command.Job.GroupName,
			"run_for_each_node":    command.Job.RunForEachNode,
			"args":                 command.Job.Arguments,
			"child_nodes":          command.Job.ChildNodes,
			"fail_on_disable":      command.Job.FailOnDisable,
			"import_options":       command.Job.ImportOptions,
			"ignore_notifications": command.Job.IgnoreNotifications,
		}
		if command.Job.NodeFilter != nil {
			nodeFilterConfigI := map[string]interface{}{
				"exclude_precedence": command.Job.NodeFilter.ExcludePrecedence,
				"filter":             command.Job.NodeFilter.Query,
				"exclude_filter":     command.Job.NodeFilter.ExcludeQuery,
			}
			jobRefConfigI["node_filters"] = append([]interface{}{}, nodeFilterConfigI)
		}
		commandConfigI["job"] = append([]interface{}{}, jobRefConfigI)
	}
	if command.StepPlugin != nil {
		commandConfigI["step_plugin"] = []interface{}{
			map[string]interface{}{
				"type":   command.StepPlugin.Type,
				"config": map[string]string(command.StepPlugin.Config),
			},
		}
	}

	if command.Plugins != nil {
		logFilterPluginI := []map[string]interface{}{}
		for _, plugin := range command.Plugins.LogFilterPlugins {
			pluginI := map[string]interface{}{
				"type":   plugin.Type,
				"config": (map[string]string)(*plugin.Config),
			}
			logFilterPluginI = append(logFilterPluginI, pluginI)
		}
		commandConfigI["plugins"] = append([]interface{}{}, map[string]interface{}{
			"log_filter_plugin": logFilterPluginI,
		})
	}

	if command.NodeStepPlugin != nil {
		commandConfigI["node_step_plugin"] = []interface{}{
			map[string]interface{}{
				"type":   command.NodeStepPlugin.Type,
				"config": map[string]string(command.NodeStepPlugin.Config),
			},
		}
	}
	return commandConfigI, nil
}

// Helper function for three different notifications
func readNotification(notification *Notification, notificationType string) map[string]interface{} {
	notificationConfigI := map[string]interface{}{
		"type": notificationType,
	}
	if notification.WebHook != nil {
		notificationConfigI["webhook_urls"] = notification.WebHook.Urls
	}
	if notification.Email != nil {
		notificationConfigI["email"] = []interface{}{
			map[string]interface{}{
				"attach_log": notification.Email.AttachLog,
				"subject":    notification.Email.Subject,
				"recipients": notification.Email.Recipients,
			},
		}
	}
	if notification.Plugin != nil {
		notificationConfigI["plugin"] = []interface{}{
			map[string]interface{}{
				"type":   notification.Plugin.Type,
				"config": map[string]string(notification.Plugin.Config),
			},
		}
	}
	return notificationConfigI
}

func resourceJobImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	idAttr := strings.SplitN(d.Id(), "/", 2)
	var jobID string
	var projectName string

	if len(idAttr) == 2 {
		projectName = idAttr[0]
		jobID = idAttr[1]
	} else {
		return nil, fmt.Errorf("invalid id %q specified, should be in format \"projectName/JobUUID\" for import", d.Id())
	}
	d.SetId(jobID)

	err := ReadJob(d, meta)
	if err != nil {
		return nil, err
	}
	// Get the information out of the api if available.
	// Otherwise use information supplied by user.
	if d.Get("project_name") == "" {
		d.Set("project_name", projectName)
	}

	return []*schema.ResourceData{d}, nil
}
