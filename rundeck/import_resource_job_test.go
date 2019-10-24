package rundeck

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
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
			},
		},
	})
}
