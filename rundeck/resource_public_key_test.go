package rundeck

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccPublicKey_basic(t *testing.T) {
	var key rundeck.StorageKeyListResponse
	var keyMaterial string

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccPublicKeyCheckDestroy(&key),
		Steps: []resource.TestStep{
			{
				Config: testAccPublicKeyConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccPublicKeyCheckExists("rundeck_public_key.test", &key, &keyMaterial),
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
						if !strings.Contains(keyMaterial, "test+public+key+for+terraform") {
							return fmt.Errorf("wrong key material")
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccPublicKeyCheckDestroy(key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
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

func testAccPublicKeyCheckExists(rn string, key *rundeck.StorageKeyListResponse, keyMaterial *string) resource.TestCheckFunc {
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

		resp, err := client.StorageKeyGetMaterial(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting key contents: %s", err)
		}

		respMaterial, err2 := io.ReadAll(resp.Body)
		if err2 != nil {
			return fmt.Errorf("error getting key contents: %s", err)
		}

		*keyMaterial = string(respMaterial)

		return nil
	}
}

const testAccPublicKeyConfig_basic = `
resource "rundeck_public_key" "test" {
  path = "terraform_acceptance_tests/public_key"
  key_material = "ssh-rsa test+public+key+for+terraform nobody@nowhere"
}
`
