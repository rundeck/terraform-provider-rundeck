package rundeck

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccRundeckJob_Import(t *testing.T) {
	name := "rundeck_job.test"
	project_name := "terraform-acc-test-job"
	var job JobDetail

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccJobCheckDestroy(&job),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_basic,
				Check:  testAccJobCheckExists(name, &job),
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
