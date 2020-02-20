package rundeck

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/rundeck/go-rundeck/rundeck"
)

func TestAccProject_Import(t *testing.T) {
	name := "rundeck_project.main"
	project_name := "terraform-acc-test-basic"
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccProjectCheckDestroy(&project),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig_basic,
				Check:  testAccProjectCheckExists(name, &project),
			},
			{
				ResourceName:      name,
				ImportStateId:     project_name,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
