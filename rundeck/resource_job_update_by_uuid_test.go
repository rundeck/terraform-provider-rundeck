package rundeck

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Rundeck writes a job's UUID out under both "uuid" and "id"
// (ScheduledExecution.toMap) but only reads it back from "uuid" (fromMap), so a
// payload carrying "id" alone is resolved by name + group + project instead —
// and that fallback only matches when exactly one job has that name.
//
// This unit test pins the wire format: the UUID must be serialized under
// "uuid", not only "id".
func TestJobJSON_serializesUUIDForImport(t *testing.T) {
	job := &jobJSON{
		ID:      "6bf08fc5-835e-4dea-a83f-56953879b497",
		UUID:    "6bf08fc5-835e-4dea-a83f-56953879b497",
		Name:    "a-job",
		Project: "a-project",
	}

	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := decoded["uuid"]; got != job.UUID {
		t.Errorf("uuid = %v, want %q — Rundeck resolves the job to update from this field", got, job.UUID)
	}
}

// An unset UUID must stay out of the payload entirely, so that creating a job
// still lets Rundeck assign one.
func TestJobJSON_omitsUUIDWhenUnset(t *testing.T) {
	raw, err := json.Marshal(&jobJSON{Name: "a-job", Project: "a-project"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if v, exists := decoded["uuid"]; exists {
		t.Errorf("uuid = %v, want it omitted when unset", v)
	}
}

// TestAccJob_renameKeepsSameJob is the behavioural test: renaming a job must
// rename it in place. Before this fix the update was resolved by name, so a
// rename matched nothing and Rundeck created a second job, leaving the original
// orphaned and invisible to Terraform. The id is asserted to be stable across
// the rename.
func TestAccJob_renameKeepsSameJob(t *testing.T) {
	var jobIDBefore string

	captureJobID := func(id *string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources["rundeck_job.test"]
			if !ok {
				return fmt.Errorf("rundeck_job.test not found in state")
			}
			*id = rs.Primary.ID
			return nil
		}
	}

	requireSameJobID := func(want *string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources["rundeck_job.test"]
			if !ok {
				return fmt.Errorf("rundeck_job.test not found in state")
			}
			if rs.Primary.ID != *want {
				return fmt.Errorf(
					"job id changed across rename: %s -> %s (a new job was created instead of renaming the existing one)",
					*want, rs.Primary.ID)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_renameBefore,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "name", "job-before-rename"),
					captureJobID(&jobIDBefore),
				),
			},
			{
				Config: testAccJobConfig_renameAfter,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "name", "job-after-rename"),
					requireSameJobID(&jobIDBefore),
				),
			},
			{
				Config:   testAccJobConfig_renameAfter,
				PlanOnly: true,
			},
		},
	})
}

const testAccJobConfig_renameBefore = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-job-rename"
  description = "Test project for job rename"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name      = rundeck_project.test.name
  name              = "job-before-rename"
  description       = "Job that will be renamed"
  execution_enabled = true
  command {
    shell_command = "echo hello"
  }
}
`

const testAccJobConfig_renameAfter = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-job-rename"
  description = "Test project for job rename"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name      = rundeck_project.test.name
  name              = "job-after-rename"
  description       = "Job that will be renamed"
  execution_enabled = true
  command {
    shell_command = "echo hello"
  }
}
`
