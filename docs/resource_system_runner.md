# Rundeck System Runner Resource

This resource allows you to create and manage Rundeck System Runners using the new Rundeck v2 API.

## Example Usage

```hcl
resource "rundeck_system_runner" "example" {
  name        = "my-runner"
  description = "Example runner created with Terraform"
  tag_names   = "production,api"
  
  assigned_projects = {
    "project1" = "full"
    "project2" = "limited"
  }
  
  project_runner_as_node = {
    "project1" = true
    "project2" = false
  }
  
  installation_type = "docker"
  replica_type      = "manual"
}
```

## Argument Reference

The following arguments are supported:

- `name` - (Required) Name of the runner
- `description` - (Required) Description of the runner
- `tag_names` - (Optional) Comma separated tags for the runner
- `assigned_projects` - (Optional) Map of assigned projects with their access levels
- `project_runner_as_node` - (Optional) Map of projects where runner acts as node (boolean values)
- `installation_type` - (Optional) Installation type of the runner
- `replica_type` - (Optional) Replica type of the runner

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `runner_id` - The ID of the created runner
- `token` - Authentication token for the runner (sensitive)
- `download_token` - Download token for the runner package (sensitive)

## Import

Runners can be imported using their ID:

```bash
terraform import rundeck_system_runner.example runner-id-here
```

## Note

This resource uses the new Rundeck v2 API and requires Rundeck version 42 or later.
