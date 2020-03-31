package rundeck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccPrivateKey_basic(t *testing.T) {
	var key rundeck.StorageKeyListResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPrivateKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			{
				Config: testAccPrivateKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPrivateKeyCheckExists("rundeck_private_key.test", &key),
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
						// Rundeck won't let us re-retrieve a private key payload, so we can't test
						// that the key material was submitted and stored correctly.
						return nil
					},
				),
			},
		},
	})
}

func testAccPrivateKeyCheckDestroy(key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()

		resp, err := client.StorageKeyGetMetadata(ctx, *key.Path)

		if resp.StatusCode == 200 {
			return fmt.Errorf("key still exists")
		}
		if resp.StatusCode != 404 {
			return fmt.Errorf("got something other than NotFoundError (%v) when getting key", err)
		}

		return nil
	}
}

func testAccPrivateKeyCheckExists(rn string, key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("key id not set")
		}

		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()
		gotKey, err := client.StorageKeyGetMetadata(ctx, rs.Primary.ID)
		if gotKey.StatusCode == 404 || err != nil {
			return fmt.Errorf("error getting key metadata: %s", err)
		}

		*key = gotKey

		return nil
	}
}

const testAccPrivateKeyConfig_basic = `
resource "rundeck_private_key" "test" {
  path = "terraform_acceptance_tests/private_key"
  key_material = "this is not a real private key"
}
`
