package rundeck

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/rundeck/go-rundeck/rundeck"
)

func TestAccProject_Import(t *testing.T) {
	name := "rundeck_project.main"
	project_name := "terraform-acc-test-basic"
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccProjectCheckDestroy(&project),
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
