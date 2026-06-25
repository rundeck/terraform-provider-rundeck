package rundeck

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

func TestAccRundeckSystemRunner_basic(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: System runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckSystemRunnerConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "test-system-runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "description", "Test system runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "tag_names", "terraform,test"),
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
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: System runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
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
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "tag_names", "terraform,updated"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "installation_type", "docker"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "replica_type", "ephemeral"),
				),
			},
		},
	})
}

func testAccSystemRunnerCheckDestroy(runner *openapi.RunnerInfo) resource.TestCheckFunc {
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

		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

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
  tag_names        = "terraform,test"
  installation_type = "linux"
  replica_type     = "manual"
}
`

const testAccRundeckSystemRunnerConfig_updated = `
resource "rundeck_project" "test" {
  name        = "test-project"
  description = "Terraform acceptance test project for system runner"

  resource_model_source {
    type = "local"
    config = {
    }
  }
}

resource "rundeck_system_runner" "test" {
  name             = "updated-system-runner"
  description      = "Updated test system runner"
  tag_names        = "terraform,updated"
  installation_type = "docker"
  replica_type     = "ephemeral"

  assigned_projects = {
    (rundeck_project.test.name) = "admin"
  }
}
`

func TestAccRundeckSystemRunner_withAssignedProjectsConfig(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: System runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckSystemRunnerConfig_withAssignedProjectsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "test-runner-with-config"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "description", "Test runner with project config"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.access_level", "admin"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.runner_as_node_enabled", "true"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.remote_node_dispatch", "true"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.runner_node_filter", "tags: RUNNER"),
				),
			},
			{
				Config: testAccRundeckSystemRunnerConfig_withAssignedProjectsConfigUpdated,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.access_level", "admin"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.runner_as_node_enabled", "true"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.remote_node_dispatch", "false"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.runner_node_filter", "tags: DEFAULT"),
				),
			},
			// Verify no plan drift on refresh (computed bool defaults + state-preserved dispatch settings)
			{
				RefreshState: true,
				PlanOnly:     true,
			},
		},
	})
}

func TestAccRundeckSystemRunner_backwardCompatAssignedProjects(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: System runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckSystemRunnerConfig_legacyAssignedProjects,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "test-legacy-runner"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects.test-project", "admin"),
				),
			},
		},
	})
}

func TestAccRundeckSystemRunner_precedenceAssignedProjectsConfig(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: System runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckSystemRunnerConfig_precedence,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "name", "test-precedence-runner"),
					// assigned_projects_config should take precedence
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.access_level", "admin"),
					resource.TestCheckResourceAttr("rundeck_system_runner.test", "assigned_projects_config.test-project.runner_as_node_enabled", "true"),
				),
			},
		},
	})
}

const testAccRundeckSystemRunnerConfig_withAssignedProjectsConfig = `
resource "rundeck_project" "test" {
  name        = "test-project"
  description = "Terraform acceptance test project for system runner"

  resource_model_source {
    type = "local"
    config = {
    }
  }
}

resource "rundeck_system_runner" "test" {
  name             = "test-runner-with-config"
  description      = "Test runner with project config"
  tag_names        = "terraform,test"
  installation_type = "kubernetes"
  replica_type     = "ephemeral"

  assigned_projects_config = {
    (rundeck_project.test.name) = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: RUNNER"
    }
  }
}
`

const testAccRundeckSystemRunnerConfig_withAssignedProjectsConfigUpdated = `
resource "rundeck_project" "test" {
  name        = "test-project"
  description = "Terraform acceptance test project for system runner"

  resource_model_source {
    type = "local"
    config = {
    }
  }
}

resource "rundeck_system_runner" "test" {
  name             = "test-runner-with-config"
  description      = "Test runner with project config"
  tag_names        = "terraform,test"
  installation_type = "kubernetes"
  replica_type     = "ephemeral"

  assigned_projects_config = {
    (rundeck_project.test.name) = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = false
      runner_node_filter       = "tags: DEFAULT"
    }
  }
}
`

const testAccRundeckSystemRunnerConfig_legacyAssignedProjects = `
resource "rundeck_project" "test" {
  name        = "test-project"
  description = "Terraform acceptance test project for system runner"

  resource_model_source {
    type = "local"
    config = {
    }
  }
}

resource "rundeck_system_runner" "test" {
  name             = "test-legacy-runner"
  description      = "Test legacy assigned_projects"
  tag_names        = "terraform,test"
  installation_type = "linux"
  replica_type     = "manual"

  assigned_projects = {
    (rundeck_project.test.name) = "admin"
  }
}
`

const testAccRundeckSystemRunnerConfig_precedence = `
resource "rundeck_project" "test" {
  name        = "test-project"
  description = "Terraform acceptance test project for system runner"

  resource_model_source {
    type = "local"
    config = {
    }
  }
}

resource "rundeck_system_runner" "test" {
  name             = "test-precedence-runner"
  description      = "Test precedence of assigned_projects_config"
  tag_names        = "terraform,test"
  installation_type = "linux"
  replica_type     = "manual"

  # This should be overridden by assigned_projects_config
  assigned_projects = {
    (rundeck_project.test.name) = "execute"
  }

  # This should take precedence
  assigned_projects_config = {
    (rundeck_project.test.name) = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = false
      runner_node_filter       = "tags: RUNNER"
    }
  }
}
`
