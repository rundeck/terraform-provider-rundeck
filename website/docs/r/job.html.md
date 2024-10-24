---
layout: "rundeck"
page_title: "Rundeck: rundeck_job"
sidebar_current: "docs-rundeck-resource-job"
description: |-
  The rundeck_job resource allows Rundeck jobs to be managed by Terraform.
---

# rundeck\_job

The job resource allows Rundeck jobs to be managed by Terraform. In Rundeck a job is a particular
named set of steps that can be executed against one or more of the nodes configured for its
associated project.

Each job belongs to a project. A project can be created with the `rundeck_project` resource.

## Example Usage

```hcl
  resource "rundeck_job" "bounceweb" {
      name              = "Bounce All Web Servers"
      project_name      = "${rundeck_project.terraform.name}"
      node_filter_query = "tags: web"
      description       = "Restart the service daemons on all the web servers"

      command {
          shell_command = "sudo service anvils restart"
      }
      notification {
          type = "on_success"
          email {
              recipients = ["example@foo.bar"]
          }
      }
  }
```

## Example Usage (Key-Value Data Log filter to pass data between jobs)

```hcl
  resource "rundeck_job" "update_review_environments" {
      name              = "Update review environments"
      project_name      = "${rundeck_project.terraform.name}"
      node_filter_query = "tags: dev_server"
      description       = "Update the code in review environments checking out the given branch"
      command {
        description           = null
        inline_script         = "#!/bin/sh\nenvironment_numbers=$(find /var/review_environments -mindepth 3 -maxdepth 4 -name '$1' | awk -F/ -vORS=, '{ print $3 }' | sed 's/.$//')\necho \"RUNDECK:DATA:environment_numbers=\"$environment_numbers\""
        script_file_args      = "$${option.git_branch}"
        plugins {
          log_filter_plugin {
            config = {
              invalidKeyPattern = "\\s|\\$|\\{|\\}|\\\\"
              logData           = "true"
              regex             = "^RUNDECK:DATA:\\s*([^\\s]+?)\\s*=\\s*(.+)$"
            }
            type = "key-value-data"
          }
        }
      }
    command {
      job {
        args              = "-environment_numbers $${data.environment_numbers}"
        name              = "git_pull_review_environments"
      }
    }
    option {
      name                      = "git_branch"
      required                  = true
    }
  }

```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the job, used to describe the job in the Rundeck UI.

* `description` - (Required) A longer description of the job, describing the job in the Rundeck UI.

* `project_name` - (Required) The name of the project that this job should belong to.

* `execution_enabled` - (Optional) If you want job execution to be enabled or disabled. Defaults to `true`.

* `default_tab` - (Optional) The default tab to show during job execution. Set to 'output' to follow the execution log. Must be set to `output`, `html`, or `nodes`

* `group_name` - (Optional) The name of a group within the project in which to place the job.
  Setting this creates collapsable subcategories within the Rundeck UI's project job index.

* `log_level` - (Optional) The log level that Rundeck should use for this job. Defaults to "INFO".

* `timeout` - (Optional) The maximum time for an execution to run. Time in seconds, or specify time units: "120m", "2h", "3d". Use blank or 0 to indicate no timeout.

* `schedule` - (Optional) The job's schedule in Quartz schedule cron format. Similar to unix crontab, but with seven fields instead of five: Second Minute Hour Day-of-Month Month Day-of-Week Year

* `orchestrator` - (Optional) The orchestrator for the job, described below and [here](https://docs.rundeck.com/docs/manual/orchestrator-plugins/bundled.html)
    - `type`: Must be one of `subset`, `rankTiered`, `maxPercentage`, `orchestrator-highest-lowest-attribute`
    - `count`: Required when types is `subset`. Selects that max number of the target nodes at random to process.
    - `percent`: Required when type is `maxPercentage`. Used to determine max percentage of the nodes to process at once.
    - `attribute`: Required when type is `orchestrator-highest-lowest-attribute`.  The node attribute to use for sorting.
    - `sort`:Required when type is `orchestrator-highest-lowest-attribute`.  Values accepted are `highest` or `lowest`.

* `schedule_enabled` - (Optional) Sets the job schedule to be enabled or disabled. Defaults to `true`.

* `time_zone` - (Optional) A valid Time Zone, either an abbreviation such as "PST", a full name such as
  "America/Los_Angeles",or a custom ID such as "GMT-8:00".

* `allow_concurrent_executions` - (Optional) Boolean defining whether two or more executions of
  this job can run concurrently. The default is `false`, meaning that jobs will only run
  sequentially.

* `retry` - (Optional) Maximum number of times to retry execution when this job is directly invoked.
  Retry will occur if the job fails or times out, but not if it is manually killed. Can use an option
  value reference like "${option.retry}". The default is `0`, meaning that jobs will only run
  once.

* `retry_delay` - (Optional) The time between the failed execution and the retry. Time in seconds or
  specify time units: "120m", "2h", "3d". Use 0 to indicate no delay. Can include option value
  references like "${option.delay}". The default is 0.

* `max_thread_count` - (Optional) The maximum number of threads to use to execute this job, which
  controls on how many nodes the commands can be run simultaneously. Defaults to 1, meaning that
  the nodes will be visited sequentially.

* `continue_on_error` - (Optional) Boolean defining whether Rundeck will continue to run
  subsequent steps if any intermediate step fails. Defaults to `false`, meaning that execution
  will stop and the execution will be considered to have failed.

* `rank_attribute` - (Optional) The name of the attribute that will be used to decide in which
  order the nodes will be visited while executing the job across multiple nodes.

* `rank_order` - (Optional) Keyword deciding which direction the nodes are sorted in terms of
  the chosen `rank_attribute`. May be either "ascending" (the default) or "descending".

* `success_on_empty_node_filter` - (Optional) Boolean determining if an empty node filter yields
  a successful result.

* `preserve_options_order`: (Optional) Boolean controlling whether the configured options will
  be presented in their configuration order when shown in the Rundeck UI. The default is `false`,
  which means that the options will be displayed in alphabetical order by name.

* `command_ordering_strategy`: (Optional) The name of the strategy used to describe how to
  traverse the matrix of nodes and commands. The default is "node-first", meaning that all commands
  will be executed on a single node before moving on to the next. May also be set to "step-first",
  meaning that a single step will be executed across all nodes before moving on to the next step, or
  "parallel", meaning that all nodes can execute the script simultaneously.

* `continue_next_node_on_error` - (Optional) Boolean defining whether Rundeck will continue to run
  on subsequent nodes if a node fails when the `command_ordering_strategy` is set to either "node-first"
  or "step-first". Defaults to `false`, meaning that the job execution will not proceed to the next node.

* `node_filter_query` - (Optional) A query string using
  [Rundeck's node filter language](http://rundeck.org/docs/manual/node-filters.html#node-filter-syntax)
  that defines which subset of the project's nodes ***will*** be used to execute this job. If neither
  `node_filter_query` nor `node_filter_exclude_query` is defined, the job will be performed locally on the
  Rundeck server.

* `node_filter_exclude_query` - (Optional) A query string using
  [Rundeck's node filter language](http://rundeck.org/docs/manual/node-filters.html#node-filter-syntax)
  that defines which subset of the project's nodes ***will not*** be used to execute this job.

* `node_filter_exclude_precedence`: (Optional, Deprecated) Boolean controlling a deprecated Rundeck feature that
  controls whether node exclusions take priority over inclusions.

* `nodes_selected_by_default`: (Optional) Boolean controlling whether nodes that match the node_query_filter are
  selected by default or not.

* `command`: (Required) Nested block defining one step in the job workflow. A job must have one or
  more commands. The structure of this nested block is described below.

* `global_log_filter`: (Optional) Nested block defining a global log filter plugin available to provide communication
  between all workflow steps. A job may have multiple global log filters.  The structure of this nested block is
  described below.

* `notification`: (Optional) Nested block defining notifications on the job workflow. The structure of this nested block
  is described below.

* `option`: (Optional) Nested block defining an option a user may set when executing this job. A
  job may have any number of options. The structure of this nested block is described below.

`option` blocks have the following structure:

* `name`: (Required) Unique name that will be shown in the UI when entering values and used as
  a variable name for template substitutions.

* `label`: (Optional) Display label that will be shown in the UI instead of the name.

* `default_value`: (Optional) A default value for the option.

* `value_choices`: (Optional) A list of strings giving a set of predefined values that the user
  may choose from when entering a value for the option.

* `value_choices_url`: (Optional) Can be used instead of `value_choices` to cause Rundeck to
  obtain a list of choices dynamically by fetching this URL.

* `require_predefined_choice`: (Optional) Boolean controlling whether the user is allowed to
  enter values not included in the predefined set of choices (`false`, the default) or whether
  a predefined choice is required (`true`).

* `validation_regex`: (Optional) A regular expression that a provided value must match in order
  to be accepted.

* `description`: (Optional) A longer description of the option to be shown in the UI.

* `required`: (Optional) Boolean defining whether the user must provide a value for the option.
  Defaults to `false`.

* `allow_multiple_values`: (Optional) Boolean defining whether the user may select multiple values
  from the set of predefined values. Defaults to `false`, meaning that the user may choose only
  one value.

* `multi_value_delimiter`: (Optional) Delimiter used to join together multiple values into a single
  string when `allow_multiple_values` is set and the user chooses multiple values.

* `obscure_input`: (Optional) Boolean controlling whether the value of this option should be obscured
  during entry and in execution logs. Defaults to `false`, but should be set to `true` when the
  requested value is a password, private key or any other secret value. This must be set to `true` when
  `storage_path` is not null.

* `exposed_to_scripts`: (Optional) Boolean controlling whether the value of this option is available
  to scripts executed by job commands. Defaults to `false`. When `true`, `obscure_input` must also be set
  to `true`.

* `storage_path`: (Optional) String of the path where the key is stored on rundeck. `obscure_input` must be set to
  `true` when using this. This results in `Secure Remote Authentication` input type. Setting `exposed_to_scripts` also
  `true` results in `Secure` input type.

* `hidden`: (Optional) Boolean controlling whether this option should be hidden from the UI on the job run page.
  Defaults to `false`.

`command` blocks must have any one of the following combinations of arguments as contents:

* `description`: (Optional) gives a description to the command block.

* `shell_command` gives a single shell command to execute on the nodes.

* `inline_script` gives a whole shell script, inline in the configuration, to execute on the nodes.

* `script_file` and `script_file_args` together describe a script that is already pre-installed
  on the nodes which is to be executed.

* `script_url` can be used to provide a URL to execute a script from a specified url.

* A `script_interpreter` block (Optional), described below, is an advanced feature specifying how
  to invoke the script file.

* A `job` block, described below, causes another job within the same project to be executed as
  a command.
* A `step_plugin` block, described below, causes a step plugin to be executed as a command.

* A `plugins` block, described below, contains a list of plugins to add to the command. At the moment, only [Log Filters](https://docs.rundeck.com/docs/manual/log-filters/) are supported

* A `node_step_plugin` block, described below, causes a node step plugin to be executed once for
  each node.

A command's `script_interpreter` block has the following structure:

* `invocation_string`: (Optional) The string describing how to invoke the script file. By
  default the temporary script file path will be appended to this string, followed by any
  arguments. Include `${scriptfile}` anywhere to change the file path argument location.

* `args_quoted`: (Optional) Quote arguments to script invocation string?

A command's `job` block has the following structure:

* `name`: (Required) The name of the job to execute. If no specific `project_name` was given the target job should be in the same project as the current job.

* `group_name`: (Optional) The name of the group that the target job belongs to, if any.

* `project_name` - (Optional) The name of another project that holds the target job.

* `run_for_each_node`: (Optional) Boolean controlling whether the job is run only once (`false`,
  the default) or whether it is run once for each node (`true`).

* `args`: (Optional) A string giving the arguments to pass to the target job, using
  [Rundeck's job arguments syntax](http://rundeck.org/docs/manual/jobs.html#job-reference-step).

* `import_options`: (Optional) Pass as argument any options that match the referenced job's options.

* `skip_notifications` (Optional) If the referenced job has notifications, they will be skipped.

* `fail_on_disable` (Optional) If the referenced job has disabled execution, it will be considered a failure 

* `child_nodes`: (Optional) If the referenced job is from another project, you can use referenced job node list instead of the parent's nodes. 

* `node_filters`: (Optional) A map for overriding the referenced job's node filters.

A command's `node_filters` block has the following structure:

* `exclude_precedence`: (Optional, Deprecated) Whether to give precedence to the exclusion filter or not.

* `filter`: (Optional) The query string for nodes ***to use***.

* `exclude_filter`: (Optional) The query string for nodes ***not to use***.

A command's `plugins` block has the following structure:

* `log_filter_plugin`: A log filter plugin to add to the command. Can be repeated to add multiple log filters. See below for the structure.

A command's `log_filter_plugin`, `step_plugin`  or `node_step_plugin` block both have the following structure, as does the job's
  `global_log_filter` blocks:

* `type`: (Required) The name of the plugin to execute.

* `config`: (Optional) Map of arbitrary configuration parameters for the selected plugin.

`notification` blocks have the following structure:

* `type`: (Required) The name of the type of notification. Can be of type `on_success`, `on_failure`, `on_start`.

* `email`: (Optional) block listed below to send emails as notificiation.

* `webhook_urls`: (Optional) A list of urls to send a webhook notification.

* `plugin`: (Optional) A block listed below to send notifications using a plugin.

A notification's `email` block has the following structure:

* `attach_log`: (Optional) A boolean to attach log to email or not. Defaults to false.

* `recipients`: (Required) A list of recipients to receive email.

* `subject`: (Optional) Name of email subject.

A notification's `plugin` block has the following structure:

* `type` - (Required) The name of the plugin to use.

* `config` - (Required) Map of arbitrary configuration properties for the selected plugin.

## Attributes Reference

The following attribute is exported:

* `id` - A unique identifier for the job.

## Import

Rundeck job can be imported using the project and job uuid, e.g.

```
$ terraform import rundeck_job.my_job project_name/JOB-UUID
```

It is also possible to use `import` blocks to generate job config from existing jobs.  [See Hashi Docs here](https://developer.hashicorp.com/terraform/language/import/generating-configuration)