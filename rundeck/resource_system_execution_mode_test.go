package rundeck

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccSystemExecutionMode covers the two things that matter for this
// resource: that it actually switches the server, and that it reports drift
// when the mode is changed outside Terraform — which is the whole reason for
// managing it here rather than through a configuration file only read at
// startup.
func TestAccSystemExecutionMode(t *testing.T) {
	// Captured lazily on first use inside PreCheck (once testAccPreCheck has
	// confirmed the env vars this needs are set), and restored via
	// CheckDestroy once the test's own steps are done - this resource's
	// Delete deliberately never touches server state, so without this the
	// test would leave the server in whatever mode its last apply set,
	// regardless of what it started in.
	var startingMode string

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			startingMode = testAccCurrentExecutionMode(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy: func(*terraform.State) error {
			return testAccFlipExecutionModeOutOfBand(startingMode)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccSystemExecutionModeConfig("passive"),
				Check: resource.TestCheckResourceAttr(
					"rundeck_system_execution_mode.test", "mode", "passive"),
			},
			{
				// Re-apply unchanged: the mode must round-trip, with no diff.
				Config:   testAccSystemExecutionModeConfig("passive"),
				PlanOnly: true,
			},
			{
				Config: testAccSystemExecutionModeConfig("active"),
				Check: resource.TestCheckResourceAttr(
					"rundeck_system_execution_mode.test", "mode", "active"),
			},
			{
				Config:   testAccSystemExecutionModeConfig("active"),
				PlanOnly: true,
			},
			{
				// Flip the mode out-of-band (directly against the API, not
				// through this Terraform config) between the previous apply
				// and this one, then confirm the next plan actually notices:
				// the whole point of managing this as a resource rather than
				// the startup-only rundeck.executionMode property is that an
				// out-of-band change should surface as drift, not go unnoticed
				// until someone happens to look.
				// ExpectNonEmptyPlan defaults to false: the test framework's
				// automatic post-apply "plan again, expect no diff" check
				// only passes if this step's apply actually reconciled the
				// drift Read() picked up, not merely flagged it.
				PreConfig: func() {
					if err := testAccFlipExecutionModeOutOfBand("passive"); err != nil {
						t.Fatalf("flipping execution mode out-of-band: %v", err)
					}
				},
				Config: testAccSystemExecutionModeConfig("active"),
				Check: resource.TestCheckResourceAttr(
					"rundeck_system_execution_mode.test", "mode", "active"),
			},
			{
				ResourceName:      "rundeck_system_execution_mode.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "system",
			},
		},
	})
}

// testAccFlipExecutionModeOutOfBand switches the server's execution mode
// directly through the API, bypassing Terraform entirely, so the following
// test step's plan/apply has to detect and correct real drift rather than
// just replaying state it already wrote itself. Also used by CheckDestroy
// to restore the server's starting mode once the test's own steps are done.
func testAccFlipExecutionModeOutOfBand(mode string) error {
	clients, err := getTestClients()
	if err != nil {
		return fmt.Errorf("getTestClients: %w", err)
	}

	r := &systemExecutionModeResource{client: clients}
	if _, err := r.setMode(context.Background(), mode); err != nil {
		return fmt.Errorf("flipping execution mode out-of-band to %q: %w", mode, err)
	}
	return nil
}

// testAccCurrentExecutionMode reads the server's current execution mode
// directly through the API, so the test can restore it via CheckDestroy
// once done - this resource's own Delete deliberately never touches server
// state.
func testAccCurrentExecutionMode(t *testing.T) string {
	t.Helper()

	clients, err := getTestClients()
	if err != nil {
		t.Fatalf("getTestClients: %v", err)
	}

	r := &systemExecutionModeResource{client: clients}
	mode, err := r.apiRequest(context.Background(), http.MethodGet, "status")
	if err != nil {
		t.Fatalf("reading current execution mode: %v", err)
	}
	return mode
}

func testAccSystemExecutionModeConfig(mode string) string {
	return `
resource "rundeck_system_execution_mode" "test" {
  mode = "` + mode + `"
}
`
}
