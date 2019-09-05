package rundeck

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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
				ExpectError: regexp.MustCompile("The notification type is not one of `on_success`, `on_failure`, `on_start`"),
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
				ExpectError: regexp.MustCompile("A block with on_success already exists"),
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
				ExpectError: regexp.MustCompile("Argument \"value_choices\" can not have empty values; try \"required\""),
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
  success_on_empty_node_filter = true
  max_thread_count = 1
  rank_order = "ascending"
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
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
	  }
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
  max_thread_count = 1
  rank_order = "ascending"
	schedule = "0 0 12 * * * *"
	schedule_enabled = true
  option {
    name = "foo"
    default_value = "bar"
  }
  command {
    job {
      name = "Other Job Name"
      run_for_each_node = true
      nodefilters = {
        filter: "name: tacobell"
      }
    }
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
  notification {
	  type = "on_success"
	  email {
		  recipients = ["foo@foo.bar"]
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
    value_choices = ["1,2,3,4,5,6,7,8,9"]
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
