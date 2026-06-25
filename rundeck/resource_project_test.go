package rundeck

import (
	"context"
	"fmt"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccProject_basic(t *testing.T) {
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccProjectCheckDestroy(&project),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectCheckExists("rundeck_project.main", &project),
					func(s *terraform.State) error {
						projectConfig := project.Config.(map[string]interface{})

						if expected := "terraform-acc-test-basic"; *project.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, project.Name)
						}
						if expected := "baz"; projectConfig["foo.bar"] != expected {
							return fmt.Errorf("wrong foo.bar config; expected %v, got %v", expected, projectConfig["foo.bar"])
						}
						if expected := "file"; projectConfig["resources.source.1.type"] != expected {
							return fmt.Errorf("wrong resources.source.1.type config; expected %v, got %v", expected, projectConfig["resources.source.1.type"])
						}
						return nil
					},
				),
			},
		},
	})
}

func testAccProjectCheckDestroy(project *rundeck.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		client := clients.V1
		ctx := context.Background()
		resp, err := client.ProjectGet(ctx, *project.Name)
		if err == nil && resp.StatusCode == 200 {
			return fmt.Errorf("project still exists")
		}
		if resp.StatusCode == 404 {
			return nil
		}

		return fmt.Errorf("Error checking if project destroyed: (%v)", err)
	}
}

func testAccProjectCheckExists(rn string, project *rundeck.Project) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("project id not set")
		}

		// Create client from environment variables for test verification
		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		client := clients.V1
		ctx := context.Background()
		gotProject, err := client.ProjectGet(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting project: %s", err)
		}

		*project = gotProject

		return nil
	}
}

func TestAccProject_localSourceNoConfig(t *testing.T) {
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccProjectCheckDestroy(&project),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig_localSourceNoConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectCheckExists("rundeck_project.test", &project),
					func(s *terraform.State) error {
						if expected := "terraform-acc-test-local-no-config"; *project.Name != expected {
							return fmt.Errorf("wrong name; expected %v, got %v", expected, project.Name)
						}
						return nil
					},
				),
			},
			// Verify no plan drift on refresh
			{
				RefreshState: true,
				PlanOnly:     true,
			},
		},
	})
}

const testAccProjectConfig_localSourceNoConfig = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-local-no-config"
  description = "Test project for local source without config"

  resource_model_source {
    type = "local"
  }
}
`

// TestAccProject_SSHKeyFilePath tests that ssh_key_file_path is correctly stored
// and does not produce drift after apply (regression for project.ssh-keypath mapping bug).
func TestAccProject_SSHKeyFilePath(t *testing.T) {
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccProjectCheckDestroy(&project),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig_SSHKeyFilePath,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectCheckExists("rundeck_project.test", &project),
					resource.TestCheckResourceAttr("rundeck_project.test", "ssh_key_file_path", "/var/lib/rundeck/.ssh/id_rsa"),
					resource.TestCheckNoResourceAttr("rundeck_project.test", "ssh_key_storage_path"),
				),
			},
			// Verify no plan drift on refresh — this is the core regression check
			{
				RefreshState: true,
				PlanOnly:     true,
			},
		},
	})
}

const testAccProjectConfig_SSHKeyFilePath = `
resource "rundeck_project" "test" {
  name             = "terraform-acc-test-ssh-key-file-path"
  description      = "Test project for ssh_key_file_path drift regression"
  ssh_key_file_path = "/var/lib/rundeck/.ssh/id_rsa"

  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}
`

// TestAccProject_NullConfigValues is a regression test for #248: a null element
// in a resource_model_source config map must be treated as omitted without
// producing an "inconsistent result after apply" error or refresh drift.
func TestAccProject_NullConfigValues(t *testing.T) {
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccProjectCheckDestroy(&project),
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig_NullConfigValues,
				Check: resource.ComposeTestCheckFunc(
					testAccProjectCheckExists("rundeck_project.test", &project),
					resource.TestCheckResourceAttr("rundeck_project.test", "resource_model_source.0.config.format", "resourceyaml"),
				),
			},
			// Core regression: a null config element must not cause drift on refresh.
			{
				RefreshState: true,
				PlanOnly:     true,
			},
		},
	})
}

const testAccProjectConfig_NullConfigValues = `
variable "optional_file" {
  type    = string
  default = null
}

resource "rundeck_project" "test" {
  name        = "terraform-acc-test-null-config"
  description = "Test project for null config element handling"

  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
      # Null element must be treated as omitted (regression for #248)
      generateFileAutomatically = var.optional_file
    }
  }
}
`

const testAccProjectConfig_basic = `
resource "rundeck_project" "main" {
  name = "terraform-acc-test-basic"
  description = "Terraform Acceptance Tests Basic Project"

  resource_model_source {
    type = "file"
    config = {
        format = "resourceyaml"
        file = "/tmp/terraform-acc-tests.yaml"
    }
  }

  extra_config = {
    "foo.bar" = "baz"
  }
}
`
