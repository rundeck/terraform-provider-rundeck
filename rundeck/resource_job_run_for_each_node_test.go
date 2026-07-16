package rundeck

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccJob_cmd_referred_job_runForEachNode is a regression test for #256:
// run_for_each_node on a job reference was serialized to a field Rundeck
// ignores, so the referenced job was always treated as a workflow step. This
// exercises run_for_each_node and its node_step alias on both a plain command
// job reference and an error_handler job reference, and asserts the setting
// round-trips (no drift, no "inconsistent result after apply") on a second
// plan.
func TestAccJob_cmd_referred_job_runForEachNode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_cmd_referred_job_runForEachNode,
				Check: resource.ComposeTestCheckFunc(
					// command[0].job: run_for_each_node explicitly set true, node_step
					// must read back as true too (alias round-trip).
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.0.job.0.run_for_each_node", "true"),
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.0.job.0.node_step", "true"),
					// command[0].error_handler.job: same setting, must also work on
					// error handlers (this was the only path that worked pre-fix).
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.0.error_handler.0.job.0.run_for_each_node", "true"),
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.0.error_handler.0.job.0.node_step", "true"),
					// command[1].job: node_step alias set true, run_for_each_node must
					// read back as true too.
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.1.job.0.node_step", "true"),
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.1.job.0.run_for_each_node", "true"),
					// command[2].job: neither alias configured, defaults to false (a
					// workflow step, not a node step).
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.2.job.0.run_for_each_node", "false"),
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.2.job.0.node_step", "false"),
				),
			},
			{
				// Re-apply with the same config: this must produce an empty plan.
				// The original fix attempt was reverted because this step surfaced
				// "unknown value after apply" / drift for some job references.
				Config:   testAccJobConfig_cmd_referred_job_runForEachNode,
				PlanOnly: true,
			},
		},
	})
}

const testAccJobConfig_cmd_referred_job_runForEachNode = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-job-run-for-each-node"
  description = "Test project for run_for_each_node regression (#256)"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "target" {
  project_name       = rundeck_project.test.name
  name               = "target-job"
  description        = "Job to be referenced"
  execution_enabled  = true
  command {
    shell_command = "echo 'I am the target job'"
  }
}

resource "rundeck_job" "caller" {
  project_name      = rundeck_project.test.name
  name              = "caller-job"
  description       = "Job referencing another job with run_for_each_node/node_step (#256)"
  execution_enabled = true

  command {
    description = "call target via run_for_each_node"
    job {
      name              = rundeck_job.target.name
      project_name      = rundeck_project.test.name
      run_for_each_node = true
    }
    error_handler {
      job {
        name              = rundeck_job.target.name
        project_name      = rundeck_project.test.name
        run_for_each_node = true
      }
    }
  }

  command {
    description = "call target via node_step alias"
    job {
      name         = rundeck_job.target.name
      project_name = rundeck_project.test.name
      node_step    = true
    }
  }

  command {
    description = "call target with neither alias set"
    job {
      name         = rundeck_job.target.name
      project_name = rundeck_project.test.name
    }
  }
}
`
