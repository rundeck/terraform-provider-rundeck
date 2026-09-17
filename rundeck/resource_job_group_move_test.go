package rundeck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// A configured group must reach the payload, which is what moves the job.
func TestPlanToJobJSON_carriesGroupName(t *testing.T) {
	r := &jobResource{}
	job, err := r.planToJobJSON(context.Background(), &jobResourceModel{
		Name:        types.StringValue("test-job"),
		ProjectName: types.StringValue("test-project"),
		Command:     testSingleShellCommandList(t),
		GroupName:   types.StringValue("reorg/after"),
	})
	if err != nil {
		t.Fatalf("planToJobJSON: %v", err)
	}

	if job.Group != "reorg/after" {
		t.Errorf("group = %q, want %q", job.Group, "reorg/after")
	}
}

// A null group must leave "group" out of the payload entirely. That omission is
// what clears the group: Rundeck reads it back as
// se.groupPath = data['group'] ? data['group'] : null (ScheduledExecution.fromMap),
// so an absent key moves the job to the project root. Sending an empty string
// would be indistinguishable here, but only because omitempty drops it — this
// test pins that, since the module configures null rather than "".
func TestPlanToJobJSON_omitsNullGroupNameSoTheJobMovesToTheRoot(t *testing.T) {
	r := &jobResource{}
	job, err := r.planToJobJSON(context.Background(), &jobResourceModel{
		Name:        types.StringValue("test-job"),
		ProjectName: types.StringValue("test-project"),
		Command:     testSingleShellCommandList(t),
	})
	if err != nil {
		t.Fatalf("planToJobJSON: %v", err)
	}

	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if v, exists := decoded["group"]; exists {
		t.Errorf("group = %v, want it omitted so Rundeck moves the job to the project root", v)
	}
}

// TestAccJob_groupNameMovesInPlace is the behavioural test. Changing group_name
// must move the job rather than replace it: same UUID, same execution history,
// and the group actually changed server-side. The last step clears the group,
// which is how the job returns to the project root.
func TestAccJob_groupNameMovesInPlace(t *testing.T) {
	var (
		jobUUID          string
		executionsBefore []string
	)

	captureAndGiveItHistory := func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["rundeck_job.test"]
		if !ok {
			return fmt.Errorf("rundeck_job.test not found in state")
		}
		jobUUID = rs.Primary.ID

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("error getting test client: %s", err)
		}
		executionID, err := testRunJob(clients, jobUUID)
		if err != nil {
			return fmt.Errorf("could not run the job to give it an execution history: %s "+
				"(a server left in passive execution mode refuses to run jobs)", err)
		}
		executionsBefore = []string{executionID}
		return nil
	}

	requireMovedInPlace := func(wantGroup string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources["rundeck_job.test"]
			if !ok {
				return fmt.Errorf("rundeck_job.test not found in state")
			}
			if jobUUID == "" {
				return fmt.Errorf("no job captured: this check needs a prior step to record the job and its executions")
			}
			if rs.Primary.ID != jobUUID {
				return fmt.Errorf("job id changed across the move: %s -> %s (the job was replaced instead of moved)",
					jobUUID, rs.Primary.ID)
			}

			clients, err := getTestClients()
			if err != nil {
				return fmt.Errorf("error getting test client: %s", err)
			}

			job, err := GetJobJSON(clients.V1, jobUUID)
			if err != nil {
				return fmt.Errorf("could not read job back: %s", err)
			}
			if job.Group != wantGroup {
				return fmt.Errorf("group server-side = %q, want %q", job.Group, wantGroup)
			}

			if len(executionsBefore) == 0 {
				return fmt.Errorf("no executions recorded before the move, so the history check would prove nothing")
			}
			after, err := testJobExecutionIDs(clients, jobUUID)
			if err != nil {
				return fmt.Errorf("could not list executions: %s", err)
			}
			kept := make(map[string]bool, len(after))
			for _, id := range after {
				kept[id] = true
			}
			for _, id := range executionsBefore {
				if !kept[id] {
					return fmt.Errorf("execution %s is gone after the move: the job was recreated and its history lost", id)
				}
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
				Config: testAccJobConfig_group(`group_name        = "reorg/before"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "group_name", "reorg/before"),
					captureAndGiveItHistory,
					requireMovedInPlace("reorg/before"),
				),
			},
			{
				Config: testAccJobConfig_group(`group_name        = "reorg/after"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "group_name", "reorg/after"),
					requireMovedInPlace("reorg/after"),
				),
			},
			{
				Config:   testAccJobConfig_group(`group_name        = "reorg/after"`),
				PlanOnly: true,
			},
			{
				// group_name absent from the configuration: the job goes back to
				// the project root, still without being replaced.
				Config: testAccJobConfig_group(""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("rundeck_job.test", "group_name"),
					requireMovedInPlace(""),
				),
			},
			{
				Config:   testAccJobConfig_group(""),
				PlanOnly: true,
			},
		},
	})
}

// testRunJob starts the job and returns the id of the execution Rundeck recorded.
func testRunJob(clients *RundeckClients, jobID string) (string, error) {
	resp, err := clients.V2.JobsAPI.ApiJobRun(clients.ctx, jobID).Execute()
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", fmt.Errorf("running job %s: %w", jobID, err)
	}

	body, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("running job %s returned status %d: %s", jobID, resp.StatusCode, testTruncate(body))
	}
	if readErr != nil {
		return "", fmt.Errorf("reading run response: %w", readErr)
	}

	var execution struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body, &execution); err != nil {
		return "", fmt.Errorf("decoding run response %s: %w", testTruncate(body), err)
	}
	if execution.ID == 0 {
		return "", fmt.Errorf("run response carried no execution id: %s", testTruncate(body))
	}
	return strconv.Itoa(execution.ID), nil
}

// testJobExecutionIDs lists every execution Rundeck holds for the job.
func testJobExecutionIDs(clients *RundeckClients, jobID string) ([]string, error) {
	list, err := clients.V1.JobExecutionList(clients.ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("listing executions of job %s: %w", jobID, err)
	}
	if list.Executions == nil {
		return nil, nil
	}

	ids := make([]string, 0, len(*list.Executions))
	for _, e := range *list.Executions {
		if e.ID != nil {
			ids = append(ids, strconv.Itoa(int(*e.ID)))
		}
	}
	return ids, nil
}

func testTruncate(body []byte) string {
	const max = 512
	if len(body) > max {
		return string(body[:max]) + "…"
	}
	return string(body)
}

func testAccJobConfig_group(groupLine string) string {
	return fmt.Sprintf(`
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-job-group-move"
  description = "Test project for moving a job between groups"
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
  name              = "job-to-move"
  %s
  description       = "Job that will be moved between groups"
  execution_enabled = true
  command {
    shell_command = "echo hello"
  }
}
`, groupLine)
}
