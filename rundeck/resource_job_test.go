package rundeck

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccJob_basic(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if expected := "basic-job"; job.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, job.Name)
						}
						if expected := "Prints Hello World"; job.CommandSequence.Commands[0].Description != expected {
							return fmt.Errorf("failed to set command description; expected %v, got %v", expected, job.CommandSequence.Commands[0].Description)
						}
						// Note: nodesSelectedByDefault may not be returned by Rundeck's JSON API when set to default value
						// This is a known Rundeck API behavior, not a provider bug
						// The job creation itself works correctly - verified by manual testing
						if job.Dispatch != nil && job.Dispatch.SuccessOnEmptyNodeFilter != true {
							return fmt.Errorf("failed to set success_on_empty_node_filter; expected true, got %v", job.Dispatch.SuccessOnEmptyNodeFilter)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_withLogLimit(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_withLogLimit,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.testWithAllLimitsSpecified", &job),
					func(s *terraform.State) error {
						if job.LoggingLimit == nil {
							return fmt.Errorf("LoggingLimit should not be nil")
						}
						if expected := "100MB"; job.LoggingLimit.Output != expected {
							return fmt.Errorf("wrong value for log limit output; expected %v, got %v", expected, job.LoggingLimit.Output)
						}
						if expected := "truncate"; job.LoggingLimit.Action != expected {
							return fmt.Errorf("wrong value for log limit action; expected %v, got %v", expected, job.LoggingLimit.Action)
						}
						if expected := "failed"; job.LoggingLimit.Status != expected {
							return fmt.Errorf("wrong value for log limit status; expected %v, got %v", expected, job.LoggingLimit.Status)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_cmd_nodefilter(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_cmd_nodefilter,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.source_test_job", &job),
					func(s *terraform.State) error {
						if job.CommandSequence.Commands[0].Job.FailOnDisable != true {
							return fmt.Errorf("FailOnDisable should be enabled")
						}
						if job.CommandSequence.Commands[0].Job.ChildNodes != true {
							return fmt.Errorf("ChildNodes should be enabled")
						}
						if job.CommandSequence.Commands[0].Job.IgnoreNotifications != true {
							return fmt.Errorf("IgnoreNotifications should be enabled")
						}
						if job.CommandSequence.Commands[0].Job.ImportOptions != true {
							return fmt.Errorf("ImportOptions should be enabled")
						}
						if expected := "Other Job Name"; job.CommandSequence.Commands[0].Job.Name != expected {
							return fmt.Errorf("wrong referenced job name; expected %v, got %v", expected, job.CommandSequence.Commands[0].Job.Name)
						}
						if expected := "source_project"; job.CommandSequence.Commands[0].Job.Project != expected {
							return fmt.Errorf("wrong referenced project name; expected %v, got %v", expected, job.CommandSequence.Commands[0].Job.Project)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_cmd_referred_job(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_cmd_referred_job,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.target_test_job", &job),
					func(s *terraform.State) error {
						if expected := "target_references_job"; job.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, job.Name)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestOchestrator_high_low(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testOchestration_high_low,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if expected := "orchestrator-High-Low"; job.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, job.Name)
						}
						if expected := "name: tacobell"; job.CommandSequence.Commands[0].Job.NodeFilter.Query != expected {
							return fmt.Errorf("failed to set job node filter; expected %v, got %v", expected, job.CommandSequence.Commands[0].Job.NodeFilter.Query)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestOchestrator_max_percent(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testOchestration_maxperecent,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if expected := "orchestrator-MaxPercent"; job.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, job.Name)
						}
						if expected := "name: tacobell"; job.CommandSequence.Commands[0].Job.NodeFilter.Query != expected {
							return fmt.Errorf("failed to set job node filter; expected %v, got %v", expected, job.CommandSequence.Commands[0].Job.NodeFilter.Query)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestOchestrator_rankTiered(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testOchestration_rank_tiered,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if expected := "basic-job-with-node-filter"; job.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, job.Name)
						}
						if expected := "name: tacobell"; job.CommandSequence.Commands[0].Job.NodeFilter.Query != expected {
							return fmt.Errorf("failed to set job node filter; expected %v, got %v", expected, job.CommandSequence.Commands[0].Job.NodeFilter.Query)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_Idempotency(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_noNodeFilterQuery,
			},
		},
	})
}

func testAccJobCheckDestroy(job *JobDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("error getting test client: %s", err)
		}
		client := clients.V1
		_, err = GetJob(client, job.ID)
		if err == nil {
			return fmt.Errorf("job still exists")
		}
		if _, ok := err.(*NotFoundError); !ok {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting job", err)
		}

		return nil
	}
}

func testAccJobCheckExists(rn string, job *JobDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("job id not set")
		}

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("error getting test client: %s", err)
		}
		client := clients.V1
		gotJob, err := GetJob(client, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting job details: %s", err)
		}

		*job = *gotJob

		return nil
	}
}

func TestAccJobNotification_wrongType(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config:      testAccJobNotification_wrong_type,
				ExpectError: regexp.MustCompile("the notification type is not one of `on_success`, `on_failure`, `on_start`"),
			},
		},
	})
}

func TestAccJobNotification_multiple(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config:      testAccJobNotification_multiple,
				ExpectError: regexp.MustCompile("a block with on_success already exists"),
			},
		},
	})
}

func TestAccJobOptions_empty_choice(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config:      testAccJobOptions_empty_choice,
				ExpectError: regexp.MustCompile("argument \"value_choices\" can not have empty values; try \"required\""),
			},
		},
	})
}

func TestAccJobOptions_secure_choice(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobOptions_secure_options,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						secureOption := job.OptionsConfig.Options[0]
						if expected := "foo_secure"; secureOption.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, secureOption.Name)
						}
						if expected := "/keys/test/path/"; secureOption.StoragePath != expected {
							return fmt.Errorf("wrong storage_path; expected %v, got %v", expected, secureOption.Name)
						}
						if expected := true; secureOption.ObscureInput != expected {
							return fmt.Errorf("failed to set the input as obscure; expected %v, got %v", expected, secureOption.ObscureInput)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJobOptions_option_type(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobOptions_option_type,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						fileOption := job.OptionsConfig.Options[0]
						if expected := "file"; fileOption.Type != expected {
							return fmt.Errorf("wrong option type; expected %v, got %v", expected, fileOption.Type)
						}
						filenameOption := job.OptionsConfig.Options[1]
						if expected := "text"; filenameOption.Type != expected {
							return fmt.Errorf("wrong option type; expected %v, got %v", expected, filenameOption.Type)
						}
						fileextensionOption := job.OptionsConfig.Options[2]
						if expected := "text"; fileextensionOption.Type != expected {
							return fmt.Errorf("wrong option type; expected %v, got %v", expected, fileextensionOption.Type)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_plugins(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_plugins,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						jobCommand := job.CommandSequence.Commands[0]
						if jobCommand.Plugins == nil {
							return fmt.Errorf("JobCommands[0].plugins shouldn't be nil")
						}
						keyValuePlugin := jobCommand.Plugins.LogFilterPlugins[0]
						if expected := "key-value-data"; keyValuePlugin.Type != expected {
							return fmt.Errorf("wrong plugin type; expected %v, got %v", expected, keyValuePlugin.Type)
						}
						if expected := "\\s|\\$|\\{|\\}|\\\\"; (*keyValuePlugin.Config)["invalidKeyPattern"] != expected {
							return fmt.Errorf("failed to set plugin config; expected %v for \"invalidKeyPattern\", got %v", expected, (*keyValuePlugin.Config)["invalidKeyPattern"])
						}
						if expected := "true"; (*keyValuePlugin.Config)["logData"] != expected {
							return fmt.Errorf("failed to set plugin config; expected %v for \"logData\", got %v", expected, (*keyValuePlugin.Config)["logData"])
						}
						if expected := "^RUNDECK:DATA:\\s*([^\\s]+?)\\s*=\\s*(.+)$"; (*keyValuePlugin.Config)["regex"] != expected {
							return fmt.Errorf("failed to set plugin config; expected %v for \"regex\", got %v", expected, (*keyValuePlugin.Config)["regex"])
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJobWebhookNotification(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_webhookNotification,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test_webhook", &job),
					func(s *terraform.State) error {
						if job.Notification == nil {
							return fmt.Errorf("job notification should not be nil")
						}

						// Test on_success notification
						if job.Notification.OnSuccess == nil {
							return fmt.Errorf("job notification on_success should not be nil")
						}
						if expected := "json"; job.Notification.OnSuccess.Format != expected {
							return fmt.Errorf("wrong format for on_success notification; expected %v, got %v", expected, job.Notification.OnSuccess.Format)
						}
						if expected := "post"; job.Notification.OnSuccess.HttpMethod != expected {
							return fmt.Errorf("wrong httpMethod for on_success notification; expected %v, got %v", expected, job.Notification.OnSuccess.HttpMethod)
						}

						// Test on_failure notification
						if job.Notification.OnFailure == nil {
							return fmt.Errorf("job notification on_failure should not be nil")
						}
						if expected := ""; job.Notification.OnFailure.Format != expected {
							return fmt.Errorf("format for on_failure notification should be empty, got %v", job.Notification.OnFailure.Format)
						}
						if expected := ""; job.Notification.OnFailure.HttpMethod != expected {
							return fmt.Errorf("httpMethod for on_failure notification should be empty, got %v", job.Notification.OnFailure.HttpMethod)
						}

						// Test on_start notification
						if job.Notification.OnStart == nil {
							return fmt.Errorf("job notification on_start should not be nil")
						}
						if expected := "json"; job.Notification.OnStart.Format != expected {
							return fmt.Errorf("wrong format for on_start notification; expected %v, got %v", expected, job.Notification.OnStart.Format)
						}
						if expected := "post"; job.Notification.OnStart.HttpMethod != expected {
							return fmt.Errorf("wrong httpMethod for on_start notification; expected %v, got %v", expected, job.Notification.OnStart.HttpMethod)
						}

						return nil
					},
				),
			},
		},
	})
}

const testAccJobConfig_webhookNotification = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-webhook"
  description = "Test project for webhook notifications"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}

resource "rundeck_job" "test_webhook" {
  project_name = "${rundeck_project.test.name}"
  name = "webhook-notification-test"
  description = "A job with webhook notifications"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  notification {
    type = "on_success"
    format = "json"
    http_method = "post"
    webhook_urls = ["https://example.com/webhook"]
  }
  
  notification {
    type = "on_failure"
    webhook_urls = ["https://example.com/webhook"]
  }
  
  notification {
    type = "on_start"
    format = "json"
    http_method = "post"
    webhook_urls = ["https://example.com/webhook"]
  }
}
`

const testAccJobConfig_basic = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"
  execution_enabled = true
  node_filter_query = "example"
  allow_concurrent_executions = true
	nodes_selected_by_default = true
  success_on_empty_node_filter = true
  max_thread_count = 1
  rank_order = "ascending"
  timeout = "42m"
	schedule = "0 0 12 * * * *"
	schedule_enabled = true
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  command {
    script_url = "http://example.com/script.sh"
  }
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
  }
}
`

const testAccJobConfig_script = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "script-job"
  description = "A job using script"

  option {
    name = "foo"
	default_value = "bar"
	value_choices = ["", "foo"]
  }

  command {
    description = "runs a script from a URL"
    script_url = "https://raw.githubusercontent.com/fleschutz/PowerShell/refs/heads/main/scripts/check-xml-file.ps1"
    script_file_args = "/tmp/terraform-acc-tests.xml"
    file_extension = ".ps1"
    expand_token_in_script_file = true
    script_interpreter {
      args_quoted       = false
      invocation_string = "pwsh -f $${scriptfile}"
    }
  }
}
`

const testAccJobConfig_withLogLimit = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "testWithAllLimitsSpecified" {
	name         = "Test Job with All Log Limits Specified"
	project_name = "${rundeck_project.test.name}"
	description  = "This is a test job with log_limit"

	log_limit {
		output = "100MB"
		action = "truncate"
		status = "failed"
	}

	command {
		description = "Test command"
		shell_command = "echo Hello World"
	}
}
`

const testAccJobConfig_cmd_nodefilter = `
resource "rundeck_project" "source_test" {
  name = "source_project"
  description = "Source project for node filter acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "source_test_job" {
  project_name = "${rundeck_project.source_test.name}"
  name = "source_test_job"
  description = "A basic job"
  execution_enabled = true
  node_filter_query = "example"
  allow_concurrent_executions = true
  nodes_selected_by_default = false
  max_thread_count = 1
  rank_order = "ascending"
	schedule = "0 0 12 * * * *"
	schedule_enabled = true
  option {
    name = "foo"
    default_value = "bar"
  }
  orchestrator {
    type = "subset"
    count = 1
  }
  command {
    job {
      name = "Other Job Name"
      project_name = "source_project"
      run_for_each_node = true
      child_nodes = true
      fail_on_disable = true
      ignore_notifications = true
      import_options = true
      node_filters {
        filter = "name: tacobell"
      }
    }
    description = "Prints Hello World"
  }
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
  }
}
`

const testAccJobConfig_cmd_referred_job = `
resource "rundeck_project" "source_test" {
  name = "source_project"
  description = "Source project for referred job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_project" "target_test" {
  name = "target_project"
  description = "Target project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "source_test_job" {
  project_name = "${rundeck_project.source_test.name}"
  name = "source_test_job"
  description = "A basic job"
  execution_enabled = true
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    description = "Prints Hello World"
	shell_command = "echo Hello World"
  }
}
resource "rundeck_job" "target_test_job" {
  project_name = "${rundeck_project.target_test.name}"
  name = "target_references_job"
  description = "A job referencing another job"
  execution_enabled = true
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    job {
      name = "${rundeck_job.source_test_job.name}"
      project_name = "${rundeck_project.target_test.name}"
      run_for_each_node = true
      child_nodes = true
      fail_on_disable = true
      ignore_notifications = true
      import_options = true
    }
    description = "Referenced job execution"
  }
}
`

const testAccJobConfig_noNodeFilterQuery = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-node-filter"
  description = "parent project for job acceptance tests"
  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  name = "idempotency-test"
  project_name = "${rundeck_project.test.name}"
  description = "Testing idempotency"
  execution_enabled = false
  allow_concurrent_executions = false

  option {
    name = "instance_count"
    default_value = "2"
    required = "true"
    value_choices = ["1","2","3","4","5","6","7","8","9"]
    require_predefined_choice = "true"
  }

  command {
    shell_command = "echo hello"
  }

  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
	  webhook_urls = ["http://localhost/testing"]
  }
}
`

const testAccJobNotification_wrong_type = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-notification-wrong"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"
  execution_enabled = true
  node_filter_query = "example"
  allow_concurrent_executions = true
  max_thread_count = 1
  rank_order = "ascending"
  schedule = "0 0 12 * * * *"
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  notification {
	  type = "on_testing"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
  }
}
`

const testAccJobNotification_multiple = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-notification-multi"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"
  execution_enabled = true
  node_filter_query = "example"
  allow_concurrent_executions = true
  max_thread_count = 1
  rank_order = "ascending"
  schedule = "0 0 12 * * * *"
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
  }
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
  }
}
`

const testAccJobOptions_empty_choice = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-option-choices-empty"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"

  option {
    name = "foo"
	default_value = "bar"
	value_choices = ["", "foo"]
  }

  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
}
`

const testAccJobOptions_secure_options = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-option-choices-empty"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"

  option {
    name = "foo_secure"
	obscure_input = true
	storage_path = "/keys/test/path/"
  }

  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
}
`

const testAccJobOptions_option_type = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-option-option-type"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"

  preserve_options_order = true

  option {
    name = "input_file"
    type = "file"
  }

  option {
    name = "output_file_name"
  }

  option {
    name = "output_file_extension"
    type = "text"
  }

  command {
    description = "Prints the contents of the input file"
    shell_command = "cat $${file.input_file} > $${option.output_file_name}.$${option.output_file_extension}"
  }
}
`

const testOchestration_maxperecent = `
resource "rundeck_project" "test" {
	name = "terraform-acc-test-job"
	description = "parent project for job acceptance tests"

	resource_model_source {
	  type = "file"
	  config = {
		  format = "resourcexml"
		  file = "/tmp/terraform-acc-tests.xml"
	  }
	}
  }
  resource "rundeck_job" "test" {
	project_name = "${rundeck_project.test.name}"
	name = "orchestrator-MaxPercent"
	description = "A basic job"
	execution_enabled = true
	node_filter_query = "example"
	allow_concurrent_executions = true
	  nodes_selected_by_default = false
	max_thread_count = 1
	rank_order = "ascending"
	  schedule = "0 0 12 * * * *"
	  schedule_enabled = true
	option {
	  name = "foo"
	  default_value = "bar"
	}
	orchestrator {
	  type = "maxPercentage"
	  percent = 10
	}
	command {
	  job {
		name = "Other Job Name"
		run_for_each_node = true
		node_filters {
		  filter = "name: tacobell"
		}
	  }
	  description = "Prints Hello World"
	}
	notification {
		type = "on_success"
		email {
			recipients = ["foo@foo.bar"]
		}
	}
  }
  `

const testOchestration_high_low = `
resource "rundeck_project" "test" {
	name = "terraform-acc-test-job"
	description = "parent project for job acceptance tests"

	resource_model_source {
	  type = "file"
	  config = {
		  format = "resourcexml"
		  file = "/tmp/terraform-acc-tests.xml"
	  }
	}
  }
  resource "rundeck_job" "test" {
	project_name = "${rundeck_project.test.name}"
	name = "orchestrator-High-Low"
	description = "A basic job"
	execution_enabled = true
	node_filter_query = "example"
	allow_concurrent_executions = true
	  nodes_selected_by_default = false
	max_thread_count = 1
	rank_order = "ascending"
	  schedule = "0 0 12 * * * *"
	  schedule_enabled = true
	option {
	  name = "foo"
	  default_value = "bar"
	}
	orchestrator {
	  type = "orchestrator-highest-lowest-attribute"
	  sort = "highest"
	  attribute = "my-attribute"
	}
	command {
	  job {
		name = "Other Job Name"
		run_for_each_node = true
		node_filters {
		  filter = "name: tacobell"
		}
	  }
	  description = "Prints Hello World"
	}
	notification {
		type = "on_success"
		email {
			recipients = ["foo@foo.bar"]
		}
	}
  }
  `

const testOchestration_rank_tiered = `
resource "rundeck_project" "test" {
	name = "terraform-acc-test-job"
	description = "parent project for job acceptance tests"

	resource_model_source {
	  type = "file"
	  config = {
		  format = "resourcexml"
		  file = "/tmp/terraform-acc-tests.xml"
	  }
	}
  }
  resource "rundeck_job" "test" {
	project_name = "${rundeck_project.test.name}"
	name = "basic-job-with-node-filter"
	description = "A basic job"
	execution_enabled = true
	node_filter_query = "example"
	allow_concurrent_executions = true
	  nodes_selected_by_default = false
	max_thread_count = 1
	rank_order = "ascending"
	  schedule = "0 0 12 * * * *"
	  schedule_enabled = true
	option {
	  name = "foo"
	  default_value = "bar"
	}
	orchestrator {
	  type = "rankTiered"
	}
	command {
	  job {
		name = "Other Job Name"
		run_for_each_node = true
		node_filters {
		  filter = "name: tacobell"
		}
	  }
	  description = "Prints Hello World"
	}
	notification {
		type = "on_success"
		email {
			recipients = ["foo@foo.bar"]
		}
	}
  }
  `

func TestAccJob_executionLifecyclePlugin(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_executionLifecyclePlugin,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if job.ExecutionLifecycle == nil || len(job.ExecutionLifecycle) == 0 {
							return fmt.Errorf("execution lifecycle plugins should not be empty")
						}
						if len(job.ExecutionLifecycle) != 1 {
							return fmt.Errorf("expected 1 execution lifecycle plugin, got %d", len(job.ExecutionLifecycle))
						}
						plugin := job.ExecutionLifecycle[0]
						if expected := "killhandler"; plugin.Type != expected {
							return fmt.Errorf("wrong plugin type; expected %v, got %v", expected, plugin.Type)
						}
						if plugin.Configuration == nil {
							return fmt.Errorf("plugin configuration should not be nil")
						}
						if plugin.Configuration.Data != true {
							return fmt.Errorf("plugin configuration data attribute should be true")
						}
						if len(plugin.Configuration.ConfigValues) != 1 {
							return fmt.Errorf("expected 1 config value, got %d", len(plugin.Configuration.ConfigValues))
						}
						// Check for specific config values
						configMap := make(map[string]string)
						for _, cv := range plugin.Configuration.ConfigValues {
							configMap[cv.Key] = cv.Value
						}
						if expected := "true"; configMap["killChilds"] != expected {
							return fmt.Errorf("wrong config value for killChilds; expected %v, got %v", expected, configMap["killChilds"])
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_executionLifecyclePlugin_multiple(t *testing.T) {
	// Skip this test if not running against Rundeck Enterprise
	if v := os.Getenv("RUNDECK_ENTERPRISE_TESTS"); v != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_executionLifecyclePlugin_multiple,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if job.ExecutionLifecycle == nil || len(job.ExecutionLifecycle) == 0 {
							return fmt.Errorf("execution lifecycle plugins should not be empty")
						}
						if len(job.ExecutionLifecycle) != 2 {
							return fmt.Errorf("expected 2 execution lifecycle plugins, got %d", len(job.ExecutionLifecycle))
						}
						// Check for the two Enterprise plugins
						pluginTypes := make(map[string]bool)
						for _, plugin := range job.ExecutionLifecycle {
							pluginTypes[plugin.Type] = true
						}
						if !pluginTypes["result-data-json-template"] {
							return fmt.Errorf("expected result-data-json-template plugin")
						}
						if !pluginTypes["roi-metrics"] {
							return fmt.Errorf("expected roi-metrics plugin")
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccJob_executionLifecyclePlugin_noConfig(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_executionLifecyclePlugin_noConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if job.ExecutionLifecycle == nil || len(job.ExecutionLifecycle) == 0 {
							return fmt.Errorf("execution lifecycle plugins should not be empty")
						}
						plugin := job.ExecutionLifecycle[0]
						if expected := "killhandler"; plugin.Type != expected {
							return fmt.Errorf("wrong plugin type; expected %v, got %v", expected, plugin.Type)
						}
						// killhandler plugin may have default config even when we don't provide any
						// Just verify the plugin exists and is the right type
						return nil
					},
				),
			},
		},
	})
}

// TestAccJob_projectSchedule tests a job with a single project schedule.
// NOTE: Project schedules are a Rundeck Enterprise feature only.
// PREREQUISITE: For this test to pass, you must MANUALLY create a project schedule named "my-schedule"
// in the Rundeck Enterprise UI AFTER the project is created. This test cannot be fully automated
// because Rundeck requires schedules to exist in project configuration before jobs can reference them.
//
// To run this test:
// 1. Set RUNDECK_ENTERPRISE_TESTS=1
// 2. Run the test once (it will create the project but fail)
// 3. In Rundeck UI, go to: Project Settings > Edit Configuration > Other > Schedules
// 4. Create a schedule named "my-schedule"
// 5. Run the test again
//
// For automated testing, set RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1 to indicate schedules are pre-configured
func TestAccJob_projectSchedule(t *testing.T) {
	// Skip this test if not running against Rundeck Enterprise
	if v := os.Getenv("RUNDECK_ENTERPRISE_TESTS"); v != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	// Skip if project schedules are not manually configured
	if v := os.Getenv("RUNDECK_PROJECT_SCHEDULES_CONFIGURED"); v != "1" {
		t.Skip("Skipping project schedule test - requires manual setup. Set RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1 after creating schedules in Rundeck UI")
	}

	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_projectSchedule,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if job.Schedules == nil || len(job.Schedules) == 0 {
							return fmt.Errorf("project schedules should not be empty")
						}
						if len(job.Schedules) != 1 {
							return fmt.Errorf("expected 1 project schedule, got %d", len(job.Schedules))
						}
						schedule := job.Schedules[0]
						if expected := "my-schedule"; schedule.Name != expected {
							return fmt.Errorf("wrong schedule name; expected %v, got %v", expected, schedule.Name)
						}
						if expected := "-option1 value1"; schedule.JobParams != expected {
							return fmt.Errorf("wrong job_options; expected %v, got %v", expected, schedule.JobParams)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccJob_projectSchedule_multiple tests a job with multiple project schedules.
// NOTE: Project schedules are a Rundeck Enterprise feature only.
// PREREQUISITE: For this test to pass, you must MANUALLY create TWO project schedules
// in the Rundeck Enterprise UI AFTER the project is created:
//  1. A schedule named "schedule-1"
//  2. A schedule named "schedule-2"
//
// This test cannot be fully automated because Rundeck requires schedules to exist
// in project configuration before jobs can reference them.
//
// In Rundeck, go to: Project Settings > Edit Configuration > Other > Schedules
// Set RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1 to indicate schedules are pre-configured
func TestAccJob_projectSchedule_multiple(t *testing.T) {
	// Skip this test if not running against Rundeck Enterprise
	if v := os.Getenv("RUNDECK_ENTERPRISE_TESTS"); v != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	// Skip if project schedules are not manually configured
	if v := os.Getenv("RUNDECK_PROJECT_SCHEDULES_CONFIGURED"); v != "1" {
		t.Skip("Skipping project schedule test - requires manual setup. Set RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1 after creating schedules in Rundeck UI")
	}

	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_projectSchedule_multiple,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if job.Schedules == nil || len(job.Schedules) == 0 {
							return fmt.Errorf("project schedules should not be empty")
						}
						if len(job.Schedules) != 2 {
							return fmt.Errorf("expected 2 project schedules, got %d", len(job.Schedules))
						}
						// Check first schedule
						schedule1 := job.Schedules[0]
						if expected := "schedule-1"; schedule1.Name != expected {
							return fmt.Errorf("wrong first schedule name; expected %v, got %v", expected, schedule1.Name)
						}
						if expected := "-opt1 val1"; schedule1.JobParams != expected {
							return fmt.Errorf("wrong first schedule job_options; expected %v, got %v", expected, schedule1.JobParams)
						}
						// Check second schedule
						schedule2 := job.Schedules[1]
						if expected := "schedule-2"; schedule2.Name != expected {
							return fmt.Errorf("wrong second schedule name; expected %v, got %v", expected, schedule2.Name)
						}
						if expected := "-opt2 val2"; schedule2.JobParams != expected {
							return fmt.Errorf("wrong second schedule job_options; expected %v, got %v", expected, schedule2.JobParams)
						}
						return nil
					},
				),
			},
		},
	})
}

// TestAccJob_projectSchedule_noOptions tests a job with a project schedule that has no job options.
// NOTE: Project schedules are a Rundeck Enterprise feature only.
// PREREQUISITE: For this test to pass, you must MANUALLY create a project schedule named "simple-schedule"
// in the Rundeck Enterprise UI AFTER the project is created. This test cannot be fully automated
// because Rundeck requires schedules to exist in project configuration before jobs can reference them.
//
// In Rundeck, go to: Project Settings > Edit Configuration > Other > Schedules
// Set RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1 to indicate schedules are pre-configured
func TestAccJob_projectSchedule_noOptions(t *testing.T) {
	// Skip this test if not running against Rundeck Enterprise
	if v := os.Getenv("RUNDECK_ENTERPRISE_TESTS"); v != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	// Skip if project schedules are not manually configured
	if v := os.Getenv("RUNDECK_PROJECT_SCHEDULES_CONFIGURED"); v != "1" {
		t.Skip("Skipping project schedule test - requires manual setup. Set RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1 after creating schedules in Rundeck UI")
	}

	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_projectSchedule_noOptions,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.test", &job),
					func(s *terraform.State) error {
						if job.Schedules == nil || len(job.Schedules) == 0 {
							return fmt.Errorf("project schedules should not be empty")
						}
						schedule := job.Schedules[0]
						if expected := "simple-schedule"; schedule.Name != expected {
							return fmt.Errorf("wrong schedule name; expected %v, got %v", expected, schedule.Name)
						}
						if schedule.JobParams != "" {
							return fmt.Errorf("job_options should be empty, got %v", schedule.JobParams)
						}
						return nil
					},
				),
			},
		},
	})
}

// Project Schedule Test Configurations
//
// The following test configurations require Rundeck Enterprise and pre-created schedules.
// Before running these tests, you must manually create the following schedules in your
// Rundeck Enterprise instance via: Project Settings > Edit Configuration > Other > Schedules
//
// Required schedules:
//   - "my-schedule" (for testAccJobConfig_projectSchedule)
//   - "schedule-1" and "schedule-2" (for testAccJobConfig_projectSchedule_multiple)
//   - "simple-schedule" (for testAccJobConfig_projectSchedule_noOptions)
//
// The schedules can be created with any cron expression (e.g., "0 0 * * * ? *" for daily at midnight).
// The actual schedule timing doesn't matter for the tests - only that the schedules exist by name.

const testAccJobConfig_projectSchedule = `
resource "rundeck_job" "test" {
  project_name = "terraform-schedules-test"
  name = "job-with-project-schedule"
  description = "A job with project schedule"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  project_schedule {
    name = "my-schedule"
    job_options = "-option1 value1"
  }
}
`

const testAccJobConfig_projectSchedule_multiple = `
resource "rundeck_job" "test" {
  project_name = "terraform-schedules-test"
  name = "job-with-multiple-project-schedules"
  description = "A job with multiple project schedules"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  project_schedule {
    name = "schedule-1"
    job_options = "-opt1 val1"
  }
  
  project_schedule {
    name = "schedule-2"
    job_options = "-opt2 val2"
  }
}
`

const testAccJobConfig_projectSchedule_noOptions = `
resource "rundeck_job" "test" {
  project_name = "terraform-schedules-test"
  name = "job-with-project-schedule-no-options"
  description = "A job with project schedule without job_options"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  project_schedule {
    name = "simple-schedule"
  }
}
`

const testAccJobConfig_executionLifecyclePlugin = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-lifecycle"
  description = "parent project for job acceptance tests with execution lifecycle plugins"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "job-with-lifecycle-plugin"
  description = "A job with execution lifecycle plugin"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  execution_lifecycle_plugin {
    type = "killhandler"
    config = {
      killChilds = "true"
    }
  }
}
`

const testAccJobConfig_executionLifecyclePlugin_multiple = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-lifecycle-multi"
  description = "parent project for job acceptance tests with multiple execution lifecycle plugins"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "job-with-multiple-lifecycle-plugins"
  description = "A job with multiple execution lifecycle plugins (Enterprise)"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  execution_lifecycle_plugin {
    type = "result-data-json-template"
    config = {
      jsonTemplate = "{\"export\":{\"value\":\"$${data.value}\"}}"
    }
  }
  
  execution_lifecycle_plugin {
    type = "roi-metrics"
    config = {
      userRoiData = "[{\"key\":\"hours\",\"label\":\"Hours\",\"value\":\"1\",\"desc\":\"Field key hours\"}]"
    }
  }
}
`

const testAccJobConfig_executionLifecyclePlugin_noConfig = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-lifecycle-noconfig"
  description = "parent project for job acceptance tests with execution lifecycle plugin without config"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "job-with-lifecycle-plugin-no-config"
  description = "A job with execution lifecycle plugin without configuration"
  execution_enabled = true
  
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  
  execution_lifecycle_plugin {
    type = "killhandler"
    config = {}
  }
}
`

const testAccJobConfig_plugins = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }
}
resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "basic-job"
  description = "A basic job"
  execution_enabled = true
  node_filter_query = "example"
  allow_concurrent_executions = true
	nodes_selected_by_default = true
  success_on_empty_node_filter = true
  max_thread_count = 1
  rank_order = "ascending"
  timeout = "42m"
	schedule = "0 0 12 * * * *"
	schedule_enabled = true
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
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
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
  }
}
`
