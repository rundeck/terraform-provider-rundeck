---
layout: "rundeck"
page_title: "Rundeck: rundeck_webhook"
sidebar_current: "docs-rundeck-resource-webhook"
description: |-
  The rundeck_webhook resource allows Rundeck webhooks to be managed by Terraform.
---

# rundeck\_webhook

Enable external systems to trigger automation in Rundeck through HTTP webhooks. Webhooks support job execution, event logging, and advanced integrations with platforms like GitHub, PagerDuty, DataDog, and AWS SNS.

**Requirements:** Requires Rundeck with API v33+ (introduced in Rundeck 3.3.0). The provider defaults to API v56. Enterprise plugins require Rundeck Enterprise.

**Key capabilities:** Simple event logging, job triggering, conditional job execution based on webhook payload, batch processing, and third-party platform integrations.

## Important Notes

### Immutable Fields

The following fields **cannot** be updated after creation and will force resource recreation:
- `name` - Webhook name
- `roles` - Authorization roles
- `enabled` - Whether webhook is enabled

### Auth Token Handling

The `auth_token` is only returned during creation and is **not** retrievable later via the API. Store it securely after creation or use Terraform outputs to capture it.

### Known API Limitations

- The Rundeck API may not reliably return updated `config` values after an update
- Config changes are applied but may require a manual refresh to verify
- This is a known Rundeck API behavior, not a provider bug

## Example Usage

### Basic Webhook with Event Logging

```hcl
resource "rundeck_project" "example" {
  name        = "example-project"
  description = "Example project for webhooks"

  resource_model_source {
    type = "file"
    config = {
      format                    = "resourcexml"
      file                      = "/tmp/example-resources.xml"
      generateFileAutomatically = "true"
      includeServerNode         = "true"
    }
  }
}

resource "rundeck_webhook" "logging" {
  project      = rundeck_project.example.name
  name         = "log-events"
  user         = "automation"
  roles        = "webhook,user"
  enabled      = true
  event_plugin = "log-webhook-event"
  
  config {
    log_level = "INFO"  # DEBUG, INFO, WARN, ERROR
  }
}

# Output the webhook URL and auth token
output "webhook_url" {
  value     = "https://rundeck.example.com/api/56/webhook/${rundeck_webhook.logging.auth_token}"
  sensitive = true
}
```

### Simple Job Trigger Webhook

```hcl
resource "rundeck_job" "deploy" {
  project_name = rundeck_project.example.name
  name         = "deploy-application"
  description  = "Deploy application to production"

  command {
    shell_command = "ansible-playbook deploy.yml"
  }
}

resource "rundeck_webhook" "deploy_trigger" {
  project      = rundeck_project.example.name
  name         = "deploy-webhook"
  user         = "ci-bot"
  roles        = "admin,webhook"
  enabled      = true
  event_plugin = "webhook-run-job"
  
  config {
    job_id      = rundeck_job.deploy.id
    arg_string  = "-opt1 value1 -opt2 value2"
    node_filter = "tags: linux"
    as_user     = "deploy-bot"
  }
}
```

### Advanced Job Trigger with Rules (Enterprise)

```hcl
resource "rundeck_job" "critical_alert_handler" {
  project_name = rundeck_project.example.name
  name         = "handle-critical-alert"
  description  = "Handle critical severity alerts"

  command {
    shell_command = "echo 'Alert ID: @option.alert_id@ Severity: @option.severity@'"
  }
}

resource "rundeck_webhook" "advanced_alerts" {
  project      = rundeck_project.example.name
  name         = "alert-handler"
  user         = "automation"
  roles        = "admin"
  enabled      = true
  event_plugin = "advanced-run-job"
  
  config {
    batch_key              = "data.alerts"
    event_id_key           = "alert_id"
    return_processing_info = true

    # Critical alerts rule
    rules {
      name   = "Critical Alerts"
      job_id = rundeck_job.critical_alert_handler.id

      job_options {
        name  = "alert_id"
        value = "$.alert_id"
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

      conditions {
        path      = "data.environment"
        condition = "equals"
        value     = "production"
      }
    }

    # Warning alerts rule (different job, different conditions)
    rules {
      name   = "Warning Alerts"
      job_id = rundeck_job.critical_alert_handler.id

      conditions {
        path      = "data.severity"
        condition = "equals"
        value     = "warning"
      }
    }
  }
}
```

### GitHub Integration (Enterprise)

```hcl
resource "rundeck_job" "github_deploy" {
  project_name = rundeck_project.example.name
  name         = "github-triggered-deploy"
  description  = "Deploy on GitHub push"

  command {
    shell_command = "git pull && ./deploy.sh"
  }
}

resource "rundeck_webhook" "github" {
  project      = rundeck_project.example.name
  name         = "github-webhook"
  user         = "github-bot"
  roles        = "admin"
  enabled      = true
  event_plugin = "github-webhook"
  
  config {
    secret = var.github_webhook_secret  # GitHub webhook secret for authentication

    rules {
      name   = "Main Branch Push"
      job_id = rundeck_job.github_deploy.id

      job_options {
        name  = "branch"
        value = "$.ref"
      }

      job_options {
        name  = "commit_sha"
        value = "$.after"
      }

      conditions {
        path      = "ref"
        condition = "equals"
        value     = "refs/heads/main"
      }
    }
  }
}
```

### DataDog Integration (Enterprise)

```hcl
resource "rundeck_job" "datadog_alert_handler" {
  project_name = rundeck_project.example.name
  name         = "handle-datadog-alert"
  description  = "Handle DataDog alerts"

  command {
    shell_command = "pagerduty create-incident --alert @option.alert_id@"
  }
}

resource "rundeck_webhook" "datadog" {
  project      = rundeck_project.example.name
  name         = "datadog-webhook"
  user         = "datadog-bot"
  roles        = "admin"
  enabled      = true
  event_plugin = "datadog-run-job"
  
  config {
    batch_key    = "body"
    event_id_key = "id"

    rules {
      name   = "Error Alerts"
      job_id = rundeck_job.datadog_alert_handler.id

      job_options {
        name  = "alert_id"
        value = "$.id"
      }

      job_options {
        name  = "alert_title"
        value = "$.title"
      }

      conditions {
        path      = "alert_type"
        condition = "equals"
        value     = "error"
      }
    }
  }
}
```

### PagerDuty Integration V2 (Enterprise)

```hcl
resource "rundeck_job" "pagerduty_incident_handler" {
  project_name = rundeck_project.example.name
  name         = "handle-pagerduty-incident"
  description  = "Handle PagerDuty incidents"

  command {
    shell_command = "slack notify --incident @option.incident_id@"
  }
}

resource "rundeck_webhook" "pagerduty" {
  project      = rundeck_project.example.name
  name         = "pagerduty-webhook"
  user         = "pagerduty-bot"
  roles        = "admin"
  enabled      = true
  event_plugin = "pagerduty-run-job"
  
  config {
    batch_key    = "messages"
    event_id_key = "id"

    rules {
      name   = "Incident Triggered"
      job_id = rundeck_job.pagerduty_incident_handler.id

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
```

### PagerDuty Integration V3 (Enterprise)

```hcl
resource "rundeck_webhook" "pagerduty_v3" {
  project      = rundeck_project.example.name
  name         = "pagerduty-v3-webhook"
  user         = "pagerduty-bot"
  roles        = "admin"
  enabled      = true
  event_plugin = "pagerduty-V3-run-job"
  
  config {
    event_id_key = "event.id"

    rules {
      name   = "Incident Triggered V3"
      job_id = rundeck_job.pagerduty_incident_handler.id

      job_options {
        name  = "event_id"
        value = "$.event.id"
      }

      job_options {
        name  = "incident_title"
        value = "$.event.data.title"
      }

      conditions {
        path      = "event.event_type"
        condition = "equals"
        value     = "incident.triggered"
      }
    }
  }
}
```

### AWS SNS Integration (Enterprise)

```hcl
resource "rundeck_job" "cloudwatch_alarm_handler" {
  project_name = rundeck_project.example.name
  name         = "handle-cloudwatch-alarm"
  description  = "Handle AWS CloudWatch alarms"

  command {
    shell_command = "aws sns publish --topic-arn arn:aws:sns:us-east-1:123456789:alerts"
  }
}

resource "rundeck_webhook" "aws_sns" {
  project      = rundeck_project.example.name
  name         = "aws-sns-webhook"
  user         = "aws-bot"
  roles        = "admin"
  enabled      = true
  event_plugin = "aws-sns-webhook"
  
  config {
    auto_subscribe = true  # Automatically confirm SNS subscription

    rules {
      name   = "High CPU Alarm"
      job_id = rundeck_job.cloudwatch_alarm_handler.id

      job_options {
        name  = "alarm_name"
        value = "$.Message.AlarmName"
      }

      job_options {
        name  = "alarm_state"
        value = "$.Message.NewStateValue"
      }

      conditions {
        path      = "Message.AlarmName"
        condition = "equals"
        value     = "HighCPU"
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `project` - (Required) The project name that owns the webhook. Cannot be changed after creation (forces new resource).

* `name` - (Required) The name of the webhook. Cannot be changed after creation (forces new resource).

* `user` - (Required) The username the webhook executes as. This user's permissions determine what the webhook can do. Cannot be changed after creation (forces new resource).

* `roles` - (Required) Comma-separated list of roles for authorization. These roles are checked when the webhook is triggered. Cannot be changed after creation (forces new resource).

* `event_plugin` - (Required) The plugin type that handles webhook events. Common values:
  * `log-webhook-event` - Log incoming webhook events (OSS)
  * `webhook-run-job` - Simple job triggering (OSS)
  * `advanced-run-job` - Advanced job triggering with conditional routing (Enterprise)
  * `datadog-run-job` - DataDog integration (Enterprise)
  * `pagerduty-run-job` - PagerDuty V2 integration (Enterprise)
  * `pagerduty-V3-run-job` - PagerDuty V3 integration (Enterprise)
  * `github-webhook` - GitHub integration (Enterprise)
  * `aws-sns-webhook` - AWS SNS integration (Enterprise)

* `enabled` - (Required) Whether the webhook is enabled. Disabled webhooks return 404 when triggered. Cannot be changed after creation (forces new resource).

* `config` - (Optional) Plugin-specific configuration block. Structure varies by plugin type. See Plugin Configuration Reference below.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the webhook (integer ID).

* `auth_token` - The authentication token for the webhook. **Important:** This token is only available after creation and cannot be retrieved later. Store it securely.

## Config Block Reference

The `config` block structure varies by plugin type. All fields are optional unless noted.

### log-webhook-event Plugin

```hcl
config {
  log_level = "INFO"  # DEBUG, INFO, WARN, ERROR
}
```

### webhook-run-job Plugin

```hcl
config {
  job_id      = "job-uuid"             # Required: Job UUID to execute
  arg_string  = "-opt1 value1 -opt2 value2"  # Job arguments
  node_filter = "tags: linux"          # Node filter override
  as_user     = "deploy-bot"           # Execute as different user
}
```

### advanced-run-job and Enterprise Plugins

All Enterprise plugins (`advanced-run-job`, `datadog-run-job`, `pagerduty-run-job`, `pagerduty-V3-run-job`, `github-webhook`, `aws-sns-webhook`) support the following structure:

```hcl
config {
  # Optional fields for batch processing
  batch_key              = "data.events"    # JSONPath to array of events
  event_id_key           = "event_id"       # JSONPath for event ID
  return_processing_info = true             # Return processing details to caller
  key_storage_path       = "keys/webhook"   # Key storage path for encryption

  # GitHub-specific (github-webhook only)
  secret = "github-secret"  # GitHub webhook secret for authentication

  # AWS SNS-specific (aws-sns-webhook only)
  auto_subscribe = true  # Auto-confirm SNS subscription

  # Rules block - repeatable
  rules {
    name        = "Rule Name"          # Required: Rule name
    job_id      = "job-uuid"           # Required: Job to execute
    job_name    = "computed-name"      # Computed: Job name from job_id
    node_filter = "tags: production"   # Optional: Node filter override
    user        = "override-user"      # Optional: User override
    enabled     = true                 # Optional: Whether rule is enabled (default: true)
    policy      = "all"                # Optional: Condition policy - "all" (AND) or "any" (OR) (default: "all")

    # Job options - repeatable
    job_options {
      name  = "option-name"   # Required: Option name
      value = "$.json.path"   # Required: JSONPath to extract value from webhook payload
    }

    # Conditions - repeatable
    conditions {
      path      = "data.field.path"  # Required: JSONPath to event field to evaluate
      condition = "equals"            # Required: Operator - equals, contains, dateTimeAfter, dateTimeBefore, exists, isA
      value     = "expected-value"    # Required: Value to compare against
    }
  }
}
```

**Rules Block Fields:**
- `name` - (Required) Descriptive name for the rule
- `job_id` - (Required) UUID of the job to execute
- `job_name` - (Computed) Job name, computed from job_id
- `node_filter` - (Optional) Node filter override for job execution
- `user` - (Optional) User override for job execution  
- `enabled` - (Optional) Whether this rule is enabled (default: true)
- `policy` - (Optional) Condition matching policy: "all" (AND) or "any" (OR) (default: "all")

**Job Options Block Fields:**
- `name` - (Required) Job option name
- `value` - (Required) JSONPath expression to extract value from webhook payload

**Conditions Block Fields:**
- `path` - (Required) JSONPath to the event field to evaluate
- `condition` - (Required) Condition operator: `equals`, `contains`, `dateTimeAfter`, `dateTimeBefore`, `exists`, `isA`
- `value` - (Required) Expected value for comparison

## Import

Webhooks can be imported using the format `project/webhook-id`:

```bash
terraform import rundeck_webhook.example production/42
```

**Important:** After importing, the `auth_token` will not be available in the Terraform state since the Rundeck API doesn't return it on read operations. If you need the auth token:

1. Note the existing token from the Rundeck UI before importing, or
2. Delete and recreate the webhook to generate a new token

## Webhook URL Format

Once created, webhooks are accessible at:

```
https://rundeck.example.com/api/56/webhook/<auth-token>
```

Example with curl:

```bash
curl -X POST "https://rundeck.example.com/api/56/webhook/${AUTH_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"event": "deploy", "environment": "production", "severity": "critical"}'
```

## Common Patterns

### CI/CD Integration

```hcl
# Webhook triggered by CI/CD pipeline
resource "rundeck_webhook" "ci_deploy" {
  project      = "production"
  name         = "ci-deploy-trigger"
  user         = "ci-bot"
  roles        = "webhook,deploy"
  enabled      = true
  event_plugin = "webhook-run-job"
  
  config {
    job_id = rundeck_job.deploy_app.id
  }
}

# Use the webhook in your CI/CD pipeline
output "ci_webhook_url" {
  value       = "https://rundeck.example.com/api/56/webhook/${rundeck_webhook.ci_deploy.auth_token}"
  sensitive   = true
  description = "Add this URL to your CI/CD pipeline"
}
```

### Multiple Webhooks for Different Environments

```hcl
variable "environments" {
  type    = list(string)
  default = ["dev", "staging", "production"]
}

resource "rundeck_webhook" "deploy" {
  for_each = toset(var.environments)
  
  project      = "devops"
  name         = "${each.key}-deploy-webhook"
  user         = "automation"
  roles        = "webhook,deploy"
  enabled      = true
  event_plugin = "webhook-run-job"
  
  config {
    job_id     = rundeck_job.deploy[each.key].id
    arg_string = "-env ${each.key}"
  }
}
```

## Security Considerations

1. **Auth Token Protection:** The `auth_token` is sensitive and grants access to trigger the webhook. Store it securely (use Terraform outputs with `sensitive = true`) and never commit it to version control.

2. **User Permissions:** The webhook executes with the permissions of the specified `user`. Follow the principle of least privilege - ensure this user has only the necessary permissions.

3. **Role Authorization:** The webhook's `roles` are checked during execution. Restrict these to the minimum required roles.

4. **HTTPS Only:** Always use HTTPS when calling webhooks in production to prevent token interception.

5. **Token Rotation:** If a webhook token is compromised, delete and recreate the webhook to generate a new token. There is no API to regenerate tokens without recreating the webhook.

6. **GitHub Secret:** For `github-webhook`, use a strong secret and configure it in GitHub's webhook settings to verify request authenticity.

## Troubleshooting

### Webhook Returns 404

- Check that `enabled = true`
- Verify the auth token is correct
- Ensure the webhook hasn't been deleted
- Check the webhook URL format is correct

### Webhook Returns 403 Forbidden

- Verify the webhook `user` has necessary permissions in Rundeck
- Check that `roles` include required authorization roles
- Ensure the user account is not locked or disabled
- Review Rundeck's aclpolicy definitions

### Job Not Triggering

- Verify `job_id` in config is correct (use job UUID from `rundeck_job.id`)
- Check job exists in the specified project
- Review Rundeck logs (`/var/log/rundeck/service.log`) for execution errors
- Ensure the webhook user has execute permission on the job
- For advanced plugins, verify conditions and rules are correct

### Rules Not Matching

- Check JSONPath expressions in `conditions.path` are correct
- Verify condition operators (`equals`, `contains`, etc.) are appropriate
- Test webhook payload structure matches expected paths
- Review `policy` setting - "all" requires all conditions to match, "any" requires at least one

### Config Changes Not Applied

Due to a known Rundeck API limitation, config updates may not be immediately visible:

1. Terraform will apply the update successfully
2. A subsequent `terraform refresh` or `terraform plan` may show the config is correct
3. The webhook will function with the new config even if the API doesn't return it
4. This is expected behavior and not a provider bug

### AWS SNS Subscription Not Confirmed

- Ensure `auto_subscribe = true` is set
- Check Rundeck logs for subscription confirmation errors
- Verify the webhook URL is reachable from AWS SNS
- The subscription confirmation may take a few moments

## Testing Enterprise Features

Enterprise plugin tests require:
```bash
export RUNDECK_ENTERPRISE_TESTS=1
go test -v -run "TestAccRundeckWebhook" ./rundeck/
```

This ensures Enterprise-only plugins are properly tested before deployment.
