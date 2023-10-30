package rundeck

import (
	"context"
	"fmt"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func testAccBaseKeyCheckDestroy(key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
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

func testAccBaseKeyCheckExists(rn string, key *rundeck.StorageKeyListResponse) resource.TestCheckFunc {
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
