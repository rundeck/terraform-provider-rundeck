package rundeck

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccJob_ComplexIntegration tests a complex job configuration and validates
// that all components are correctly stored in Rundeck by querying the API directly
func TestAccJob_ComplexIntegration(t *testing.T) {
	var jobID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_ComplexIntegration,
				Check: resource.ComposeTestCheckFunc(
					// Capture job ID for API validation
					testAccJobGetID("rundeck_job.complex_test", &jobID),

					// Standard Terraform state checks
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "name", "complex-integration-test"),
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "description", "Complex job for integration testing"),
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "schedule", "0 0 12 ? * * *"),

					// Validate orchestrator
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "orchestrator.0.type", "maxPercentage"),
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "orchestrator.0.percent", "75"),

					// Validate log limit
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "log_limit.0.output", "10MB"),
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "log_limit.0.action", "halt"),

					// Validate options
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "option.0.name", "environment"),
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "option.0.require_predefined_choice", "true"),

					// Validate notifications (alphabetical order)
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "notification.0.type", "on_failure"),
					resource.TestCheckResourceAttr("rundeck_job.complex_test", "notification.1.type", "on_success"),

					// API validation - verify actual stored values
					testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
						// Validate schedule is stored correctly
						if schedule, ok := jobData["schedule"].(map[string]interface{}); ok {
							if time, ok := schedule["time"].(map[string]interface{}); ok {
								if hour := time["hour"]; hour != "12" {
									return fmt.Errorf("Expected schedule hour=12, got %v", hour)
								}
							} else {
								return fmt.Errorf("Schedule missing time field")
							}
						} else {
							return fmt.Errorf("Schedule not found in API response")
						}

						// Validate orchestrator is stored
						if orch, ok := jobData["orchestrator"].(map[string]interface{}); ok {
							if orchType := orch["type"]; orchType != "maxPercentage" {
								return fmt.Errorf("Expected orchestrator type=maxPercentage, got %v", orchType)
							}
							if config, ok := orch["configuration"].(map[string]interface{}); ok {
								if percent := config["percent"]; percent != "75" {
									return fmt.Errorf("Expected orchestrator percent=75, got %v", percent)
								}
							} else {
								return fmt.Errorf("Orchestrator missing configuration")
							}
						} else {
							return fmt.Errorf("Orchestrator not found in API response")
						}

						// Validate log limit is stored
						if logLimit, ok := jobData["loglimit"].(string); ok {
							if logLimit != "10MB" {
								return fmt.Errorf("Expected loglimit=10MB, got %v", logLimit)
							}
						} else {
							return fmt.Errorf("Log limit not found in API response")
						}

						// Validate log limit action
						if action, ok := jobData["loglimitAction"].(string); ok {
							if action != "halt" {
								return fmt.Errorf("Expected loglimitAction=halt, got %v", action)
							}
						} else {
							return fmt.Errorf("Log limit action not found in API response")
						}

						// Validate notifications are stored (should be in object format from API)
						if notifications, ok := jobData["notification"].(map[string]interface{}); ok {
							// Check onsuccess notification
							if onsuccess, ok := notifications["onsuccess"].(map[string]interface{}); ok {
								if email, ok := onsuccess["email"].(map[string]interface{}); ok {
									if recipients := email["recipients"]; recipients != "success@example.com" {
										return fmt.Errorf("Expected success email recipients=success@example.com, got %v", recipients)
									}
								} else {
									return fmt.Errorf("onsuccess notification missing email")
								}
							} else {
								return fmt.Errorf("onsuccess notification not found")
							}

							// Check onfailure notification
							if onfailure, ok := notifications["onfailure"].(map[string]interface{}); ok {
								if email, ok := onfailure["email"].(map[string]interface{}); ok {
									if recipients := email["recipients"]; recipients != "failure@example.com" {
										return fmt.Errorf("Expected failure email recipients=failure@example.com, got %v", recipients)
									}
								} else {
									return fmt.Errorf("onfailure notification missing email")
								}
							} else {
								return fmt.Errorf("onfailure notification not found")
							}
						} else {
							return fmt.Errorf("Notifications not found in API response")
						}

						// Validate options are stored
						if options, ok := jobData["options"].([]interface{}); ok {
							if len(options) != 1 {
								return fmt.Errorf("Expected 1 option, got %d", len(options))
							}
							option := options[0].(map[string]interface{})
							if name := option["name"]; name != "environment" {
								return fmt.Errorf("Expected option name=environment, got %v", name)
							}
							// Check if enforcedValues is set (or inferred from values presence)
							if values, hasValues := option["values"]; hasValues {
								valuesList, ok := values.([]interface{})
								if !ok || len(valuesList) == 0 {
									return fmt.Errorf("Expected option to have values")
								}
							} else {
								return fmt.Errorf("Expected option to have values for enforcement")
							}
						} else {
							return fmt.Errorf("Options not found in API response")
						}

						return nil
					}),
				),
			},
		},
	})
}

// TestAccJob_NotificationIntegration specifically tests notification API format handling
func TestAccJob_NotificationIntegration(t *testing.T) {
	var jobID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccJobCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccJobConfig_NotificationIntegration,
				Check: resource.ComposeTestCheckFunc(
					testAccJobGetID("rundeck_job.notification_test", &jobID),

					// Validate via API that notifications are stored correctly
					testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
						notifications, ok := jobData["notification"].(map[string]interface{})
						if !ok {
							return fmt.Errorf("Notifications not found in API response")
						}

						// Validate email notification
						if onsuccess, ok := notifications["onsuccess"].(map[string]interface{}); ok {
							if email, ok := onsuccess["email"].(map[string]interface{}); ok {
								if recipients := email["recipients"]; recipients != "test@example.com" {
									return fmt.Errorf("Email recipients mismatch: got %v", recipients)
								}
								if subject := email["subject"]; subject != "Job Success" {
									return fmt.Errorf("Email subject mismatch: got %v", subject)
								}
								// Validate attach_log handling
								if attachLog, ok := email["attachLog"].(bool); ok {
									if attachLog != true {
										return fmt.Errorf("Expected attachLog=true, got %v", attachLog)
									}
								} else {
									return fmt.Errorf("attachLog not found in API response")
								}
							} else {
								return fmt.Errorf("Email configuration not found")
							}
						} else {
							return fmt.Errorf("onsuccess notification not found")
						}

						// Validate webhook notification
						if onstart, ok := notifications["onstart"].(map[string]interface{}); ok {
							// Webhook fields should be at top level, not nested
							if urls, ok := onstart["urls"].(string); ok {
								if urls != "https://webhook.example.com/notify" {
									return fmt.Errorf("Webhook URL mismatch: got %v", urls)
								}
							} else {
								return fmt.Errorf("Webhook URLs not found at top level")
							}
							if format, ok := onstart["format"].(string); ok {
								if format != "json" {
									return fmt.Errorf("Webhook format mismatch: got %v", format)
								}
							} else {
								return fmt.Errorf("Webhook format not found at top level")
							}
							if method, ok := onstart["httpMethod"].(string); ok {
								if method != "post" {
									return fmt.Errorf("Webhook method mismatch: got %v", method)
								}
							} else {
								return fmt.Errorf("Webhook httpMethod not found at top level")
							}
						} else {
							return fmt.Errorf("onstart notification not found")
						}

						return nil
					}),
				),
			},
		},
	})
}

// Helper function to capture job ID from Terraform state
func testAccJobGetID(resourceName string, jobID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource %s not found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No job ID is set")
		}

		*jobID = rs.Primary.ID
		return nil
	}
}

// Helper function to validate job via direct API call
func testAccJobValidateAPI(jobID *string, validateFn func(map[string]interface{}) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *jobID == "" {
			return fmt.Errorf("Job ID not set")
		}

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("Failed to get test clients: %s", err)
		}

		// Use GetJobJSON to get job data directly from API
		jobJSON, err := GetJobJSON(clients.V1, *jobID)
		if err != nil {
			return fmt.Errorf("Failed to get job from API: %s", err)
		}

		// Convert JobJSON struct to map for validation
		jobBytes, _ := json.Marshal(jobJSON)
		var jobData map[string]interface{}
		if err := json.Unmarshal(jobBytes, &jobData); err != nil {
			return fmt.Errorf("Failed to parse job JSON: %s", err)
		}

		// Run custom validation function
		if err := validateFn(jobData); err != nil {
			// Include actual API response in error for debugging
			prettyJSON, _ := json.MarshalIndent(jobData, "", "  ")
			return fmt.Errorf("API validation failed: %s\n\nActual API response:\n%s", err, string(prettyJSON))
		}

		return nil
	}
}

const testAccJobConfig_ComplexIntegration = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-complex-integration"
  description = "Complex integration test project"

  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "complex_test" {
  project_name = rundeck_project.test.name
  name = "complex-integration-test"
  description = "Complex job for integration testing"
  execution_enabled = true
  schedule = "0 0 12 ? * * *"
  schedule_enabled = true
  node_filter_query = "tags: test"
  max_thread_count = 4

  command {
    shell_command = "echo 'Testing complex job'"
  }

  orchestrator {
    type = "maxPercentage"
    percent = 75
  }

  log_limit {
    output = "10MB"
    action = "halt"
    status = "failed"
  }

  option {
    name = "environment"
    description = "Target environment"
    required = true
    value_choices = ["dev", "staging", "prod"]
    require_predefined_choice = true
    default_value = "dev"
  }

  # Notifications in alphabetical order: onfailure, onsuccess
  notification {
    type = "on_failure"
    email {
      recipients = ["failure@example.com"]
      subject = "Job Failed"
      attach_log = true
    }
  }

  notification {
    type = "on_success"
    email {
      recipients = ["success@example.com"]
      subject = "Job Succeeded"
    }
  }
}
`

const testAccJobConfig_NotificationIntegration = `
resource "rundeck_project" "test" {
  name = "terraform-acc-test-notification-integration"
  description = "Notification integration test project"

  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file = "/tmp/terraform-acc-tests.yaml"
    }
  }
}

resource "rundeck_job" "notification_test" {
  project_name = rundeck_project.test.name
  name = "notification-integration-test"
  description = "Testing notification API format handling"
  execution_enabled = true

  command {
    shell_command = "echo 'Testing notifications'"
  }

  # Notifications in alphabetical order: onstart, onsuccess
  
  # Test webhook notification (fields at top level)
  notification {
    type = "on_start"
    format = "json"
    http_method = "post"
    webhook_urls = ["https://webhook.example.com/notify"]
  }

  # Test email notification with attach_log
  notification {
    type = "on_success"
    email {
      recipients = ["test@example.com"]
      subject = "Job Success"
      attach_log = true
    }
  }
}
`
