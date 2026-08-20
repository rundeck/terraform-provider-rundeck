package rundeck

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

// TestAccRundeckScmExport_basic requires a git remote reachable from the
// Rundeck *server* itself (not from wherever `go test` runs) - e.g. a bare
// repo over SSH - and an SSH private key (readable locally, on the machine
// running `go test`) authorized to push to it. Set RUNDECK_SCM_TEST_GIT_URL
// (an SSH-form git URL, e.g. git@host:org/repo.git) and
// RUNDECK_SCM_TEST_SSH_KEY_PATH (a local path to the matching PEM private
// key) to run this; it's skipped if either is unset. The key's contents are
// read by Terraform's own file() function at apply time, straight from
// disk - never interpolated through this Go test code. There is no
// Enterprise gate here - git/svn SCM plugins ship with OSS Rundeck (v15+).
func TestAccRundeckScmExport_basic(t *testing.T) {
	gitURL := os.Getenv("RUNDECK_SCM_TEST_GIT_URL")
	keyPath := os.Getenv("RUNDECK_SCM_TEST_SSH_KEY_PATH")
	if gitURL == "" || keyPath == "" {
		t.Skip("Skipping TestAccRundeckScmExport_basic: RUNDECK_SCM_TEST_GIT_URL and RUNDECK_SCM_TEST_SSH_KEY_PATH must both be set")
	}

	var config openapi.ScmProjectPluginConfig

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccScmCheckDestroy("export", &config),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckScmExportConfig_basic(gitURL, keyPath),
				Check: resource.ComposeTestCheckFunc(
					testAccScmCheckExists("rundeck_scm_export.test", "export", &config),
					resource.TestCheckResourceAttr("rundeck_scm_export.test", "type", "git-export"),
					resource.TestCheckResourceAttr("rundeck_scm_export.test", "config.url", gitURL),
					resource.TestCheckResourceAttr("rundeck_scm_export.test", "enabled", "true"),
				),
			},
		},
	})
}

func testAccScmCheckDestroy(integration string, config *openapi.ScmProjectPluginConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		project := ""
		if config.Project != nil {
			project = *config.Project
		}
		if project == "" {
			return nil
		}

		got, resp, err := clients.V2.SCMAPI.ApiProjectConfig(clients.ctx, project, integration).Execute()
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}
		if err == nil && got != nil && got.Enabled != nil && *got.Enabled {
			return fmt.Errorf("scm %s plugin still enabled for project: %s", integration, project)
		}
		return nil
	}
}

func testAccScmCheckExists(rn string, integration string, config *openapi.ScmProjectPluginConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}
		if rs.Primary.Attributes["project"] == "" {
			return fmt.Errorf("project not set")
		}

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		got, resp, err := clients.V2.SCMAPI.ApiProjectConfig(clients.ctx, rs.Primary.Attributes["project"], integration).Execute()
		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to get scm %s config: %v", integration, err)
		}

		*config = *got
		return nil
	}
}

func testAccRundeckScmExportConfig_basic(gitURL string, keyPath string) string {
	return fmt.Sprintf(`
resource "rundeck_private_key" "test" {
  path         = "terraform_acceptance_tests/scm_export_key"
  key_material = file(%q)
}

resource "rundeck_project" "test" {
  name        = "test-project-scm"
  description = "Terraform acceptance test project for SCM export"

  resource_model_source {
    type   = "local"
    config = {}
  }
}

resource "rundeck_scm_export" "test" {
  project = rundeck_project.test.name
  type    = "git-export"

  config = {
    url                   = %q
    dir                   = "/tmp/rundeck-scm-test-export"
    branch                = "main"
    createBranch          = "true"
    committerName         = "terraform-test"
    committerEmail        = "terraform-test@example.com"
    pathTemplate          = "$${job.group}$${job.name}-$${job.id}.xml"
    format                = "xml"
    sshPrivateKeyPath     = "keys/${rundeck_private_key.test.path}"
    strictHostKeyChecking = "no"
  }

  depends_on = [rundeck_private_key.test]
}
`, keyPath, gitURL)
}
