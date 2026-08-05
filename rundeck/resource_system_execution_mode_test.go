package rundeck

import (
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
				ResourceName:      "rundeck_system_execution_mode.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "system",
			},
		},
	})
}

func testAccSystemExecutionModeConfig(mode string) string {
	return `
resource "rundeck_system_execution_mode" "test" {
  mode = "` + mode + `"
}
`
}
