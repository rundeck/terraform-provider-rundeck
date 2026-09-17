package rundeck

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// A configured uuid must reach the payload's "uuid" field: that is the one the
// import reads back, and the one uuidOption=preserve acts on.
func TestPlanToJobJSON_carriesConfiguredUUID(t *testing.T) {
	const want = "11111111-2222-3333-4444-555555555555"

	r := &jobResource{}
	job, err := r.planToJobJSON(context.Background(), &jobResourceModel{
		Name:        types.StringValue("test-job"),
		ProjectName: types.StringValue("test-project"),
		Command:     testSingleShellCommandList(t),
		UUID:        types.StringValue(want),
	})
	if err != nil {
		t.Fatalf("planToJobJSON: %v", err)
	}

	if job.UUID != want {
		t.Errorf("uuid = %q, want %q", job.UUID, want)
	}
}

// Unset, the uuid must stay out of the payload so Rundeck keeps assigning one.
func TestPlanToJobJSON_omitsUnsetUUID(t *testing.T) {
	r := &jobResource{}
	job, err := r.planToJobJSON(context.Background(), &jobResourceModel{
		Name:        types.StringValue("test-job"),
		ProjectName: types.StringValue("test-project"),
		Command:     testSingleShellCommandList(t),
	})
	if err != nil {
		t.Fatalf("planToJobJSON: %v", err)
	}

	if job.UUID != "" {
		t.Errorf("uuid = %q, want it empty so the field is omitted", job.UUID)
	}
}

func TestCanonicalUUIDPattern(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"11111111-2222-3333-4444-555555555555", true},
		{"6bf08fc5-835e-4dea-a83f-56953879b497", true},
		{"6BF08FC5-835E-4DEA-A83F-56953879B497", false},
		{"6bf08fc5835e4dea a83f56953879b497", false},
		{"6bf08fc5-835e-4dea-a83f", false},
		// Rundeck's own constraint would accept this; the provider does not.
		{"deploy-prod-app1", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := canonicalUUIDPattern.MatchString(tc.value); got != tc.valid {
			t.Errorf("MatchString(%q) = %v, want %v", tc.value, got, tc.valid)
		}
	}
}

// TestAccJob_configuredUUIDIsPreserved covers the three things a pinned uuid
// promises: Rundeck honours it on create, an update keeps the job under it, and
// changing it replaces the job.
func TestAccJob_configuredUUIDIsPreserved(t *testing.T) {
	pinned, replacement := randomJobUUID(t), randomJobUUID(t)

	requireJobID := func(want *string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			rs, ok := s.RootModule().Resources["rundeck_job.test"]
			if !ok {
				return fmt.Errorf("rundeck_job.test not found in state")
			}
			if rs.Primary.ID != *want {
				return fmt.Errorf("job id = %s, want %s (Rundeck did not honour the configured uuid)",
					rs.Primary.ID, *want)
			}

			clients, err := getTestClients()
			if err != nil {
				return fmt.Errorf("error getting test client: %s", err)
			}
			job, err := GetJobJSON(clients.V1, *want)
			if err != nil {
				return fmt.Errorf("job %s could not be read back from Rundeck: %s", *want, err)
			}
			if got := jobIdentity(job.UUID, job.ID); got != *want {
				return fmt.Errorf("uuid in Rundeck = %s, want %s", got, *want)
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
				Config: testAccJobConfig_settableUUID(pinned, "Job whose identity comes from the configuration"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "uuid", pinned),
					requireJobID(&pinned),
				),
			},
			{
				Config:   testAccJobConfig_settableUUID(pinned, "Job whose identity comes from the configuration"),
				PlanOnly: true,
			},
			{
				// Update with the uuid still pinned: the job must stay under it.
				Config: testAccJobConfig_settableUUID(pinned, "Description changed while the uuid stays pinned"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "description", "Description changed while the uuid stays pinned"),
					resource.TestCheckResourceAttr("rundeck_job.test", "uuid", pinned),
					requireJobID(&pinned),
				),
			},
			{
				// Changing the uuid replaces the job under the new one.
				Config: testAccJobConfig_settableUUID(replacement, "Description changed while the uuid stays pinned"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "uuid", replacement),
					requireJobID(&replacement),
				),
			},
		},
	})
}

// randomJobUUID keeps each run on its own UUID: Rundeck enforces uniqueness
// instance-wide, so a literal would strand the next run behind any job a killed
// run left behind.
func randomJobUUID(t *testing.T) string {
	t.Helper()

	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("generating a uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func testAccJobConfig_settableUUID(uuid, description string) string {
	return fmt.Sprintf(`
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-job-uuid"
  description = "Test project for settable job uuid"
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
  uuid              = %q
  name              = "job-with-pinned-uuid"
  description       = %q
  execution_enabled = true
  command {
    shell_command = "echo hello"
  }
}
`, uuid, description)
}
