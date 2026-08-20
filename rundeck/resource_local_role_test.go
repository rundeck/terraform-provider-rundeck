package rundeck

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

func TestAccRundeckLocalRole_basic(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: Local roles (requires the local user store auth realm, API v44+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var role openapi.LoginRoleData

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccLocalRoleCheckDestroy(&role),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckLocalRoleConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccLocalRoleCheckExists("rundeck_local_role.test", &role),
					resource.TestCheckResourceAttr("rundeck_local_role.test", "authority", "terraform-test-role"),
					resource.TestCheckResourceAttr("rundeck_local_role.test", "description", "Test role"),
					resource.TestCheckResourceAttrSet("rundeck_local_role.test", "id"),
				),
			},
		},
	})
}

func TestAccRundeckLocalRole_update(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: Local roles (requires the local user store auth realm, API v44+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var role openapi.LoginRoleData

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccLocalRoleCheckDestroy(&role),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckLocalRoleConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccLocalRoleCheckExists("rundeck_local_role.test", &role),
					resource.TestCheckResourceAttr("rundeck_local_role.test", "description", "Test role"),
				),
			},
			{
				Config: testAccRundeckLocalRoleConfig_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccLocalRoleCheckExists("rundeck_local_role.test", &role),
					resource.TestCheckResourceAttr("rundeck_local_role.test", "description", "Updated test role"),
				),
			},
		},
	})
}

// TestAccRundeckLocalRole_members exercises role membership management. It
// requires a pre-existing local username on the target instance (Terraform
// cannot provision one - rundeck_local_user isn't implemented, see TODO.md),
// so it's skipped unless RUNDECK_LOCAL_ROLE_TEST_USERNAME names a real local
// user.
func TestAccRundeckLocalRole_members(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: Local roles (requires the local user store auth realm, API v44+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	username := os.Getenv("RUNDECK_LOCAL_ROLE_TEST_USERNAME")
	if username == "" {
		t.Skip("Skipping TestAccRundeckLocalRole_members: RUNDECK_LOCAL_ROLE_TEST_USERNAME not set to an existing local username")
	}

	var role openapi.LoginRoleData

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccLocalRoleCheckDestroy(&role),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckLocalRoleConfig_withMember(username),
				Check: resource.ComposeTestCheckFunc(
					testAccLocalRoleCheckExists("rundeck_local_role.test", &role),
					resource.TestCheckResourceAttr("rundeck_local_role.test", "members.#", "1"),
					resource.TestCheckTypeSetElemAttr("rundeck_local_role.test", "members.*", username),
				),
			},
			{
				// Members is Optional+Computed by design: omitting it from
				// config leaves membership untouched (so unrelated applies
				// never silently wipe it). Explicitly asserting an empty set
				// is the correct way to test actual removal.
				Config: testAccRundeckLocalRoleConfig_emptyMembers,
				Check: resource.ComposeTestCheckFunc(
					testAccLocalRoleCheckExists("rundeck_local_role.test", &role),
					resource.TestCheckResourceAttr("rundeck_local_role.test", "members.#", "0"),
				),
			},
		},
	})
}

func testAccLocalRoleCheckDestroy(role *openapi.LoginRoleData) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		roleID := ""
		if role.Id != nil {
			roleID = fmt.Sprintf("%d", *role.Id)
		}
		if roleID == "" {
			return nil
		}

		roleInfo, resp, err := clients.V2.UserAPI.ApiGet(clients.ctx, roleID).Execute()

		if resp != nil && resp.StatusCode == 404 {
			return nil
		}
		if err == nil && roleInfo != nil {
			return fmt.Errorf("local role still exists: %s", roleID)
		}
		return nil
	}
}

func testAccLocalRoleCheckExists(rn string, role *openapi.LoginRoleData) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("role id not set")
		}

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		gotRole, resp, err := clients.V2.UserAPI.ApiGet(clients.ctx, rs.Primary.ID).Execute()
		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to get role info: %v", err)
		}

		*role = *gotRole
		return nil
	}
}

const testAccRundeckLocalRoleConfig_basic = `
resource "rundeck_local_role" "test" {
  authority   = "terraform-test-role"
  description = "Test role"
}
`

const testAccRundeckLocalRoleConfig_updated = `
resource "rundeck_local_role" "test" {
  authority   = "terraform-test-role"
  description = "Updated test role"
}
`

const testAccRundeckLocalRoleConfig_emptyMembers = `
resource "rundeck_local_role" "test" {
  authority   = "terraform-test-role"
  description = "Test role"
  members     = []
}
`

func testAccRundeckLocalRoleConfig_withMember(username string) string {
	return fmt.Sprintf(`
resource "rundeck_local_role" "test" {
  authority   = "terraform-test-role"
  description = "Test role"
  members     = [%q]
}
`, username)
}
