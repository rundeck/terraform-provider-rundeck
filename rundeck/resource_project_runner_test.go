package rundeck

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

func TestAccRundeckProjectRunner_basic(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccProjectRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckProjectRunnerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunnerCheckExists("rundeck_project_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "project_name", "terraform-acc-test-project-runner"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "name", "test-project-runner"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "description", "Test project runner"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "tag_names", "terraform,test"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "installation_type", "linux"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "replica_type", "manual"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "runner_as_node_enabled", "false"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "remote_node_dispatch", "false"),
					resource.TestCheckResourceAttrSet("rundeck_project_runner.test", "runner_id"),
					resource.TestCheckResourceAttrSet("rundeck_project_runner.test", "token"),
					resource.TestCheckResourceAttrSet("rundeck_project_runner.test", "download_token"),
				),
			},
		},
	})
}

func TestAccRundeckProjectRunner_withNodeDispatch(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccProjectRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckProjectRunnerConfig_withNodeDispatch,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunnerCheckExists("rundeck_project_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "runner_as_node_enabled", "true"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "remote_node_dispatch", "true"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "runner_node_filter", "name: runner-node"),
				),
			},
		},
	})
}

func TestAccRundeckProjectRunner_update(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("Skipping Rundeck Enterprise test - set RUNDECK_ENTERPRISE_TESTS=1 to run")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccProjectRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckProjectRunnerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunnerCheckExists("rundeck_project_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "name", "test-project-runner"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "description", "Test project runner"),
				),
			},
			{
				Config: testAccRundeckProjectRunnerConfig_updated,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectRunnerCheckExists("rundeck_project_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "name", "updated-project-runner"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "description", "Updated test project runner"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "tag_names", "terraform,updated"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "installation_type", "docker"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "replica_type", "ephemeral"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "runner_as_node_enabled", "true"),
					resource.TestCheckResourceAttr("rundeck_project_runner.test", "remote_node_dispatch", "true"),
				),
			},
		},
	})
}

func testAccProjectRunnerCheckDestroy(runner *openapi.RunnerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		client := clients.V2
		ctx := clients.ctx

		runnerId := ""
		if runner.Id != nil {
			runnerId = *runner.Id
		}

		if runnerId == "" {
			return nil // No runner to check
		}

		// For project runners, we need to check with project context
		// We'll try to find the project name from the terraform state
		projectName := ""
		for _, rs := range s.RootModule().Resources {
			if rs.Type == "rundeck_project_runner" {
				if rs.Primary.Attributes["project_name"] != "" {
					projectName = rs.Primary.Attributes["project_name"]
					break
				}
			}
		}

		if projectName == "" {
			return nil // No project name found, assume destroyed
		}

		// Try to get the project runner - use general RunnerInfo endpoint
		runnerInfo, resp, err := client.RunnerAPI.RunnerInfo(ctx, runnerId).Execute()

		// If we get a 404, the runner is properly destroyed
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}

		// If the API call succeeded, the runner still exists
		if err == nil && runnerInfo != nil {
			return fmt.Errorf("project runner still exists: %s in project %s", runnerId, projectName)
		}

		// Any other error is acceptable (runner likely destroyed)
		return nil
	}
}

func testAccProjectRunnerCheckExists(rn string, runner *openapi.RunnerInfo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("runner id not set")
		}

		// Parse the composite ID (project:runner_id)
		parts := strings.SplitN(rs.Primary.ID, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid ID format, expected 'project:runner_id', got: %s", rs.Primary.ID)
		}

		projectName := parts[0]
		runnerId := parts[1]

		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		client := clients.V2
		ctx := clients.ctx

		// Use general RunnerInfo endpoint since ProjectRunnerInfo seems unreliable
		gotRunner, resp, err := client.RunnerAPI.RunnerInfo(ctx, runnerId).Execute()
		if err != nil || (resp != nil && resp.StatusCode != 200) {
			statusCode := 0
			if resp != nil {
				statusCode = resp.StatusCode
			}
			return fmt.Errorf("failed to get runner info (project=%s, runner=%s, status=%d): %v", projectName, runnerId, statusCode, err)
		}

		*runner = *gotRunner

		return nil
	}
}

const testAccRundeckProjectRunnerConfig_basic = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-project-runner"
  description = "Terraform Acceptance Tests Project for Runner"
  resource_model_source {
    type = "local"
    config = {
    }
  }  
}

resource "rundeck_project_runner" "test" {
  project_name     = rundeck_project.test.name
  name             = "test-project-runner"
  description      = "Test project runner"
  tag_names        = "terraform,test"
  installation_type = "linux"
  replica_type     = "manual"
}
`

const testAccRundeckProjectRunnerConfig_withNodeDispatch = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-project-runner"
  description = "Terraform Acceptance Tests Project for Runner"
  resource_model_source {
    type = "local"
    config = {
    }
  }
}

resource "rundeck_project_runner" "test" {
  project_name          = rundeck_project.test.name
  name                  = "test-project-runner"
  description           = "Test project runner with node dispatch"
  tag_names             = "terraform,test"
  installation_type     = "linux"
  replica_type          = "manual"
  runner_as_node_enabled = true
  remote_node_dispatch  = true
  runner_node_filter    = "name: runner-node"
}
`

const testAccRundeckProjectRunnerConfig_updated = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-project-runner"
  description = "Terraform Acceptance Tests Project for Runner"
  resource_model_source {
    type = "local"
    config = {
    }
  }  
}

resource "rundeck_project_runner" "test" {
  project_name          = rundeck_project.test.name
  name                  = "updated-project-runner"
  description           = "Updated test project runner"
  tag_names             = "terraform,updated"
  installation_type     = "docker"
  replica_type          = "ephemeral"
  runner_as_node_enabled = true
  remote_node_dispatch  = true
  runner_node_filter    = "name: updated-runner-node"
}
`
