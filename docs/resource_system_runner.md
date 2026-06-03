# Rundeck System Runner Resource

This resource allows you to create and manage Rundeck System Runners using the new Rundeck v2 API.

## Example Usage

### Basic Usage with Simple Project Assignment

```hcl
resource "rundeck_system_runner" "basic" {
  name              = "my-runner"
  description       = "Example runner created with Terraform"
  tag_names         = "production,api"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"
  
  assigned_projects = {
    "project1" = "admin"
    "project2" = "user"
  }
}
```

### Advanced Usage with Per-Project Dispatch Configuration

```hcl
resource "rundeck_system_runner" "with_dispatch" {
  name              = "runner-with-dispatch"
  description       = "Runner with per-project dispatch settings"
  tag_names         = "production,runners"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"
  
  assigned_projects_config = {
    "project1" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: RUNNER"
    }
    
    "project2" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = false
      runner_node_filter       = "tags: DEFAULT"
    }
  }
}
```

### Mixed Configuration (Gradual Migration)

```hcl
resource "rundeck_system_runner" "mixed" {
  name              = "mixed-runner"
  description       = "Runner with mixed configuration"
  tag_names         = "production"
  installation_type = "linux"
  replica_type      = "manual"
  
  # Simple projects without special dispatch needs
  assigned_projects = {
    "simple-project" = "admin"
  }
  
  # Projects requiring specific dispatch configuration
  assigned_projects_config = {
    "dr-project" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR,tags: RUNNER"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

### Required Arguments

- `name` - (Required) Name of the runner.
- `description` - (Required) Description of the runner.

### Optional Arguments

- `tag_names` - (Optional) Comma-separated tags for the runner. Rundeck normalizes tags to lowercase and sorts them alphabetically. The provider handles this automatically to prevent plan drift.

- `assigned_projects` - (Optional) Map of assigned projects with their access levels (e.g., `"project-name" = "admin"`). This is the legacy approach for simple project assignments without dispatch configuration.

- `project_runner_as_node` - (Optional) Map of projects where the runner acts as a node (boolean values). **Deprecated:** Use `assigned_projects_config` instead for new configurations.

- `assigned_projects_config` - (Optional) Map of project configurations with full dispatch settings. Each project configuration supports:
  - `access_level` - (Required) Access level for the project (e.g., "admin", "user").
  - `runner_as_node_enabled` - (Optional, Default: `false`) Enable the runner to act as a node for this project.
  - `remote_node_dispatch` - (Optional, Default: `false`) Enable remote node dispatch for the runner in this project.
  - `runner_node_filter` - (Optional) Node filter string for the runner in this project (e.g., "tags: RUNNER").

- `installation_type` - (Optional, Default: `"linux"`) Installation type of the runner. Valid values: `linux`, `windows`, `kubernetes`, `docker`.

- `replica_type` - (Optional, Default: `"manual"`) Replica type of the runner. Valid values: `manual`, `ephemeral`.

### Precedence Rules

When a project appears in both `assigned_projects` and `assigned_projects_config`, the configuration in `assigned_projects_config` takes precedence. This allows for gradual migration from the legacy approach to the new dispatch configuration.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The ID of the runner.
- `runner_id` - The ID of the created runner.
- `token` - Authentication token for the runner (sensitive, only available on creation).
- `download_token` - Download token for the runner package (sensitive, only available on creation).

## Important Notes

### API Limitations

The Rundeck API does not expose a GET endpoint for per-project dispatch settings (`runner_as_node_enabled`, `remote_node_dispatch`, `runner_node_filter`). As a result:

- These settings are stored in Terraform state during `create`/`update` operations.
- The provider cannot detect out-of-band changes made via the Rundeck UI or API.
- If dispatch settings are modified outside of Terraform, use `terraform import` to refresh the state.

### Disaster Recovery / High Availability Use Case

This feature was designed to support large-scale DR automation scenarios where hundreds of runners need to be managed with Terraform. Example:

```hcl
resource "rundeck_system_runner" "dr_primary" {
  name              = "dr-primary-runner"
  description       = "Primary DR runner"
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
  }
}
```

## Import

Runners can be imported using their ID:

```bash
terraform import rundeck_system_runner.example runner-id-here
```

## Requirements

- Rundeck version: 5.0.0+ / RBA 6.1.0+
- API version: 56+ (Enterprise feature)
- Set `api_version = "56"` or higher in your provider configuration
