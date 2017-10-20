package rundeck

import (
	"fmt"
	"testing"

	"github.com/apparentlymart/go-rundeck-api/rundeck"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccJob_basic(t *testing.T) {
	var job rundeck.JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			resource.TestStep{
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

						if expected := "echo Hello World"; job.CommandSequence.Commands[1].Script != expected {
							return fmt.Errorf("failed to set command inline_script; expected %v, got %v", expected, job.CommandSequence.Commands[1].Script)
						}

						if expected := "sudo"; job.CommandSequence.Commands[1].ScriptInterpreter.InvocationString != expected {
							return fmt.Errorf("failed to set command description; expected %v, got %v", expected, job.CommandSequence.Commands[1].ScriptInterpreter.InvocationString)
						}

						if expected := true; job.CommandSequence.Commands[1].ScriptInterpreter.ArgsQuoted != expected {
							return fmt.Errorf("failed to set command description; expected %v, got %v", expected, job.CommandSequence.Commands[1].ScriptInterpreter.ArgsQuoted)
						}

						return nil
					},
				),
			},
		},
	})
}

func testAccJobCheckDestroy(job *rundeck.JobDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*rundeck.Client)
		_, err := client.GetJob(job.ID)
		if err == nil {
			return fmt.Errorf("key still exists")
		}
		if _, ok := err.(*rundeck.NotFoundError); !ok {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting key", err)
		}

		return nil
	}
}

func testAccJobCheckExists(rn string, job *rundeck.JobDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("job id not set")
		}

		client := testAccProvider.Meta().(*rundeck.Client)
		gotJob, err := client.GetJob(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting job details: %s", err)
		}

		*job = *gotJob

		return nil
	}
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
  node_filter_query = "example"
  allow_concurrent_executions = 1
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
  command {
    description = "Prints Hello World"
    inline_script = "echo Hello World"
    inline_script_invocation_string = "sudo"
    inline_script_args_quoted = true
  }
}
`
