package rundeck

import (
	"fmt"
	"github.com/hashicorp/terraform/terraform"

	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccRundeckPassword_Import(t *testing.T) {
	name := "rundeck_password.test"

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccPasswordConfig_basic,
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: false,
				ImportStateCheck: resource.ImportStateCheckFunc(
					func(s []*terraform.InstanceState) error {
						if expected := "THIS_WILL_CHANGE"; s[0].Attributes["password"] != expected {
							return fmt.Errorf("Password can't be read from rundeck. Should be set to THIS_WILL_CHANGE to delete/create imported password")
						}
						return nil
					},
				),
			},
		},
	})
}
