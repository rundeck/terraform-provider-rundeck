package rundeck

import (
	"context"
	"fmt"
	"testing"

	"github.com/rundeck/go-rundeck/rundeck"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccProject_basic(t *testing.T) {
	var project rundeck.Project

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccProjectCheckDestroy(&project),
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
		client := testAccProvider.Meta().(*rundeck.BaseClient)
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

		client := testAccProvider.Meta().(*rundeck.BaseClient)
		ctx := context.Background()
		gotProject, err := client.ProjectGet(ctx, rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("error getting project: %s", err)
		}

		*project = gotProject

		return nil
	}
}

const testAccProjectConfig_basic = `
resource "rundeck_project" "main" {
  name = "terraform-acc-test-basic"
  description = "Terraform Acceptance Tests Basic Project"

  resource_model_source {
    type = "file"
    config = {
        format = "resourcexml"
        file = "/tmp/terraform-acc-tests.xml"
    }
  }

  extra_config = {
    "foo/bar" = "baz"
  }
}
`
