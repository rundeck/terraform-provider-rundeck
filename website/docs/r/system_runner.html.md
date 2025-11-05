---
layout: "rundeck"
page_title: "Rundeck: rundeck_system_runner"
sidebar_current: "docs-rundeck-resource-system-runner"
description: |-
  The rundeck_system_runner resource allows system-level (global) Enterprise Runners to be managed by Terraform.
---

# rundeck\_system\_runner

The system runner resource allows system level Enterprise Runners (Runbook Automation commercial feature) to be managed by Terraform. System runners are created at the system level and can be assigned to multiple projects.

## Example Usage

```hcl
# Create a system runner
resource "rundeck_system_runner" "example" {
  name        = "example-system-runner"
  description = "Global runner for multiple projects"
  
  tag_names = "system,production,global"
  
  assigned_projects = {
    "project-1" = "read"
    "project-2" = "execute"
    "project-3" = "admin"
  }
  
  project_runner_as_node = {
    "project-1" = true
    "project-2" = false
  }
  
  installation_type = "docker"
  replica_type      = "manual"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the runner.

* `description` - (Required) The description of the runner.

* `tag_names` - (Optional) Comma-separated tags for the runner.

* `assigned_projects` - (Optional) Map of assigned projects with their access levels (e.g., "read", "execute", "admin").

* `project_runner_as_node` - (Optional) Map of projects where the runner acts as a node (boolean values).

* `installation_type` - (Optional) Installation type of the runner. Valid values: `linux`, `windows`, `kubernetes`, `docker`. Defaults to `linux`.

* `replica_type` - (Optional) Replica type of the runner. Valid values: `manual`, `ephemeral`. Defaults to `manual`.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `runner_id` - The ID of the created runner.

* `token` - The authentication token for the runner (sensitive).

* `download_token` - The download token for the runner package (sensitive).

## Import

System runners can be imported using the runner ID:

```
terraform import rundeck_system_runner.example 12345678-1234-1234-1234-123456789abc
```

## Enterprise Feature

**Note:** System runners are an Enterprise-only feature and require Rundeck Enterprise. This resource will not work with Rundeck Community Edition.
