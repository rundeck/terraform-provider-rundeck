package rundeck

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
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

			"max_thread_count": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
			},

			"continue_on_error": {
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

			"node_filter_exclude_precedence": {
				Type:     schema.TypeBool,
				Optional: true,
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
					},
				},
			},

			"command": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
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

						"script_file": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"script_file_args": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"job": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"group_name": {
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
									"nodefilters": {
										Type:     schema.TypeMap,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"excludeprecedence": {
													Type:     schema.TypeString,
													Optional: true,
												},
												"filter": {
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
								},
							},
						},

						"step_plugin": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceRundeckJobPluginResource(),
						},

						"node_step_plugin": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     resourceRundeckJobPluginResource(),
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
		if _, ok := err.(*NotFoundError); ok == true {
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

	return false, fmt.Errorf("Error checking if job exists: (%v)", err)
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
		ScheduleEnabled:           d.Get("schedule_enabled").(bool),
		LogLevel:                  d.Get("log_level").(string),
		AllowConcurrentExecutions: d.Get("allow_concurrent_executions").(bool),
		Dispatch: &JobDispatch{
			MaxThreadCount:  d.Get("max_thread_count").(int),
			ContinueOnError: d.Get("continue_on_error").(bool),
			RankAttribute:   d.Get("rank_attribute").(string),
			RankOrder:       d.Get("rank_order").(string),
		},
	}
	successOnEmpty := d.Get("success_on_empty_node_filter")
	if successOnEmpty != nil {
		job.Dispatch.SuccessOnEmptyNodeFilter = successOnEmpty.(bool)
	}

	sequence := &JobCommandSequence{
		ContinueOnError:  d.Get("continue_on_error").(bool),
		OrderingStrategy: d.Get("command_ordering_strategy").(string),
		Commands:         []JobCommand{},
	}

	commandConfigs := d.Get("command").([]interface{})
	for _, commandI := range commandConfigs {
		commandMap := commandI.(map[string]interface{})
		command := JobCommand{
			Description:    commandMap["description"].(string),
			ShellCommand:   commandMap["shell_command"].(string),
			Script:         commandMap["inline_script"].(string),
			ScriptFile:     commandMap["script_file"].(string),
			ScriptFileArgs: commandMap["script_file_args"].(string),
		}

		jobRefsI := commandMap["job"].([]interface{})
		if len(jobRefsI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one job")
		}

		if len(jobRefsI) > 0 {
			jobRefMap := jobRefsI[0].(map[string]interface{})
			command.Job = &JobCommandJobRef{
				Name:           jobRefMap["name"].(string),
				GroupName:      jobRefMap["group_name"].(string),
				RunForEachNode: jobRefMap["run_for_each_node"].(bool),
				Arguments:      JobCommandJobRefArguments(jobRefMap["args"].(string)),
				NodeFilter:     &JobNodeFilter{},
			}
			nodeFilterMap := jobRefMap["nodefilters"].(map[string]interface{})
			if nodeFilterMap["filter"] != nil {
				command.Job.NodeFilter.Query = nodeFilterMap["filter"].(string)
			}
			if nodeFilterMap["excludeprecedence"] != nil {
				command.Job.NodeFilter.ExcludePrecedence = nodeFilterMap["excludeprecedence"].(bool)
			}
		}

		stepPluginsI := commandMap["step_plugin"].([]interface{})
		if len(stepPluginsI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one step plugin")
		}
		if len(stepPluginsI) > 0 {
			stepPluginMap := stepPluginsI[0].(map[string]interface{})
			configI := stepPluginMap["config"].(map[string]interface{})
			config := map[string]string{}
			for k, v := range configI {
				config[k] = v.(string)
			}
			command.StepPlugin = &JobPlugin{
				Type:   stepPluginMap["type"].(string),
				Config: config,
			}
		}

		stepPluginsI = commandMap["node_step_plugin"].([]interface{})
		if len(stepPluginsI) > 1 {
			return nil, fmt.Errorf("rundeck command may have no more than one node step plugin")
		}
		if len(stepPluginsI) > 0 {
			stepPluginMap := stepPluginsI[0].(map[string]interface{})
			configI := stepPluginMap["config"].(map[string]interface{})
			config := map[string]string{}
			for k, v := range configI {
				config[k] = v.(string)
			}
			command.NodeStepPlugin = &JobPlugin{
				Type:   stepPluginMap["type"].(string),
				Config: config,
			}
		}

		sequence.Commands = append(sequence.Commands, command)
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
			}
			if option.StoragePath != "" && option.ObscureInput == false {
				return nil, fmt.Errorf("Argument \"obscure_input\" must be set to `true` when \"storage_path\" is not empty.")
			}
			if option.ValueIsExposedToScripts && option.ObscureInput == false {
				return nil, fmt.Errorf("Argument \"obscure_input\" must be set to `true` when \"exposed_to_scripts\" is set to true.")
			}

			for _, iv := range optionMap["value_choices"].([]interface{}) {
				if iv == nil {
					return nil, fmt.Errorf("Argument \"value_choices\" can not have empty values; try \"required\"")
				}
				option.ValueChoices = append(option.ValueChoices, iv.(string))
			}

			optionsConfig.Options = append(optionsConfig.Options, option)
		}
		job.OptionsConfig = optionsConfig
	}

	if d.Get("node_filter_query").(string) != "" {
		job.NodeFilter = &JobNodeFilter{
			ExcludePrecedence: d.Get("node_filter_exclude_precedence").(bool),
			Query:             d.Get("node_filter_query").(string),
		}

		job.Dispatch = &JobDispatch{
			MaxThreadCount:  d.Get("max_thread_count").(int),
			ContinueOnError: d.Get("continue_on_error").(bool),
			RankAttribute:   d.Get("rank_attribute").(string),
			RankOrder:       d.Get("rank_order").(string),
		}

		successOnEmpty := d.Get("success_on_empty_node_filter")
		if successOnEmpty != nil {
			job.Dispatch.SuccessOnEmptyNodeFilter = successOnEmpty.(bool)
		}
	}

	if d.Get("schedule").(string) != "" {
		schedule := strings.Split(d.Get("schedule").(string), " ")
		if len(schedule) != 7 {
			return nil, fmt.Errorf("Rundeck schedule must be formated like a cron expression, as defined here: http://www.quartz-scheduler.org/documentation/quartz-2.2.x/tutorials/tutorial-lesson-06.html")
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
			WeekDay: &JobScheduleWeekDay{
				Day: schedule[5],
			},
			Year: JobScheduleYear{
				Year: schedule[6],
			},
		}
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
						return nil, fmt.Errorf("A block with %s already exists", jobType)
					}
					jobNotification.OnSuccess = &notification
					job.Notification = &jobNotification
				case "on_failure":
					if jobNotification.OnFailure != nil {
						return nil, fmt.Errorf("A block with %s already exists", jobType)
					}
					jobNotification.OnFailure = &notification
					job.Notification = &jobNotification
				case "on_start":
					if jobNotification.OnStart != nil {
						return nil, fmt.Errorf("A block with %s already exists", jobType)
					}
					jobNotification.OnStart = &notification
					job.Notification = &jobNotification
				default:
					return nil, fmt.Errorf("The notification type is not one of `on_success`, `on_failure`, `on_start`")
				}
			}
		} else {
			return nil, fmt.Errorf("Can only have up to three notfication blocks, `on_success`, `on_failure`, `on_start`")
		}
	}

	return job, nil
}

func jobToResourceData(job *JobDetail, d *schema.ResourceData) error {

	d.SetId(job.ID)
	d.Set("name", job.Name)
	d.Set("group_name", job.GroupName)

	// The project name is not consistently returned in all rundeck versions,
	// so we'll only update it if it's set. Jobs can't move between projects
	// anyway, so this is harmless.
	if job.ProjectName != "" {
		d.Set("project_name", job.ProjectName)
	}

	d.Set("description", job.Description)
	d.Set("execution_enabled", job.ExecutionEnabled)
	d.Set("schedule_enabled", job.ScheduleEnabled)
	d.Set("log_level", job.LogLevel)
	d.Set("allow_concurrent_executions", job.AllowConcurrentExecutions)

	if job.Dispatch != nil {
		d.Set("max_thread_count", job.Dispatch.MaxThreadCount)
		d.Set("continue_on_error", job.Dispatch.ContinueOnError)
		d.Set("rank_attribute", job.Dispatch.RankAttribute)
		d.Set("rank_order", job.Dispatch.RankOrder)
		d.Set("success_on_empty_node_filter", job.Dispatch.SuccessOnEmptyNodeFilter)
	} else {
		d.Set("max_thread_count", 1)
		d.Set("continue_on_error", nil)
		d.Set("rank_attribute", nil)
		d.Set("rank_order", "ascending")
	}

	d.Set("node_filter_query", nil)
	d.Set("node_filter_exclude_precedence", nil)
	if job.NodeFilter != nil {
		d.Set("node_filter_query", job.NodeFilter.Query)
		d.Set("node_filter_exclude_precedence", job.NodeFilter.ExcludePrecedence)
	}

	optionConfigsI := []interface{}{}
	if job.OptionsConfig != nil {
		d.Set("preserve_options_order", job.OptionsConfig.PreserveOrder)
		for _, option := range job.OptionsConfig.Options {
			optionConfigI := map[string]interface{}{
				"name":                      option.Name,
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
			}
			optionConfigsI = append(optionConfigsI, optionConfigI)
		}
	}
	d.Set("option", optionConfigsI)

	commandConfigsI := []interface{}{}
	if job.CommandSequence != nil {
		d.Set("command_ordering_strategy", job.CommandSequence.OrderingStrategy)
		for _, command := range job.CommandSequence.Commands {
			commandConfigI := map[string]interface{}{
				"description":      command.Description,
				"shell_command":    command.ShellCommand,
				"inline_script":    command.Script,
				"script_file":      command.ScriptFile,
				"script_file_args": command.ScriptFileArgs,
			}

			if command.Job != nil {

				var nodeFilterMap map[string]interface{}
				if command.Job.NodeFilter != nil {
					nodeFilterMap = make(map[string]interface{})
					nodeFilterMap["filter"] = command.Job.NodeFilter.Query
				}
				commandConfigI["job"] = []interface{}{
					map[string]interface{}{
						"name":              command.Job.Name,
						"group_name":        command.Job.GroupName,
						"run_for_each_node": command.Job.RunForEachNode,
						"args":              command.Job.Arguments,
						"nodefilters":       nodeFilterMap,
					},
				}
			}

			if command.StepPlugin != nil {
				commandConfigI["step_plugin"] = []interface{}{
					map[string]interface{}{
						"type":   command.StepPlugin.Type,
						"config": map[string]string(command.StepPlugin.Config),
					},
				}
			}

			if command.NodeStepPlugin != nil {
				commandConfigI["node_step_plugin"] = []interface{}{
					map[string]interface{}{
						"type":   command.NodeStepPlugin.Type,
						"config": map[string]string(command.NodeStepPlugin.Config),
					},
				}
			}

			commandConfigsI = append(commandConfigsI, commandConfigI)
		}
	}
	d.Set("command", commandConfigsI)

	if job.Schedule != nil {
		schedule := []string{}
		schedule = append(schedule, job.Schedule.Time.Seconds)
		schedule = append(schedule, job.Schedule.Time.Minute)
		schedule = append(schedule, job.Schedule.Time.Hour)
		if job.Schedule.Month.Day != "" {
			schedule = append(schedule, job.Schedule.Month.Day)
		} else {
			schedule = append(schedule, "?")
		}
		schedule = append(schedule, job.Schedule.Month.Month)
		if job.Schedule.WeekDay != nil {
			schedule = append(schedule, job.Schedule.WeekDay.Day)
		} else {
			schedule = append(schedule, "*")
		}
		schedule = append(schedule, job.Schedule.Year.Year)

		d.Set("schedule", strings.Join(schedule, " "))
	}
	notificationConfigsI := []interface{}{}
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

	d.Set("notification", notificationConfigsI)

	return nil
}

// Helper function for three different notifications
func readNotification(notification *Notification, notificationType string) map[string]interface{} {
	notificationConfigI := map[string]interface{}{
		"type": notificationType,
	}
	if notification.WebHook != nil {
		notificationConfigI["webhok_urls"] = notification.WebHook.Urls
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
