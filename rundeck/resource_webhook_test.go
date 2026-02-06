package rundeck

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccRundeckWebhook_basic(t *testing.T) {
	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "project", "terraform-acc-test-webhook"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "name", "test-webhook"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "user", "admin"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "roles", "admin"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "log-webhook-event"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("rundeck_webhook.test", "id"),
					resource.TestCheckResourceAttrSet("rundeck_webhook.test", "auth_token"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_withJob(t *testing.T) {
	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-job", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_withJob,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "project", "terraform-acc-test-webhook-job"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "name", "job-trigger-webhook"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "webhook-run-job"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("rundeck_webhook.test", "config.job_id"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_update(t *testing.T) {
	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-update", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_update_before,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "name", "test-webhook"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "enabled", "true"),
					resource.TestCheckNoResourceAttr("rundeck_webhook.test", "config.log_level"),
				),
			},
			{
				Config: testAccRundeckWebhookConfig_update_after,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "name", "test-webhook"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.log_level", "DEBUG"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_import(t *testing.T) {
	var webhookID string
	resourceName := "rundeck_webhook.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-import", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_import,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists(resourceName, &webhookID),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_token"}, // auth_token is not returned by API on import
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", resourceName)
					}
					project := rs.Primary.Attributes["project"]
					id := rs.Primary.ID
					return fmt.Sprintf("%s/%s", project, id), nil
				},
			},
		},
	})
}

func testAccWebhookCheckExists(resourceName string, webhookID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("webhook ID is not set")
		}

		project := rs.Primary.Attributes["project"]
		if project == "" {
			return fmt.Errorf("webhook project is not set")
		}

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		ctx := clients.ctx
		_, resp, err := clients.V2.WebhookAPI.Get(ctx, project, rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("error fetching webhook: %s, HTTP response: %+v", err, resp)
		}

		*webhookID = rs.Primary.ID
		return nil
	}
}

func testAccWebhookCheckDestroy(project string, webhookID *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if webhookID == nil || *webhookID == "" {
			return nil // No webhook to check
		}

		clients, err := getTestClients()
		if err != nil {
			return fmt.Errorf("failed to create test client: %s", err)
		}

		ctx := clients.ctx
		_, resp, err := clients.V2.WebhookAPI.Get(ctx, project, *webhookID).Execute()

		// If we get a 404, the webhook is properly destroyed
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}

		if err == nil {
			return fmt.Errorf("webhook still exists: %s", *webhookID)
		}

		return nil
	}
}

// Test configurations

const testAccRundeckWebhookConfig_basic = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook"
  description = "Test project for webhook acceptance tests"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "test-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "log-webhook-event"
}
`

const testAccRundeckWebhookConfig_withJob = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-job"
  description = "Test project for webhook with job"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-job.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "test-job-for-webhook"
  description  = "Test job triggered by webhook"

  command {
    shell_command = "echo 'triggered by webhook'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "job-trigger-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "webhook-run-job"
  
  config {
    job_id = rundeck_job.test.id
  }
}
`

const testAccRundeckWebhookConfig_update_before = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-update"
  description = "Test project for webhook update"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-update.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "test-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "log-webhook-event"
}
`

const testAccRundeckWebhookConfig_update_after = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-update"
  description = "Test project for webhook update"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-update.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "test-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "log-webhook-event"
  
  config {
    log_level = "DEBUG"
  }
}
`

const testAccRundeckWebhookConfig_import = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-import"
  description = "Test project for webhook import"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-import.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "import-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "log-webhook-event"
}
`

// Enterprise Tests - require RUNDECK_ENTERPRISE_TESTS=1

func TestAccRundeckWebhook_advancedRunJob(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: advanced-run-job plugin - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-advanced", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_advancedRunJob,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "advanced-run-job"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.batch_key", "data.alerts"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.event_id_key", "alertId"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.return_processing_info", "true"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_datadogRunJob(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: datadog-run-job plugin - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-datadog", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_datadogRunJob,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "datadog-run-job"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.batch_key", "body"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.event_id_key", "id"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_pagerdutyRunJob(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: pagerduty-run-job plugin - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-pagerduty", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_pagerdutyRunJob,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "pagerduty-run-job"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.batch_key", "messages"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.event_id_key", "id"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_pagerdutyV3RunJob(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: pagerduty-V3-run-job plugin - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-pagerduty-v3", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_pagerdutyV3RunJob,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "pagerduty-V3-run-job"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.event_id_key", "event.id"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_githubWebhook(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: github-webhook plugin - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-github", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_githubWebhook,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "github-webhook"),
					// secret is write-only, won't be returned by API
				),
			},
		},
	})
}

func TestAccRundeckWebhook_awsSnsWebhook(t *testing.T) {
	if os.Getenv("RUNDECK_ENTERPRISE_TESTS") != "1" {
		t.Skip("ENTERPRISE ONLY: aws-sns-webhook plugin - set RUNDECK_ENTERPRISE_TESTS=1")
	}

	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-aws-sns", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_awsSnsWebhook,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "aws-sns-webhook"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.auto_subscribe", "true"),
				),
			},
		},
	})
}

func TestAccRundeckWebhook_webhookRunJob(t *testing.T) {
	var webhookID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccWebhookCheckDestroy("terraform-acc-test-webhook-run-job-full", &webhookID),
		Steps: []resource.TestStep{
			{
				Config: testAccRundeckWebhookConfig_webhookRunJobFull,
				Check: resource.ComposeTestCheckFunc(
					testAccWebhookCheckExists("rundeck_webhook.test", &webhookID),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "event_plugin", "webhook-run-job"),
					resource.TestCheckResourceAttrSet("rundeck_webhook.test", "config.job_id"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.arg_string", "-opt1 value1 -opt2 value2"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.node_filter", "tags: linux"),
					resource.TestCheckResourceAttr("rundeck_webhook.test", "config.as_user", "webhook-user"),
				),
			},
		},
	})
}

// Enterprise Test Configurations

const testAccRundeckWebhookConfig_advancedRunJob = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-advanced"
  description = "Test project for advanced-run-job webhook"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-advanced.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "advanced-webhook-job"
  description  = "Job triggered by advanced webhook"

  command {
    shell_command = "echo 'Alert ID: @option.alertId@ Severity: @option.severity@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "advanced-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "advanced-run-job"
  
  config {
    batch_key              = "data.alerts"
    event_id_key           = "alertId"
    return_processing_info = true

    rules {
      name   = "Critical Alerts"
      job_id = rundeck_job.test.id

      job_options {
        name  = "alertId"
        value = "$.alertId"
      }

      job_options {
        name  = "severity"
        value = "$.severity"
      }

      conditions {
        path      = "data.severity"
        condition = "equals"
        value     = "critical"
      }
    }

    rules {
      name   = "Warning Alerts"
      job_id = rundeck_job.test.id

      conditions {
        path      = "data.severity"
        condition = "equals"
        value     = "warning"
      }

      conditions {
        path      = "data.environment"
        condition = "equals"
        value     = "production"
      }
    }
  }
}
`

const testAccRundeckWebhookConfig_datadogRunJob = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-datadog"
  description = "Test project for datadog-run-job webhook"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-datadog.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "datadog-alert-job"
  description  = "Job triggered by DataDog webhook"

  command {
    shell_command = "echo 'DataDog alert: @option.alert_id@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "datadog-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "datadog-run-job"
  
  config {
    batch_key    = "body"
    event_id_key = "id"

    rules {
      name   = "DataDog Error Alerts"
      job_id = rundeck_job.test.id

      job_options {
        name  = "alert_id"
        value = "$.id"
      }

      conditions {
        path      = "alert_type"
        condition = "equals"
        value     = "error"
      }
    }
  }
}
`

const testAccRundeckWebhookConfig_pagerdutyRunJob = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-pagerduty"
  description = "Test project for pagerduty-run-job webhook"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-pagerduty.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "pagerduty-incident-job"
  description  = "Job triggered by PagerDuty webhook"

  command {
    shell_command = "echo 'PagerDuty incident: @option.incident_id@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "pagerduty-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "pagerduty-run-job"
  
  config {
    batch_key    = "messages"
    event_id_key = "id"

    rules {
      name   = "PagerDuty Incident Trigger"
      job_id = rundeck_job.test.id

      job_options {
        name  = "incident_id"
        value = "$.id"
      }

      conditions {
        path      = "event"
        condition = "equals"
        value     = "incident.trigger"
      }
    }
  }
}
`

const testAccRundeckWebhookConfig_pagerdutyV3RunJob = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-pagerduty-v3"
  description = "Test project for pagerduty-V3-run-job webhook"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-pagerduty-v3.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "pagerduty-v3-incident-job"
  description  = "Job triggered by PagerDuty V3 webhook"

  command {
    shell_command = "echo 'PagerDuty V3 event: @option.event_id@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "pagerduty-v3-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "pagerduty-V3-run-job"
  
  config {
    event_id_key = "event.id"

    rules {
      name   = "PagerDuty V3 Incident Triggered"
      job_id = rundeck_job.test.id

      job_options {
        name  = "event_id"
        value = "$.event.id"
      }

      conditions {
        path      = "event.event_type"
        condition = "equals"
        value     = "incident.triggered"
      }
    }
  }
}
`

const testAccRundeckWebhookConfig_githubWebhook = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-github"
  description = "Test project for github-webhook"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-github.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "github-deploy-job"
  description  = "Job triggered by GitHub webhook"

  command {
    shell_command = "echo 'GitHub push to branch: @option.branch@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "github-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "github-webhook"
  
  config {
    secret = "my-github-webhook-secret-123"

    rules {
      name   = "GitHub Main Branch Push"
      job_id = rundeck_job.test.id

      job_options {
        name  = "branch"
        value = "$.ref"
      }

      conditions {
        path      = "ref"
        condition = "equals"
        value     = "refs/heads/main"
      }
    }
  }
}
`

const testAccRundeckWebhookConfig_awsSnsWebhook = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-aws-sns"
  description = "Test project for aws-sns-webhook"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-aws-sns.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "aws-cloudwatch-job"
  description  = "Job triggered by AWS SNS webhook"

  command {
    shell_command = "echo 'AWS CloudWatch alarm: @option.alarm_name@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "aws-sns-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "aws-sns-webhook"
  
  config {
    auto_subscribe = true

    rules {
      name   = "High CPU Alarm"
      job_id = rundeck_job.test.id

      job_options {
        name  = "alarm_name"
        value = "$.Message.AlarmName"
      }

      conditions {
        path      = "Message.AlarmName"
        condition = "equals"
        value     = "HighCPU"
      }
    }
  }
}
`

const testAccRundeckWebhookConfig_webhookRunJobFull = `
resource "rundeck_project" "test" {
  name        = "terraform-acc-test-webhook-run-job-full"
  description = "Test project for webhook-run-job with all options"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/terraform-acc-test-webhook-run-job-full.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "full-webhook-job"
  description  = "Job triggered by webhook with all options"

  command {
    shell_command = "echo 'Webhook options: opt1=@option.opt1@ opt2=@option.opt2@'"
  }
}

resource "rundeck_webhook" "test" {
  project      = rundeck_project.test.name
  name         = "full-webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "webhook-run-job"
  
  config {
    job_id      = rundeck_job.test.id
    arg_string  = "-opt1 value1 -opt2 value2"
    node_filter = "tags: linux"
    as_user     = "webhook-user"
  }
}
`
