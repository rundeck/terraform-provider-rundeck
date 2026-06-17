package rundeck

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestUUIDFromPlan verifies that a caller-supplied uuid is forwarded to the
// import payload, while a null/unknown uuid yields an empty string so the
// omitempty payload field is dropped and Rundeck generates one.
func TestUUIDFromPlan(t *testing.T) {
	cases := []struct {
		name string
		uuid types.String
		want string
	}{
		{"supplied", types.StringValue("11111111-2222-3333-4444-555555555555"), "11111111-2222-3333-4444-555555555555"},
		{"null", types.StringNull(), ""},
		{"unknown", types.StringUnknown(), ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := &jobResourceModel{UUID: tc.uuid}
			if got := uuidFromPlan(plan); got != tc.want {
				t.Fatalf("uuidFromPlan() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestJobJSONUUIDPayload confirms the import payload carries the UUID in the
// "uuid" field (the field Rundeck's import honors for uuidOption=preserve; the
// "id" field is ignored on import) only when one is supplied. An empty value
// must be omitted entirely so Rundeck mints one when none was given.
func TestJobJSONUUIDPayload(t *testing.T) {
	withUUID, err := json.Marshal(&jobJSON{UUID: "abc-123", Name: "j", Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(withUUID), `"uuid":"abc-123"`) {
		t.Fatalf("expected uuid in payload, got %s", withUUID)
	}

	noUUID, err := json.Marshal(&jobJSON{UUID: "", Name: "j", Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(noUUID), `"uuid"`) {
		t.Fatalf("expected uuid to be omitted when empty, got %s", noUUID)
	}
}

// TestUUIDRegexp verifies the uuid attribute validator accepts canonical
// lowercase UUIDs and rejects malformed or non-canonical input (uppercase,
// missing/extra characters, surrounding whitespace).
func TestUUIDRegexp(t *testing.T) {
	valid := []string{
		"11111111-2222-3333-4444-555555555555",
		"abcdabcd-1234-1234-1234-abcdabcdabcd",
		"00000000-0000-0000-0000-000000000000",
		"deadbeef-dead-beef-dead-beefdeadbeef",
	}
	for _, s := range valid {
		if !uuidRegexp.MatchString(s) {
			t.Errorf("expected %q to be accepted", s)
		}
	}

	invalid := []string{
		"",
		"not-a-uuid",
		"ABCDABCD-1234-1234-1234-ABCDABCDABCD",      // uppercase
		"11111111-2222-3333-4444-55555555555",       // too short
		"11111111-2222-3333-4444-5555555555555",     // too long
		"11111111222233334444555555555555",          // missing hyphens
		"11111111-2222-3333-4444-55555555555g",      // non-hex
		" 11111111-2222-3333-4444-555555555555",     // leading space
		"11111111-2222-3333-4444-555555555555\n",    // trailing newline
		"{11111111-2222-3333-4444-555555555555}",    // braced form
	}
	for _, s := range invalid {
		if uuidRegexp.MatchString(s) {
			t.Errorf("expected %q to be rejected", s)
		}
	}
}

// TestAccJob_uuidPreserved verifies a caller-supplied uuid becomes the job's
// Rundeck UUID and survives a no-op re-apply (the second identical step would
// produce a non-empty plan or a new id if the uuid were not preserved).
func TestAccJob_uuidPreserved(t *testing.T) {
	const jobUUID = "abcdabcd-1234-1234-1234-abcdabcdabcd"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_uuid,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.uuid_test", "uuid", jobUUID),
					resource.TestCheckResourceAttr("rundeck_job.uuid_test", "id", jobUUID),
				),
			},
			{
				// Re-apply the identical config; the uuid (and thus id) must
				// be unchanged and the plan must be empty (no churn).
				Config: testAccJobConfig_uuid,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.uuid_test", "uuid", jobUUID),
					resource.TestCheckResourceAttr("rundeck_job.uuid_test", "id", jobUUID),
				),
			},
		},
	})
}

// TestAccJob_uuidParallelUpdate is a regression guard for UUID stability under
// concurrent apply (Terraform's default parallelism is 10, i.e. > 1): creating
// then updating many jobs at once must never churn any job's uuid/id. NOTE:
// this is a guard, not a reproduction of the historical "id changed under
// -parallelism>1" report (gap #5) — that symptom could not be reproduced on
// Rundeck 5.x, where concurrent updates of distinct jobs preserve uuids
// regardless of whether the uuid is sent in the import payload.
func TestAccJob_uuidParallelUpdate(t *testing.T) {
	const n = 20
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				// Create all n jobs (concurrently, default parallelism).
				Config: testAccJobConfig_uuidFanout(n, "v1"),
				Check:  testAccCheckFanoutUUIDs(n),
			},
			{
				// Update every job at once; the concurrent update must not
				// churn any uuid/id.
				Config: testAccJobConfig_uuidFanout(n, "v2"),
				Check:  testAccCheckFanoutUUIDs(n),
			},
		},
	})
}

// fanoutUUID returns a deterministic, canonical-lowercase UUID for job i.
func fanoutUUID(i int) string {
	return fmt.Sprintf("aaaaaaaa-0000-0000-0000-%012d", i)
}

// testAccCheckFanoutUUIDs asserts each job's uuid and id equal the deterministic
// value it was created with.
func testAccCheckFanoutUUIDs(n int) resource.TestCheckFunc {
	checks := make([]resource.TestCheckFunc, 0, n*2)
	for i := 0; i < n; i++ {
		res := fmt.Sprintf("rundeck_job.fanout_%d", i)
		checks = append(checks,
			resource.TestCheckResourceAttr(res, "uuid", fanoutUUID(i)),
			resource.TestCheckResourceAttr(res, "id", fanoutUUID(i)),
		)
	}
	return resource.ComposeTestCheckFunc(checks...)
}

func testAccJobConfig_uuidFanout(n int, version string) string {
	var b strings.Builder
	b.WriteString(`
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-fanout"
  description = "parent project for job uuid parallelism test"

  resource_model_source {
    type = "file"
    config = {
        format = "resourceyaml"
        file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}
`)
	for i := 0; i < n; i++ {
		b.WriteString(fmt.Sprintf(`
resource "rundeck_job" "fanout_%d" {
  project_name = "${rundeck_project.test.name}"
  uuid = "%s"
  name = "fanout-job-%d"
  description = "fanout job %d (%s)"
  execution_enabled = true
  command {
    shell_command = "echo %s"
  }
}
`, i, fanoutUUID(i), i, i, version, version))
	}
	return b.String()
}

const testAccJobConfig_uuid = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job-uuid"
  description = "parent project for job uuid acceptance test"

  resource_model_source {
    type = "file"
    config = {
        format = "resourceyaml"
        file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}
resource "rundeck_job" "uuid_test" {
  project_name = "${rundeck_project.test.name}"
  uuid = "abcdabcd-1234-1234-1234-abcdabcdabcd"
  name = "uuid-job"
  description = "A job with a caller-supplied UUID"
  execution_enabled = true
  command {
    description = "Prints Hello World"
    shell_command = "echo Hello World"
  }
}
`
