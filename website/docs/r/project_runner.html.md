---
layout: "rundeck"
page_title: "Rundeck: rundeck_project_runner"
sidebar_current: "docs-rundeck-resource-project-runner"
description: |-
  The rundeck_project_runner resource allows Project based Enterprise Runners to be managed by Terraform.
---

# rundeck\_project\_runner

The project runner resource allows project-specific Enterprise Runners to be managed by Terraform. Project runners are created within the context of a specific project and are typically used for project-scoped execution.

## Example Usage

```hcl
# Create a project first
resource "rundeck_project" "example" {
  name        = "example-project"
  description = "Example project for project runner"

  resource_model_source {
    type = "file"
    config = {
      format = "resourcexml"
      file   = "/tmp/example-resources.xml"
    }
  }
}

# Create a project runner
resource "rundeck_project_runner" "example" {
  project_name = rundeck_project.example.name
  name         = "example-project-runner"
  description  = "Project-specific runner for example project"
  
  tag_names = "project,example,dev"
  
  assigned_projects = {
    "example-project" = "read"
    "other-project"   = "execute"
  }
  
  project_runner_as_node = {
    "example-project" = true
  }
  
  installation_type = "docker"
  replica_type      = "single"
  
  # Node dispatch configuration
  runner_as_node_enabled = true
  remote_node_dispatch   = false
  runner_node_filter     = "tags: example"
}
```

## Argument Reference

The following arguments are supported:

* `project_name` - (Required) The name of the project where the runner will be created. This field forces a new resource if changed.

* `name` - (Required) The name of the runner.

* `description` - (Required) The description of the runner.

* `tag_names` - (Optional) Comma-separated tags for the runner.

* `assigned_projects` - (Optional) Map of assigned projects with their access levels.

* `project_runner_as_node` - (Optional) Map of projects where the runner acts as a node (boolean values).

* `installation_type` - (Optional) Installation type of the runner. Valid values are `"linux"`, `"windows"`, `"kubernetes"`, and `"docker"`.

* `replica_type` - (Optional) Replica type of the runner (e.g., "single", "multi").

* `runner_as_node_enabled` - (Optional) Enable the runner to act as a node. Defaults to `false`.

* `remote_node_dispatch` - (Optional) Enable remote node dispatch for the runner. Defaults to `false`.

* `runner_node_filter` - (Optional) Node filter string for the runner.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `runner_id` - The ID of the created runner.

* `token` - The authentication token for the runner (sensitive).

* `download_token` - The download token for the runner package (sensitive).

## Import

Project runners can be imported using the format `project_name:runner_id`:

```
terraform import rundeck_project_runner.example example-project:12345678-1234-1234-1234-123456789abc
```

Note: The project name and runner ID are both required for import.
