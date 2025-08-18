package rundeck

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	openapi "github.com/rundeck/go-rundeck-v2"
)

func TestAccRundeckSystemRunner_basic(t *testing.T) {
	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckSystemRunnerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "test-system-runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "description", "Test system runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "tag_names", "test,terraform"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "installation_type", "linux"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "replica_type", "manual"),
					resource.TestCheckResourceAttrSet("rundeck_system_runner.test", "runner_id"),
					resource.TestCheckResourceAttrSet("rundeck_system_runner.test", "token"),
					resource.TestCheckResourceAttrSet("rundeck_system_runner.test", "download_token"),
				),
			},
		},
	})
}

func TestAccRundeckSystemRunner_update(t *testing.T) {
	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckSystemRunnerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "test-system-runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "description", "Test system runner"),
				),
			},
			{
				Config: testAccRundeckSystemRunnerConfig_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "updated-system-runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "description", "Updated test system runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "tag_names", "updated,terraform"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "installation_type", "docker"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "replica_type", "ephemeral"),
				),
			},
		},
	})
}

func testAccSystemRunnerCheckDestroy(runner *openapi.RunnerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clients := testAccProvider.Meta().(*RundeckClients)
		client := clients.V2
		ctx := clients.ctx

		runnerId := ""
		if runner.Id != nil {
			runnerId = *runner.Id
		}

		if runnerId == "" {
			return nil // No runner to check
		}

		// Try to get the runner
		runnerInfo, resp, err := client.RunnerAPI.RunnerInfo(ctx, runnerId).Execute()

		// If we get a 404, the runner is properly destroyed
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}

		// If the API call succeeded, the runner still exists
		if err == nil && runnerInfo != nil {
			return fmt.Errorf("system runner still exists: %s", runnerId)
		}

		// Any other error is acceptable (runner likely destroyed)
		return nil
	}
}

func testAccSystemRunnerCheckExists(rn string, runner *openapi.RunnerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("runner id not set")
		}

		clients := testAccProvider.Meta().(*RundeckClients)
		client := clients.V2
		ctx := clients.ctx

		gotRunner, resp, err := client.RunnerAPI.RunnerInfo(ctx, rs.Primary.ID).Execute()

		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to get runner info: %v", err)
		}

		*runner = *gotRunner

		return nil
	}
}

const testAccRundeckSystemRunnerConfig_basic = `
resource "rundeck_system_runner" "test" {
  name             = "test-system-runner"
  description      = "Test system runner"
  tag_names        = "test,terraform"
  installation_type = "linux"
  replica_type     = "manual"
}
`

const testAccRundeckSystemRunnerConfig_updated = `
resource "rundeck_system_runner" "test" {
  name             = "updated-system-runner"
  description      = "Updated test system runner"
  tag_names        = "updated,terraform"
  installation_type = "docker"
  replica_type     = "ephemeral"
  
  assigned_projects = {
    "test-project" = ".*"
  }
}
`
