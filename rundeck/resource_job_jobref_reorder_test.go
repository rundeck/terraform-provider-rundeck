package rundeck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccJob_cmd_referred_job_reorder moves job references around a workflow
// and checks that each one keeps its own settings.
//
// A job reference's uuid, name, group_name, project_name, run_for_each_node
// and node_step are Optional + Computed. They used to carry UseStateForUnknown,
// which reads prior state at the same list index: once the commands moved, an
// attribute left out of the configuration was filled from whichever command
// used to sit at that position.
//
//   - The first reorder swaps a reference to a job of another project with a
//     reference that names no project. The latter used to inherit the former's
//     project_name and node_step, and Rundeck then looked for the job in the
//     wrong project at run time.
//   - The second reorder moves references to positions that held a plain
//     command or a reference to another job. Attributes were planned as null
//     where there was no prior reference, read back as false, and the apply
//     failed with "Provider produced inconsistent result after apply".
//
// The reference by uuid covers the other half: Rundeck resolves it, and what
// it returns must neither show up as a diff on the next plan nor follow the
// reference that takes its place.
//
// Two commands carry an error handler that is itself a job reference, one with
// its node-step flag left out and one with it set to false, so that the
// handlers move along and land on each other's positions too.
func TestAccJob_cmd_referred_job_reorder(t *testing.T) {
	const (
		local  = "local-target"
		remote = "remote-target"
	)

	// Each element is one command of the caller job, in workflow order.
	exec := fmt.Sprintf(`
  command {
    shell_command = "echo between"
    error_handler {
      job {
        name = %q
      }
    }
  }`, local)
	localRef := fmt.Sprintf(`
  command {
    job {
      name = %q
    }
    error_handler {
      job {
        name         = %q
        project_name = rundeck_project.remote.name
        node_step    = false
      }
    }
  }`, local, remote)
	remoteRef := fmt.Sprintf(`
  command {
    job {
      name         = %q
      project_name = rundeck_project.remote.name
      node_step    = true
    }
  }`, remote)
	uuidRef := `
  command {
    job {
      uuid = rundeck_job.local.id
    }
  }`

	// checkRefs asserts what each job reference holds, given the index in the
	// workflow of the plain command, of the reference by name, of the one to
	// another project, and of the one by uuid.
	checkRefs := func(execIdx, localIdx, remoteIdx, uuidIdx int) resource.TestCheckFunc {
		eh := fmt.Sprintf("command.%d.error_handler.0.job.0.", execIdx)
		lh := fmt.Sprintf("command.%d.error_handler.0.job.0.", localIdx)
		l := fmt.Sprintf("command.%d.job.0.", localIdx)
		r := fmt.Sprintf("command.%d.job.0.", remoteIdx)
		u := fmt.Sprintf("command.%d.job.0.", uuidIdx)
		return resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("rundeck_job.caller", l+"name", local),
			resource.TestCheckNoResourceAttr("rundeck_job.caller", l+"uuid"),
			resource.TestCheckNoResourceAttr("rundeck_job.caller", l+"project_name"),
			resource.TestCheckResourceAttr("rundeck_job.caller", l+"node_step", "false"),
			resource.TestCheckResourceAttr("rundeck_job.caller", l+"run_for_each_node", "false"),
			resource.TestCheckResourceAttr("rundeck_job.caller", r+"name", remote),
			resource.TestCheckResourceAttr("rundeck_job.caller", r+"project_name", "terraform-acc-test-jobref-reorder-remote"),
			resource.TestCheckResourceAttr("rundeck_job.caller", r+"node_step", "true"),
			resource.TestCheckResourceAttr("rundeck_job.caller", r+"run_for_each_node", "true"),
			resource.TestCheckResourceAttrPair("rundeck_job.caller", u+"uuid", "rundeck_job.local", "id"),
			resource.TestCheckResourceAttr("rundeck_job.caller", u+"node_step", "false"),
			// A handler's reference is a node step unless it says otherwise.
			resource.TestCheckResourceAttr("rundeck_job.caller", eh+"name", local),
			resource.TestCheckNoResourceAttr("rundeck_job.caller", eh+"project_name"),
			resource.TestCheckResourceAttr("rundeck_job.caller", eh+"node_step", "true"),
			resource.TestCheckResourceAttr("rundeck_job.caller", eh+"run_for_each_node", "true"),
			resource.TestCheckResourceAttr("rundeck_job.caller", lh+"name", remote),
			resource.TestCheckResourceAttr("rundeck_job.caller", lh+"project_name", "terraform-acc-test-jobref-reorder-remote"),
			resource.TestCheckResourceAttr("rundeck_job.caller", lh+"node_step", "false"),
			resource.TestCheckResourceAttr("rundeck_job.caller", lh+"run_for_each_node", "false"),
		)
	}

	initial := testAccJobConfig_cmd_referred_job_reorder(exec, localRef, remoteRef, uuidRef)
	swapped := testAccJobConfig_cmd_referred_job_reorder(exec, remoteRef, localRef, uuidRef)
	moved := testAccJobConfig_cmd_referred_job_reorder(uuidRef, localRef, exec, remoteRef)
	// Same workflow, the handler's node_step = false taken out: it goes back
	// to a handler's default instead of keeping the value it had.
	unset := testAccJobConfig_cmd_referred_job_reorder(uuidRef, strings.Replace(localRef, "node_step    = false", "", 1), exec, remoteRef)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{Config: initial, Check: checkRefs(0, 1, 2, 3)},
			{Config: initial, PlanOnly: true},
			{Config: swapped, Check: checkRefs(0, 2, 1, 3)},
			{Config: swapped, PlanOnly: true},
			{Config: moved, Check: checkRefs(2, 1, 3, 0)},
			{Config: moved, PlanOnly: true},
			{
				Config: unset,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.1.error_handler.0.job.0.node_step", "true"),
					resource.TestCheckResourceAttr("rundeck_job.caller", "command.1.error_handler.0.job.0.run_for_each_node", "true"),
				),
			},
			{Config: unset, PlanOnly: true},
		},
	})
}

func testAccJobConfig_cmd_referred_job_reorder(commands ...string) string {
	caller := ""
	for _, c := range commands {
		caller += c
	}

	return fmt.Sprintf(`
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-jobref-reorder"
  description = "Project of the job whose references are reordered"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_project" "remote" {
  name        = "terraform-acc-test-jobref-reorder-remote"
  description = "Project holding a job referenced from another project"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "local" {
  project_name      = rundeck_project.test.name
  name              = "local-target"
  description       = "Referenced without a project"
  execution_enabled = true
  command {
    shell_command = "echo local"
  }
}

resource "rundeck_job" "remote" {
  project_name      = rundeck_project.remote.name
  name              = "remote-target"
  description       = "Referenced from another project"
  execution_enabled = true
  command {
    shell_command = "echo remote"
  }
}

resource "rundeck_job" "caller" {
  project_name      = rundeck_project.test.name
  name              = "caller-job"
  description       = "Job whose references are reordered"
  execution_enabled = true
  depends_on        = [rundeck_job.local, rundeck_job.remote]
%s
}
`, caller)
}
