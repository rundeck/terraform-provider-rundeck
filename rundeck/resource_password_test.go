package rundeck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccPassword_basic(t *testing.T) {
	var key rundeck.StorageKeyListResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccBaseKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			{
				Config: testAccPasswordConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccBaseKeyCheckExists("rundeck_password.test", &key),
					func(s *terraform.State) error {
						if expected := "keys/terraform_acceptance_tests/password"; *key.Path != expected {
							return fmt.Errorf("wrong path; expected %v, got %v", expected, *key.Path)
						}
						if !strings.HasSuffix(*key.URL, "/storage/keys/terraform_acceptance_tests/password") {
							return fmt.Errorf("wrong URL; expected to end with the key path")
						}
						if expected := "file"; *key.Type != expected {
							return fmt.Errorf("wrong resource type; expected %v, got %v", expected, *key.Type)
						}
						if expected := ""; string(key.Meta.RundeckKeyType) != expected {
							return fmt.Errorf("wrong key type; expected %v, got %v", expected, key.Meta.RundeckKeyType)
						}
						if expected := "application/x-rundeck-data-password"; *key.Meta.RundeckContentType != expected {
							return fmt.Errorf("wrong RundeckContentType; expected %v, got %v", expected, *key.Meta.RundeckContentType)
						}
						// Rundeck won't let us re-retrieve a private key payload, so we can't test
						// that the key material was submitted and stored correctly.
						return nil
					},
				),
			},
		},
	})
}

const testAccPasswordConfig_basic = `
resource "rundeck_password" "test" {
  path = "terraform_acceptance_tests/password"
  password = "qwerty"
}
`
