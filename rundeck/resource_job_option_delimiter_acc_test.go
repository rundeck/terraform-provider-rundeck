package rundeck

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Each option must keep its own delimiter as options are added and moved.
func TestAccJob_option_values_list_delimiter_reorder(t *testing.T) {
	custom := `
  option {
    name                  = "custom"
    value_choices         = ["a,b", "c"]
    values_list_delimiter = "|"
  }`
	plain := `
  option {
    name          = "plain"
    value_choices = ["x|y", "z"]
  }`
	text := `
  option {
    name = "text"
  }`
	// Without its delimiter, custom needs choices free of commas.
	customUnset := `
  option {
    name          = "custom"
    value_choices = ["a", "c"]
  }`

	checkOpts := func(customIdx, plainIdx, textIdx int) resource.TestCheckFunc {
		c := fmt.Sprintf("option.%d.", customIdx)
		p := fmt.Sprintf("option.%d.", plainIdx)
		x := fmt.Sprintf("option.%d.", textIdx)
		return resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("rundeck_job.test", c+"name", "custom"),
			resource.TestCheckResourceAttr("rundeck_job.test", c+"values_list_delimiter", "|"),
			resource.TestCheckResourceAttr("rundeck_job.test", c+"value_choices.#", "2"),
			resource.TestCheckResourceAttr("rundeck_job.test", c+"value_choices.0", "a,b"),
			resource.TestCheckResourceAttr("rundeck_job.test", p+"name", "plain"),
			resource.TestCheckResourceAttr("rundeck_job.test", p+"values_list_delimiter", ","),
			resource.TestCheckResourceAttr("rundeck_job.test", p+"value_choices.#", "2"),
			resource.TestCheckResourceAttr("rundeck_job.test", p+"value_choices.0", "x|y"),
			resource.TestCheckResourceAttr("rundeck_job.test", x+"name", "text"),
			resource.TestCheckNoResourceAttr("rundeck_job.test", x+"values_list_delimiter"),
		)
	}

	initial := testAccJobConfig_option_delimiter(custom, text)
	added := testAccJobConfig_option_delimiter(custom, text, plain)
	swapped := testAccJobConfig_option_delimiter(plain, custom, text)
	moved := testAccJobConfig_option_delimiter(text, plain, custom)
	unset := testAccJobConfig_option_delimiter(text, plain, customUnset)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "option.0.values_list_delimiter", "|"),
					resource.TestCheckNoResourceAttr("rundeck_job.test", "option.1.values_list_delimiter"),
				),
			},
			{Config: initial, PlanOnly: true},
			{Config: added, Check: checkOpts(0, 2, 1)},
			{Config: added, PlanOnly: true},
			{Config: swapped, Check: checkOpts(1, 0, 2)},
			{Config: swapped, PlanOnly: true},
			{Config: moved, Check: checkOpts(2, 1, 0)},
			{Config: moved, PlanOnly: true},
			{
				Config: unset,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "option.2.name", "custom"),
					resource.TestCheckResourceAttr("rundeck_job.test", "option.2.values_list_delimiter", ","),
					resource.TestCheckResourceAttr("rundeck_job.test", "option.2.value_choices.#", "2"),
				),
			},
			{Config: unset, PlanOnly: true},
		},
	})
}

// Removing a delimiter that protects a comma in a choice is rejected at plan time.
func TestAccJobOptions_valuesListDelimiterRemoved(t *testing.T) {
	custom := testAccJobConfig_option_delimiter(`
  option {
    name                  = "custom"
    value_choices         = ["a,b", "c"]
    values_list_delimiter = "|"
  }`)
	removed := testAccJobConfig_option_delimiter(`
  option {
    name          = "custom"
    value_choices = ["a,b", "c"]
  }`)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{Config: custom},
			{
				Config:      removed,
				ExpectError: regexp.MustCompile("Value choice contains the values list delimiter"),
			},
			{
				Config:   custom,
				PlanOnly: true,
			},
		},
	})
}

// Delimiters Rundeck would not return as configured are rejected at plan time.
func TestAccJobOptions_valuesListDelimiterRejected(t *testing.T) {
	cases := map[string]struct {
		option string
		error  string
	}{
		"free text": {`
  option {
    name                  = "text"
    values_list_delimiter = "|"
  }`, "Values list delimiter without value choices"},
		"empty choices": {`
  option {
    name                  = "empty"
    value_choices         = []
    values_list_delimiter = "|"
  }`, "Values list delimiter without value choices"},
		"choices url": {`
  option {
    name                  = "remote"
    value_choices_url     = "http://localhost:4440/robots.txt"
    values_list_delimiter = "|"
  }`, "Values list delimiter without value choices"},
		"empty delimiter": {`
  option {
    name                  = "choices"
    value_choices         = ["a", "b"]
    values_list_delimiter = ""
  }`, "string length must be at least 1"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      testAccJobConfig_option_delimiter(tc.option),
						ExpectError: regexp.MustCompile(tc.error),
					},
				},
			})
		})
	}
}

func testAccJobConfig_option_delimiter(options ...string) string {
	opts := ""
	for _, o := range options {
		opts += o
	}

	return fmt.Sprintf(`
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-option-delimiter"
  description = "Project of the job exercising values_list_delimiter"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name      = rundeck_project.test.name
  name              = "option-delimiter-job"
  description       = "Job exercising values_list_delimiter"
  execution_enabled = true
%s
  command {
    shell_command = "echo hello"
  }
}
`, opts)
}
