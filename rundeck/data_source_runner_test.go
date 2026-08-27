package rundeck

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

func TestAccRundeckRunnerDataSource_basic(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: Runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckRunnerDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttrPair("data.rundeck_runner.test", "runner_id", "rundeck_system_runner.test", "runner_id"),
					resource.TestCheckResourceAttr("data.rundeck_runner.test", "name", "test-datasource-runner"),
					resource.TestCheckResourceAttr("data.rundeck_runner.test", "description", "Test runner for data source"),
					resource.TestCheckResourceAttr("data.rundeck_runner.test", "tag_names", "terraform,test"),
				),
			},
		},
	})
}

func TestAccRundeckRunnersDataSource_basic(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: Runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var runner openapi.RunnerInfo

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccSystemRunnerCheckDestroy(&runner),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckRunnersDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccSystemRunnerCheckExists("rundeck_system_runner.test", &runner),
					resource.TestCheckResourceAttrSet("data.rundeck_runners.test", "runners.#"),
				),
			},
		},
	})
}

func TestAccRundeckRunnerTagsDataSource_basic(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: Runners (requires Rundeck 5.17.0+, API v56+) - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckRunnerTagsDataSourceConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.rundeck_runner_tags.test", "tags.%"),
				),
			},
		},
	})
}

const testAccRundeckRunnerDataSourceConfig_basic = `
resource "rundeck_system_runner" "test" {
  name              = "test-datasource-runner"
  description       = "Test runner for data source"
  tag_names         = "terraform,test"
  installation_type = "linux"
  replica_type      = "manual"
}

data "rundeck_runner" "test" {
  runner_id = rundeck_system_runner.test.runner_id
}
`

const testAccRundeckRunnersDataSourceConfig_basic = `
resource "rundeck_system_runner" "test" {
  name              = "test-datasource-runners"
  description       = "Test runner for runners data source"
  tag_names         = "terraform,test"
  installation_type = "linux"
  replica_type      = "manual"
}

data "rundeck_runners" "test" {
  tags = "terraform"

  depends_on = [rundeck_system_runner.test]
}
`

const testAccRundeckRunnerTagsDataSourceConfig_basic = `
resource "rundeck_project" "test" {
  name        = "test-project-runner-tags"
  description = "Terraform acceptance test project for runner_tags data source"

  resource_model_source {
    type   = "local"
    config = {}
  }
}

resource "rundeck_system_runner" "test" {
  name              = "test-datasource-runner-tags"
  description       = "Test runner for runner_tags data source"
  tag_names         = "terraform,test"
  installation_type = "linux"
  replica_type      = "manual"

  assigned_projects = {
    (rundeck_project.test.name) = "read"
  }
}

data "rundeck_runner_tags" "test" {
  project_name = rundeck_project.test.name

  depends_on = [rundeck_system_runner.test]
}
`
