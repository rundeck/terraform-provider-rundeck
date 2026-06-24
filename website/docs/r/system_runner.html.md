---
layout: "rundeck"
page_title: "Rundeck: rundeck_system_runner"
sidebar_current: "docs-rundeck-resource-system-runner"
description: |-
  The rundeck_system_runner resource allows system-level (global) Enterprise Runners to be managed by Terraform with full per-project dispatch configuration.
---

# rundeck\_system\_runner

The system runner resource allows system level Enterprise Runners (Runbook Automation commercial feature) to be managed by Terraform. System runners are created at the system level and can be assigned to multiple projects with per-project dispatch configuration.

**Requirements:** Requires Rundeck Enterprise 5.17.0+ (API v56). Configure the provider with `api_version = "56"` or higher.

## Example Usage

### Basic Usage

```hcl
provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "your-token"
  api_version = "56"  # Required for runner resources
}

# Simple system runner with basic project assignment
resource "rundeck_system_runner" "basic" {
  name              = "basic-runner"
  description       = "Basic system runner"
  tag_names         = "production,default"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"
  
  assigned_projects = {
    "project-1" = "admin"
    "project-2" = "execute"
  }
}
```

### Advanced Usage with Per-Project Dispatch Configuration

```hcl
# System runner with full per-project dispatch settings
resource "rundeck_system_runner" "with_dispatch" {
  name              = "runner-with-dispatch"
  description       = "Runner with per-project dispatch configuration"
  tag_names         = "production,runners"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"
  
  assigned_projects_config = {
    "project-1" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: RUNNER"
    }
    
    "project-2" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = false
      runner_node_filter       = "tags: DEFAULT"
    }
  }
}
```

### DR/HA Use Case

```hcl
# Primary DR runner for disaster recovery automation
resource "rundeck_system_runner" "dr_primary" {
  name              = "dr-primary-runner"
  description       = "Primary DR runner for HA"
  tag_names         = "dr,primary"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"

  assigned_projects_config = {
    "dr-project-1" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR-PRIMARY"
    }
    "dr-project-2" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR-PRIMARY"
    }
  }
}

# Secondary DR runner
resource "rundeck_system_runner" "dr_secondary" {
  name              = "dr-secondary-runner"
  description       = "Secondary DR runner for HA"
  tag_names         = "dr,secondary"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"

  assigned_projects_config = {
    "dr-project-1" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR-SECONDARY"
    }
    "dr-project-2" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR-SECONDARY"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the runner.

* `description` - (Required) The description of the runner.

* `tag_names` - (Optional) Comma-separated tags for the runner. Rundeck normalizes tags to lowercase and sorts them alphabetically. The provider handles this automatically using semantic equality to prevent plan drift.

* `assigned_projects` - (Optional) Map of assigned projects with their access levels. Valid access levels: `read`, `execute`, `admin`. This is the legacy approach for simple project assignments without dispatch configuration.

* `project_runner_as_node` - (Optional) **Deprecated.** Map of projects where the runner acts as a node (boolean values). Use `assigned_projects_config` instead for new configurations.

* `assigned_projects_config` - (Optional) Map of project configurations with full dispatch settings. Each project configuration supports:
  * `access_level` - (Required) Access level for the project. Valid values: `read`, `execute`, `admin`.
  * `runner_as_node_enabled` - (Optional, Default: `false`) Enable the runner to act as a node for this project.
  * `remote_node_dispatch` - (Optional, Default: `false`) Enable remote node dispatch for the runner in this project.
  * `runner_node_filter` - (Optional) Node filter string for the runner in this project (e.g., "tags: RUNNER").

* `installation_type` - (Optional) Installation type of the runner. Valid values: `linux`, `windows`, `kubernetes`, `docker`. Defaults to `linux`.

* `replica_type` - (Optional) Replica type of the runner. Valid values: `manual`, `ephemeral`. Defaults to `manual`.

### Precedence Rules

When a project appears in both `assigned_projects` and `assigned_projects_config`, the configuration in `assigned_projects_config` **takes precedence**. This allows for gradual migration from the legacy approach to the new dispatch configuration.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the runner.

* `runner_id` - The ID of the created runner.

* `token` - The authentication token for the runner (sensitive, only returned on creation).

* `download_token` - The download token for the runner package (sensitive, only returned on creation).

## Important Notes

### API Limitations

The Rundeck API does not currently expose a GET endpoint for per-project dispatch settings (`runner_as_node_enabled`, `remote_node_dispatch`, `runner_node_filter`). As a result:

- These settings are stored in Terraform state during `create`/`update` operations.
- The provider cannot detect out-of-band changes made via the Rundeck UI or API.
- If dispatch settings are modified outside of Terraform, use `terraform import` to refresh the state.

### Use Case: Disaster Recovery Automation

This feature was designed to support large-scale DR automation scenarios where hundreds of runners need to be managed as code. The per-project dispatch configuration allows customers to:

- Manage 600+ runners across multiple projects with Terraform
- Configure different node dispatch settings per project
- Support HA/DR with primary/secondary runner configurations
- Eliminate manual UI/API configuration after Terraform apply

## Import

System runners can be imported using the runner ID:

```
terraform import rundeck_system_runner.example 12345678-1234-1234-1234-123456789abc
```

**Note:** Imported runners will preserve `assigned_projects` in state, but `assigned_projects_config` settings (if configured via UI) will need to be manually added to your Terraform configuration after import.

## Enterprise Feature

**Note:** System runners are an Enterprise-only feature and require Rundeck Enterprise. This resource will not work with Rundeck Community Edition.
