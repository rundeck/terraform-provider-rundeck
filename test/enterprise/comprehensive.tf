terraform {
  required_providers {
    rundeck = {
      source = "terraform-providers/rundeck"
    }
  }
}

provider "rundeck" {
  url         = var.rundeck_url
  auth_token  = var.rundeck_token
  api_version = "56"
}

variable "rundeck_url" {
  description = "Rundeck URL"
  type        = string
  default     = "http://localhost:4440"
}

variable "rundeck_token" {
  description = "Rundeck API Token"
  type        = string
  sensitive   = true
}

#===============================================================================
# Project Setup
#===============================================================================

resource "rundeck_project" "comprehensive_test" {
  name        = "comprehensive-test"
  description = "Comprehensive test project for provider modernization"
  
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/comprehensive-test-nodes.yaml"
    }
  }
}

#===============================================================================
# Test 1: Node Filters & Dispatch Settings (CRITICAL FIX)
#===============================================================================

resource "rundeck_job" "node_dispatch_test" {
  project_name = rundeck_project.comprehensive_test.name
  name         = "01-Node-Dispatch-Test"
  description  = "Tests node filters and dispatch settings are correctly nested"
  
  # Node filter configuration
  node_filter_query         = "tags: web"
  node_filter_exclude_query = "name: maintenance-*"
  
  # Dispatch settings (should be nested inside nodefilters)
  max_thread_count               = 10
  continue_next_node_on_error    = true
  rank_order                     = "ascending"
  rank_attribute                 = "nodename"
  node_filter_exclude_precedence = true
  success_on_empty_node_filter   = false
  
  command {
    shell_command = "echo 'Testing node dispatch'"
    description   = "Verify job dispatches to filtered nodes"
  }
  
  command {
    shell_command = "hostname && uptime"
    description   = "Show node info"
  }
}

#===============================================================================
# Test 2: Execution Lifecycle Plugins (CRITICAL FIX)
#===============================================================================

resource "rundeck_job" "lifecycle_plugins_test" {
  project_name      = rundeck_project.comprehensive_test.name
  name              = "02-Lifecycle-Plugins-Test"
  description       = "Tests execution lifecycle plugins use correct map structure"
  execution_enabled = true
  
  node_filter_query = ".*"
  
  # Multiple execution lifecycle plugins (now using correct map structure)
  execution_lifecycle_plugin {
    type = "Retry-Failed-Nodes"
  }
  
  execution_lifecycle_plugin {
    type = "killhandler"
    config = {
      killChilds = "true"
    }
  }
  
  execution_lifecycle_plugin {
    type = "refreshHealthCheckerCache"
    config = {
      enabled = "true"
    }
  }
  
  execution_lifecycle_plugin {
    type = "resume"
    config = {
      onRetry = "true"
    }
  }
  
  command {
    shell_command = "echo 'Testing lifecycle plugins'"
  }
}

#===============================================================================
# Test 3: UUID-Based Job References (NEW FEATURE)
#===============================================================================

resource "rundeck_job" "target_job" {
  project_name      = rundeck_project.comprehensive_test.name
  name              = "03a-Target-Job"
  description       = "Target job to be referenced by UUID"
  execution_enabled = true
  
  command {
    shell_command = "echo 'I am the target job referenced by UUID'"
  }
}

resource "rundeck_job" "caller_job_uuid" {
  project_name = rundeck_project.comprehensive_test.name
  name         = "03b-Caller-Job-UUID"
  description  = "Calls another job using immutable UUID reference"
  
  command {
    shell_command = "echo 'Starting workflow'"
  }
  
  command {
    job {
      uuid = rundeck_job.target_job.id
    }
    description = "Call target job by UUID (immutable)"
  }
  
  command {
    shell_command = "echo 'Workflow complete'"
  }
}

#===============================================================================
# Test 4: Complex Job Configuration
#===============================================================================

resource "rundeck_job" "complex_job" {
  project_name      = rundeck_project.comprehensive_test.name
  name              = "04-Complex-Job"
  description       = "Comprehensive job testing multiple features"
  execution_enabled = true
  
  # Scheduling
  schedule         = "0 0 12 ? * * *"
  schedule_enabled = false  # Disabled for testing
  time_zone        = "America/Los_Angeles"
  
  # Node filters and dispatch
  node_filter_query            = "tags: app"
  max_thread_count             = 5
  continue_next_node_on_error  = true
  command_ordering_strategy    = "node-first"
  
  # Job settings
  allow_concurrent_executions = false
  timeout                     = "30m"
  retry                       = "3"
  retry_delay                 = "10s"
  log_level                   = "INFO"
  
  # Log limit
  log_limit {
    output = "100MB"
    action = "halt"
    status = "failed"
  }
  
  # Options
  option {
    name          = "environment"
    label         = "Environment"
    default_value = "staging"
    required      = true
    value_choices = ["dev", "staging", "production"]
    description   = "Target environment"
  }
  
  option {
    name                      = "verbose"
    label                     = "Verbose Output"
    default_value             = "false"
    value_choices             = ["true", "false"]
    require_predefined_choice = true
  }
  
  # Commands
  command {
    shell_command = "echo 'Environment: $${option.environment}'"
    description   = "Display selected environment"
  }
  
  command {
    inline_script = <<-EOT
      #!/bin/bash
      set -e
      echo "Starting deployment..."
      sleep 2
      echo "Deployment complete!"
    EOT
    description   = "Run deployment script"
  }
  
  # Notifications
  notification {
    type         = "on_success"
    webhook_urls = ["https://example.com/webhook/success"]
  }
  
  notification {
    type         = "on_failure"
    webhook_urls = ["https://example.com/webhook/failure"]
  }
}

#===============================================================================
# Test 5: Local Execution (No Node Filters)
#===============================================================================

resource "rundeck_job" "local_execution" {
  project_name      = rundeck_project.comprehensive_test.name
  name              = "05-Local-Execution"
  description       = "Job that executes locally (no node dispatch)"
  execution_enabled = true
  
  # No node_filter_query = executes locally
  
  command {
    shell_command = "echo 'Running locally on Rundeck server'"
  }
  
  command {
    shell_command = "date && hostname"
  }
}

#===============================================================================
# Test 6: Job with Orchestrator
#===============================================================================

resource "rundeck_job" "orchestrator_test" {
  project_name = rundeck_project.comprehensive_test.name
  name         = "06-Orchestrator-Test"
  description  = "Tests orchestrator configuration"
  
  node_filter_query = ".*"
  
  orchestrator {
    type    = "maxPercentage"
    percent = 50
  }
  
  command {
    shell_command = "echo 'Testing orchestrator'"
  }
}

#===============================================================================
# Test 7: Job Reference by Name (Backward Compatibility)
#===============================================================================

resource "rundeck_job" "caller_job_name" {
  project_name = rundeck_project.comprehensive_test.name
  name         = "07-Caller-Job-Name"
  description  = "Calls another job using traditional name-based reference"
  
  command {
    shell_command = "echo 'Starting name-based workflow'"
  }
  
  command {
    job {
      name         = "03a-Target-Job"
      project_name = rundeck_project.comprehensive_test.name
    }
    description = "Call target job by name (backward compatible)"
  }
}

#===============================================================================
# Test 8: Import Test - Minimal Job
#===============================================================================

resource "rundeck_job" "import_test" {
  project_name      = rundeck_project.comprehensive_test.name
  name              = "08-Import-Test"
  description       = "Minimal job for testing import functionality"
  execution_enabled = true
  
  node_filter_query = ".*"
  max_thread_count  = 3
  
  command {
    shell_command = "echo 'Import test'"
  }
  
  execution_lifecycle_plugin {
    type = "Retry-Failed-Nodes"
  }
}

#===============================================================================
# Outputs
#===============================================================================

output "project_url" {
  value       = "http://localhost:4440/project/${rundeck_project.comprehensive_test.name}"
  description = "URL to view the project in Rundeck UI"
}

output "job_ids" {
  value = {
    node_dispatch        = rundeck_job.node_dispatch_test.id
    lifecycle_plugins    = rundeck_job.lifecycle_plugins_test.id
    target_job           = rundeck_job.target_job.id
    caller_uuid          = rundeck_job.caller_job_uuid.id
    caller_name          = rundeck_job.caller_job_name.id
    complex              = rundeck_job.complex_job.id
    local_execution      = rundeck_job.local_execution.id
    orchestrator         = rundeck_job.orchestrator_test.id
    import_test          = rundeck_job.import_test.id
  }
  description = "Job UUIDs for all test jobs"
}

output "test_summary" {
  value = <<-EOT
  
  ========================================
  Comprehensive Test Summary
  ========================================
  
  Project: ${rundeck_project.comprehensive_test.name}
  URL: http://localhost:4440/project/${rundeck_project.comprehensive_test.name}
  
  Test Jobs Created:
  1. Node Dispatch Test      - Verifies nodefilters/dispatch structure fix
  2. Lifecycle Plugins Test  - Verifies plugin map structure fix
  3a. Target Job            - For UUID reference testing
  3b. Caller Job (UUID)     - Tests UUID-based job references
  4. Complex Job            - Tests multiple features together
  5. Local Execution        - Tests jobs without node dispatch
  6. Orchestrator Test      - Tests orchestrator configuration
  7. Caller Job (Name)      - Tests backward compatible name references
  8. Import Test            - Minimal job for import testing
  
  What to Verify in UI:
  ✓ Job 1: Should dispatch to nodes (not "Execute Locally")
  ✓ Job 2: Should have 4 execution lifecycle plugins visible
  ✓ Job 3b: Should reference job 3a by UUID
  ✓ Job 4: Should have schedule, options, notifications, log limits
  ✓ Job 5: Should be set to "Execute Locally"
  ✓ Job 6: Should have maxPercentage orchestrator
  ✓ Job 7: Should reference job 3a by name
  ✓ Job 8: Should have lifecycle plugins and node filters
  
  To test import:
    terraform state rm rundeck_job.import_test
    terraform import rundeck_job.import_test <job-uuid>
    terraform plan  # Should show no changes
  
  EOT
}

