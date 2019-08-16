package rundeck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccPassword_basic(t *testing.T) {
	var password rundeck.StorageKeyListResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPasswordCheckDestroy(&password),
		Steps: []resource.TestStep{
			{
				Config: testAccPasswordConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPasswordCheckExists("rundeck_password.test", &password),
					func(s *terraform.State) error {
						if expected := "keys/terraform_acceptance_tests/password"; *password.Path != expected {
							return fmt.Errorf("wrong path; expected %v, got %v", expected, *password.Path)
						}
						if !strings.HasSuffix(*password.URL, "/storage/keys/terraform_acceptance_tests/password") {
							return fmt.Errorf("wrong URL; expected to end with the password path")
						}
						if expected := "file"; *password.Type != expected {
							return fmt.Errorf("wrong resource type; expected %v, got %v", expected, *password.Type)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccPasswordCheckDestroy(password *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()

		resp, err := client.StorageKeyGetMetadata(ctx, *password.Path)

		if resp.StatusCode == 200 {
			return fmt.Errorf("password still exists")
		}
		if resp.StatusCode != 404 {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting password", err)
		}

		return nil
	}
}

func testAccPasswordCheckExists(rn string, password *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("password id not set")
		}

		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()
		gotKey, err := client.StorageKeyGetMetadata(ctx, rs.Primary.ID)
		if gotKey.StatusCode == 404 || err != nil {
			return fmt.Errorf("error getting password metadata: %s", err)
		}

		*password = gotKey

		return nil
	}
}

const testAccPasswordConfig_basic = `
resource "rundeck_password" "test" {
  path = "terraform_acceptance_tests/password"
  password = "this is not a real password"
}
`
