package rundeck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPrivateKey_basic(t *testing.T) {
	var key rundeck.StorageKeyListResponse

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccBaseKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccBaseKeyCheckExists("rundeck_private_key.test", &key),
					func(s *terraform.State) error {
						if expected := "keys/terraform_acceptance_tests/private_key"; *key.Path != expected {
							return fmt.Errorf("wrong path; expected %v, got %v", expected, *key.Path)
						}
						if !strings.HasSuffix(*key.URL, "/storage/keys/terraform_acceptance_tests/private_key") {
							return fmt.Errorf("wrong URL; expected to end with the key path")
						}
						if expected := "file"; *key.Type != expected {
							return fmt.Errorf("wrong resource type; expected %v, got %v", expected, *key.Type)
						}
						if expected := rundeck.Private; key.Meta.RundeckKeyType != expected {
							return fmt.Errorf("wrong key type; expected %v, got %v", expected, key.Meta.RundeckKeyType)
						}
						if expected := "application/octet-stream"; *key.Meta.RundeckContentType != expected {
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

const testAccPrivateKeyConfig_basic = `
resource "rundeck_private_key" "test" {
  path = "terraform_acceptance_tests/private_key"
  key_material = "this is not a real private key"
}
`
