package rundeck

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/rundeck/go-rundeck/rundeck"
)

func TestAccAclPolicy_basic(t *testing.T) {
	var aclPolicy string

	basicAclResourceConfig := fmt.Sprintf(testAccAclPolicyConfig_basic, basicAclPolicy)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccAclPolicyCheckDestroy("TerraformBasicAcl.aclpolicy"),
		Steps: []resource.TestStep{
			{
				Config: basicAclResourceConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAclPolicyCheckExists("rundeck_acl_policy.test", &aclPolicy),
					func(s *terraform.State) error {
						if expected := basicAclPolicy; aclPolicy != expected {
							return fmt.Errorf("acl policy does not match; expected (%v), got (%v)", expected, aclPolicy)
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccAclPolicyCheckDestroy(policyName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()
		resp, err := client.SystemACLPolicyGet(ctx, policyName)
		if err != nil || resp.StatusCode != 404 {
			return fmt.Errorf("key still exists")
		}

		return nil
	}
}

func testAccAclPolicyCheckExists(rn string, aclPolicy *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("job id not set")
		}

		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()
		resp, err := client.SystemACLPolicyGet(ctx, rs.Primary.ID)
		if err != nil || resp.StatusCode != 200 {
			return fmt.Errorf("Error getting ACL policy: (%v) (%v)", rs.Primary.ID, resp.StatusCode)
		}

		*aclPolicy = *resp.Contents

		return nil
	}
}

const basicAclPolicy = `description: Admin project level access control. Applies to resources within a specific project.
context:
  project: '.*' # all projects
for:
  resource:
    - equals:
        kind: job
      allow: [create] # allow create jobs
    - equals:
        kind: node
      allow: [read,create,update,refresh] # allow refresh node sources
    - equals:
        kind: event
      allow: [read,create] # allow read/create events
  adhoc:
    - allow: [read,run,runAs,kill,killAs] # allow running/killing adhoc jobs
  job:
    - allow: [create,read,update,delete,run,runAs,kill,killAs] # allow create/read/write/delete/run/kill of all jobs
  node:
    - allow: [read,run] # allow read/run for nodes
by:
  group: foo`

const testAccAclPolicyConfig_basic = `
resource "rundeck_acl_policy" "test" {
	name = "TerraformBasicAcl.aclpolicy"
	policy = %q
}
`
