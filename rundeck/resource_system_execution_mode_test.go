package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccSystemExecutionMode covers the two things that matter for this
// resource: that it actually switches the server, and that it reports drift
// when the mode is changed outside Terraform — which is the whole reason for
// managing it here rather than through a configuration file only read at
// startup.
func TestAccSystemExecutionMode(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
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
				PreConfig: func() { testAccFlipExecutionModeOutOfBand(t, "passive") },
				Config:    testAccSystemExecutionModeConfig("active"),
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
// just replaying state it already wrote itself.
func testAccFlipExecutionModeOutOfBand(t *testing.T, mode string) {
	t.Helper()

	clients, err := getTestClients()
	if err != nil {
		t.Fatalf("getTestClients: %v", err)
	}

	r := &systemExecutionModeResource{client: clients}
	if _, err := r.setMode(context.Background(), mode); err != nil {
		t.Fatalf("flipping execution mode out-of-band to %q: %v", mode, err)
	}
}

func testAccSystemExecutionModeConfig(mode string) string {
	return `
resource "rundeck_system_execution_mode" "test" {
  mode = "` + mode + `"
}
`
}
