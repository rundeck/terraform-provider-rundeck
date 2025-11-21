package rundeck

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccJob_scriptFields_comprehensive tests all script-related fields
// Based on Script_Samples.json - validates args, file_extension, expand_token_in_script_file
func TestAccJob_scriptFields_comprehensive(t *testing.T) {
	var jobID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_scriptFields_comprehensive,
				Check: resource.ComposeTestCheckFunc(
					testAccJobGetID("rundeck_job.test", &jobID),

					// Validate all script field mappings
					resource.TestCheckResourceAttr("rundeck_job.test", "name", "script-fields-test"),
					
					// Command 1: inline script with args and interpreter
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.description", "Inline Script with Args"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.inline_script", "echo \"Hello World\""),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_file_args", "--arguments=field"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.file_extension", ".ps1"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_interpreter.0.invocation_string", "pwsh -f ${scriptfile}"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_interpreter.0.args_quoted", "true"),

					// Command 2: script_url with expand_token
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.description", "Script URL with Expand Token"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_url", "https://raw.githubusercontent.com/fleschutz/PowerShell/refs/heads/main/scripts/check-file.ps1"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_file_args", "-argumentsField"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.expand_token_in_script_file", "true"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_interpreter.0.invocation_string", "pwsh -f ${scriptfile}"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_interpreter.0.args_quoted", "false"),

					// Command 3: script_file with all fields
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.description", "Script File with All Fields"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.script_file", "/tmp/test-script.sh"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.script_file_args", "--test-arg"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.file_extension", ".sh"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.expand_token_in_script_file", "true"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.script_interpreter.0.invocation_string", "bash ${scriptfile}"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.script_interpreter.0.args_quoted", "true"),

					// API Validation - verify field mappings
					testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
						sequence, ok := jobData["sequence"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("Sequence not found")
						}

						commands, ok := sequence["commands"].([]interface{})
						if !ok || len(commands) < 3 {
							return fmt.Errorf("Expected 3 commands, got %d", len(commands))
						}

						// Validate Command 1 - inline script
						cmd1 := commands[0].(map[string]interface{})
						if cmd1["args"] != "--arguments=field" {
							return fmt.Errorf("Command 1: Expected args='--arguments=field', got '%v'", cmd1["args"])
						}
						if cmd1["fileExtension"] != ".ps1" {
							return fmt.Errorf("Command 1: Expected fileExtension='.ps1', got '%v'", cmd1["fileExtension"])
						}
						if cmd1["script"] == nil {
							return fmt.Errorf("Command 1: Expected 'script' field (inline_script)")
						}
						if cmd1["interpreterArgsQuoted"] != true {
							return fmt.Errorf("Command 1: Expected interpreterArgsQuoted=true, got %v", cmd1["interpreterArgsQuoted"])
						}

						// Validate Command 2 - script_url
						cmd2 := commands[1].(map[string]interface{})
						if cmd2["scripturl"] == nil {
							return fmt.Errorf("Command 2: Expected 'scripturl' field")
						}
						if cmd2["args"] != "-argumentsField" {
							return fmt.Errorf("Command 2: Expected args='-argumentsField', got '%v'", cmd2["args"])
						}
						if cmd2["expandTokenInScriptFile"] != true {
							return fmt.Errorf("Command 2: Expected expandTokenInScriptFile=true, got %v", cmd2["expandTokenInScriptFile"])
						}
						if cmd2["interpreterArgsQuoted"] != false {
							return fmt.Errorf("Command 2: Expected interpreterArgsQuoted=false, got %v", cmd2["interpreterArgsQuoted"])
						}

						// Validate Command 3 - script_file
						cmd3 := commands[2].(map[string]interface{})
						if cmd3["scriptfile"] != "/tmp/test-script.sh" {
							return fmt.Errorf("Command 3: Expected scriptfile='/tmp/test-script.sh', got '%v'", cmd3["scriptfile"])
						}
						if cmd3["args"] != "--test-arg" {
							return fmt.Errorf("Command 3: Expected args='--test-arg', got '%v'", cmd3["args"])
						}

						return nil
					}),
				),
			},
			// Step 2: Verify no drift on re-apply
			{
				Config: testAccJobConfig_scriptFields_comprehensive,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_file_args", "--arguments=field"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.expand_token_in_script_file", "true"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.2.script_file", "/tmp/test-script.sh"),
				),
			},
		},
	})
}

// TestAccJob_errorHandler_comprehensive tests error handlers with all fields
// Based on Script_Samples.json - validates keep_going_on_success and error handler script fields
func TestAccJob_errorHandler_comprehensive(t *testing.T) {
	var jobID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_errorHandler_comprehensive,
				Check: resource.ComposeTestCheckFunc(
					testAccJobGetID("rundeck_job.test", &jobID),

					resource.TestCheckResourceAttr("rundeck_job.test", "name", "error-handler-test"),
					
					// Command with error handler
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.description", "Command with Error Handler"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.inline_script", "echo \"Main Command\""),
					
					// Error handler fields
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.error_handler.0.shell_command", "echo \"Rollback\""),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.error_handler.0.keep_going_on_success", "true"),

					// Command 2: Error handler with script fields
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.description", "Error Handler with Script"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.shell_command", "exit 1"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.error_handler.0.inline_script", "echo \"Error Recovery\""),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.error_handler.0.script_file_args", "--recover"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.error_handler.0.file_extension", ".sh"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.error_handler.0.keep_going_on_success", "false"),

					// API Validation
					testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
						sequence, ok := jobData["sequence"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("Sequence not found")
						}

						commands, ok := sequence["commands"].([]interface{})
						if !ok || len(commands) < 2 {
							return fmt.Errorf("Expected 2 commands, got %d", len(commands))
						}

						// Validate Command 1 error handler
						cmd1 := commands[0].(map[string]interface{})
						errorHandler1, ok := cmd1["errorhandler"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("Command 1: errorhandler not found")
						}
						if errorHandler1["exec"] != "echo \"Rollback\"" {
							return fmt.Errorf("Command 1: Expected exec='echo \"Rollback\"', got '%v'", errorHandler1["exec"])
						}
						if errorHandler1["keepgoingOnSuccess"] != true {
							return fmt.Errorf("Command 1: Expected keepgoingOnSuccess=true, got %v", errorHandler1["keepgoingOnSuccess"])
						}

						// Validate Command 2 error handler with script fields
						cmd2 := commands[1].(map[string]interface{})
						errorHandler2, ok := cmd2["errorhandler"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("Command 2: errorhandler not found")
						}
						if errorHandler2["script"] == nil {
							return fmt.Errorf("Command 2: Expected 'script' field in error handler")
						}
						if errorHandler2["args"] != "--recover" {
							return fmt.Errorf("Command 2: Expected args='--recover' in error handler, got '%v'", errorHandler2["args"])
						}
					if errorHandler2["fileExtension"] != ".sh" {
						return fmt.Errorf("Command 2: Expected fileExtension='.sh' in error handler, got '%v'", errorHandler2["fileExtension"])
					}
					// keepgoingOnSuccess should be false or nil (not present) when not explicitly set to true
					if kgos, ok := errorHandler2["keepgoingOnSuccess"].(bool); ok && kgos {
						return fmt.Errorf("Command 2: Expected keepgoingOnSuccess to be false or absent, got true")
					}

						return nil
					}),
				),
			},
			// Step 2: Verify no drift
			{
				Config: testAccJobConfig_errorHandler_comprehensive,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.error_handler.0.keep_going_on_success", "true"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.error_handler.0.script_file_args", "--recover"),
				),
			},
		},
	})
}

// TestAccJob_argsQuoted_variants tests args_quoted true and false
// Based on Args_Quoted.json - ensures both checkbox states work correctly
func TestAccJob_argsQuoted_variants(t *testing.T) {
	var jobID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_argsQuoted_variants,
				Check: resource.ComposeTestCheckFunc(
					testAccJobGetID("rundeck_job.test", &jobID),

					resource.TestCheckResourceAttr("rundeck_job.test", "name", "args-quoted-test"),
					
					// Command 1: args_quoted = true
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.description", "Args Quoted True"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_interpreter.0.args_quoted", "true"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_interpreter.0.invocation_string", "pwsh -f ${scriptfile}"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_file_args", "--test=yes"),
					
					// Command 2: args_quoted = false
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.description", "Args Quoted False"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_interpreter.0.args_quoted", "false"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_interpreter.0.invocation_string", "bash ${scriptfile}"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_file_args", "--test=no"),

					// API Validation
					testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
						sequence, ok := jobData["sequence"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("Sequence not found")
						}

						commands, ok := sequence["commands"].([]interface{})
						if !ok || len(commands) < 2 {
							return fmt.Errorf("Expected 2 commands, got %d", len(commands))
						}

						// Validate Command 1: interpreterArgsQuoted = true
						cmd1 := commands[0].(map[string]interface{})
						if cmd1["interpreterArgsQuoted"] != true {
							return fmt.Errorf("Command 1: Expected interpreterArgsQuoted=true, got %v", cmd1["interpreterArgsQuoted"])
						}
						if scriptInterp, ok := cmd1["scriptInterpreter"].(string); !ok || scriptInterp != "pwsh -f ${scriptfile}" {
							return fmt.Errorf("Command 1: Expected scriptInterpreter='pwsh -f ${scriptfile}', got '%v'", cmd1["scriptInterpreter"])
						}
						if cmd1["args"] != "--test=yes" {
							return fmt.Errorf("Command 1: Expected args='--test=yes', got '%v'", cmd1["args"])
						}

						// Validate Command 2: interpreterArgsQuoted = false
						cmd2 := commands[1].(map[string]interface{})
						if cmd2["interpreterArgsQuoted"] != false {
							return fmt.Errorf("Command 2: Expected interpreterArgsQuoted=false, got %v", cmd2["interpreterArgsQuoted"])
						}
						if scriptInterp, ok := cmd2["scriptInterpreter"].(string); !ok || scriptInterp != "bash ${scriptfile}" {
							return fmt.Errorf("Command 2: Expected scriptInterpreter='bash ${scriptfile}', got '%v'", cmd2["scriptInterpreter"])
						}
						if cmd2["args"] != "--test=no" {
							return fmt.Errorf("Command 2: Expected args='--test=no', got '%v'", cmd2["args"])
						}

						return nil
					}),
				),
			},
			// Step 2: Verify no drift
			{
				Config: testAccJobConfig_argsQuoted_variants,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rundeck_job.test", "command.0.script_interpreter.0.args_quoted", "true"),
					resource.TestCheckResourceAttr("rundeck_job.test", "command.1.script_interpreter.0.args_quoted", "false"),
				),
			},
		},
	})
}

// Test configurations

const testAccJobConfig_scriptFields_comprehensive = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourceyaml"
        file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "script-fields-test"
  description = "Comprehensive test of all script-related fields"

  command {
    description = "Inline Script with Args"
    inline_script = "echo \"Hello World\""
    script_file_args = "--arguments=field"
    file_extension = ".ps1"
    script_interpreter {
      args_quoted       = true
      invocation_string = "pwsh -f $${scriptfile}"
    }
  }

  command {
    description = "Script URL with Expand Token"
    script_url = "https://raw.githubusercontent.com/fleschutz/PowerShell/refs/heads/main/scripts/check-file.ps1"
    script_file_args = "-argumentsField"
    expand_token_in_script_file = true
    script_interpreter {
      args_quoted       = false
      invocation_string = "pwsh -f $${scriptfile}"
    }
  }

  command {
    description = "Script File with All Fields"
    script_file = "/tmp/test-script.sh"
    script_file_args = "--test-arg"
    file_extension = ".sh"
    expand_token_in_script_file = true
    script_interpreter {
      args_quoted       = true
      invocation_string = "bash $${scriptfile}"
    }
  }
}
`

const testAccJobConfig_errorHandler_comprehensive = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourceyaml"
        file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "error-handler-test"
  description = "Comprehensive test of error handlers"

  command {
    description = "Command with Error Handler"
    inline_script = "echo \"Main Command\""
    error_handler {
      shell_command = "echo \"Rollback\""
      keep_going_on_success = true
    }
  }

  command {
    description = "Error Handler with Script"
    shell_command = "exit 1"
    error_handler {
      inline_script = "echo \"Error Recovery\""
      script_file_args = "--recover"
      file_extension = ".sh"
      keep_going_on_success = false
    }
  }
}
`

const testAccJobConfig_argsQuoted_variants = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-job"
  description = "parent project for job acceptance tests"

  resource_model_source {
    type = "file"
    config = {
        format = "resourceyaml"
        file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = "${rundeck_project.test.name}"
  name = "args-quoted-test"
  description = "Test args_quoted true and false"

  command {
    description = "Args Quoted True"
    inline_script = "echo \"Test\""
    script_file_args = "--test=yes"
    file_extension = ".sh"
    script_interpreter {
      args_quoted       = true
      invocation_string = "pwsh -f $${scriptfile}"
    }
  }

  command {
    description = "Args Quoted False"
    inline_script = "echo \"Test\""
    script_file_args = "--test=no"
    file_extension = ".sh"
    script_interpreter {
      args_quoted       = false
      invocation_string = "bash $${scriptfile}"
    }
  }
}
`

