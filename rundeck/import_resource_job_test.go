package rundeck

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRundeckJob_Import(t *testing.T) {
	name := "rundeck_job.test"
	project_name := "terraform-acc-test-job"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "basic-job"),
					resource.TestCheckResourceAttrSet(name, "id"),
				),
			},
			{
				ResourceName:        name,
				ImportStateIdPrefix: fmt.Sprintf("%s/", project_name),
				ImportState:         true,
				ImportStateVerify:   true,
				// These fields have default values that may not be returned by the API
				ImportStateVerifyIgnore: []string{"runner_selector_filter_mode", "runner_selector_filter_type"},
			},
		},
	})
}
