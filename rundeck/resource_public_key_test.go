package rundeck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPublicKey_basic(t *testing.T) {
	var key rundeck.StorageKeyListResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy: testAccPublicKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			{
				Config: testAccPublicKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPublicKeyCheckExists("rundeck_public_key.test", &key),
					func(s *terraform.State) error {
						if expected := "keys/terraform_acceptance_tests/public_key"; *key.Path != expected {
							return fmt.Errorf("wrong path; expected %v, got %v", expected, key.Path)
						}
						if !strings.HasSuffix(*key.URL, "/storage/keys/terraform_acceptance_tests/public_key") {
							return fmt.Errorf("wrong URL; expected to end with the key path")
						}
						if expected := "file"; *key.Type != expected {
							return fmt.Errorf("wrong resource type; expected %v, got %v", expected, *key.Type)
						}
						if expected := rundeck.Public; key.Meta.RundeckKeyType != expected {
							return fmt.Errorf("wrong key type; expected %v, got %v", expected, key.Meta.RundeckKeyType)
						}
						// Note: We don't check key material because the go-rundeck client's
						// StorageKeyGetMaterial returns JSON metadata instead of actual content for public keys
						return nil
					},
				),
			},
		},
	})
}

func testAccPublicKeyCheckDestroy(key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		client := clients.V1
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

func testAccPublicKeyCheckExists(rn string, key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("key id not set")
		}

		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		client := clients.V1
		ctx := context.Background()
		gotKey, err := client.StorageKeyGetMetadata(ctx, rs.Primary.ID)
		if gotKey.StatusCode == 404 || err != nil {
			return fmt.Errorf("error getting key metadata: %s", err)
		}

		*key = gotKey

		return nil
	}
}

const testAccPublicKeyConfig_basic = `
resource "rundeck_public_key" "test" {
  path = "terraform_acceptance_tests/public_key"
  key_material = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3m5YqLb8PlVdJQL+5yx+KEirU3Pp5ONh7TCuT7gJPV8R1KTVI7TzGD1lR3e3L8P06rFvFnhgQkPJBPL6CcBdABLm9N/xVLQhFjkl0l4pGT6rZh3LiLXXvvULgVqyN+hLd5VXF6p5IjZKZQy9O3hGYhKh+rCt7gJxV4j5A6hXKZQ test-key-for-terraform"
}
`
