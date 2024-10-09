package rundeck

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/rundeck/go-rundeck/rundeck"
)

func TestAccJob_basic(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
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
						if expected := true; job.NodesSelectedByDefault != expected {
							return fmt.Errorf("failed to set node selected by default; expected %v, got %v", expected, job.NodesSelectedByDefault)
						}
						if job.Dispatch.SuccessOnEmptyNodeFilter != true {
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
		Providers:    testAccProviders,
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_withLogLimit,
				Check: resource.ComposeTestCheckFunc(
					testAccJobCheckExists("rundeck_job.testWithAllLimitsSpecified", &job),
					func(s *terraform.State) error {
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
		Providers:    testAccProviders,
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
						if expected := "source_test_job"; job.CommandSequence.Commands[0].Job.Name != expected {
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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
		client := testAccProvider.Meta().(*rundeck.BaseClient)
		_, err := GetJob(client, job.ID)
		if err == nil {
			return fmt.Errorf("key still exists")
		}
		if _, ok := err.(*NotFoundError); !ok {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting key", err)
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

		client := testAccProvider.Meta().(*rundeck.BaseClient)
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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
		Providers:    testAccProviders,
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

func TestAccJob_plugins(t *testing.T) {
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
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
    script_url = "notarealurl.end"
  }
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
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
    type = "subset"
    count = 1
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
}
resource "rundeck_job" "target_test_job" {
  project_name = "${rundeck_project.target_test.name}"
  name = "target_references_job"
  description = "A job referencing another job"
  execution_enabled = true
  option {
    name = "foo"
    default_value = "bar"
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
