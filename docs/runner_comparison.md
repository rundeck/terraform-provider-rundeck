# Rundeck Runner Resources Comparison

This document compares the two runner resources available in the Rundeck Terraform provider.

## Overview

| Resource | Scope | Use Case |
|----------|-------|----------|
| `rundeck_system_runner` | Global/System-wide | Enterprise-wide runners for multiple projects |
| `rundeck_project_runner` | Project-specific | Project-scoped runners for targeted execution |

## Resource Comparison

### Common Features

Both resources support:
- Runner name and description
- Tag names for categorization
- Assigned projects mapping
- Project runner as node configuration
- Installation and replica type settings
- Authentication and download tokens (computed)

### Key Differences

#### System Runner (`rundeck_system_runner`)
- **Scope**: Global across the entire Rundeck instance
- **API**: Uses `CreateRunner()` from RunnerAPI
- **ID Format**: Simple runner ID
- **Use Case**: Organization-wide runners that can serve multiple projects
- **Management**: Centralized runner management

```hcl
resource "rundeck_system_runner" "global" {
  name        = "global-runner"
  description = "System-wide runner for all projects"
  tag_names   = "global,production"
  
  assigned_projects = {
    "project-a" = "execute"
    "project-b" = "execute"
    "project-c" = "read"
  }
}
```

#### Project Runner (`rundeck_project_runner`)
- **Scope**: Limited to a specific project
- **API**: Uses `CreateProjectRunner(project)` from RunnerAPI
- **ID Format**: Composite `project:runner_id`
- **Use Case**: Project-specific runners with targeted execution
- **Management**: Project-scoped runner management

```hcl
resource "rundeck_project_runner" "specific" {
  project_name = "my-project"
  name         = "project-specific-runner"
  description  = "Runner dedicated to my-project"
  tag_names    = "project,dev"
  
  assigned_projects = {
    "my-project" = "execute"
  }
}
```

## API Endpoints Used

### System Runner
- **Create**: `POST /api/53/runner`
- **Read**: `GET /api/53/runner/{id}`
- **Delete**: Via project associations

### Project Runner
- **Create**: `POST /api/53/project/{project}/runner`
- **Read**: `GET /api/53/project/{project}/runners` (list and filter)
- **Delete**: `DELETE /api/53/project/{project}/runner/{id}`

## Import Syntax

### System Runner
```bash
terraform import rundeck_system_runner.example 12345678-1234-1234-1234-123456789abc
```

### Project Runner
```bash
terraform import rundeck_project_runner.example my-project:12345678-1234-1234-1234-123456789abc
```

## When to Use Which

### Use System Runner When:
- You need a runner that serves multiple projects
- You want centralized runner management
- You have enterprise-wide execution requirements
- You need to manage runner permissions across multiple projects

### Use Project Runner When:
- You need project-specific execution isolation
- You want project-scoped runner management
- You have dedicated resources for specific projects
- You need fine-grained control over project execution

## Configuration Examples

### Mixed Configuration
```hcl
# Global system runner for enterprise-wide tasks
resource "rundeck_system_runner" "enterprise" {
  name        = "enterprise-runner"
  description = "Enterprise-wide execution runner"
  tag_names   = "enterprise,global,production"
  
  assigned_projects = {
    "project-a" = "execute"
    "project-b" = "execute"
    "project-c" = "execute"
  }
}

# Project-specific runner for development
resource "rundeck_project_runner" "dev_project" {
  project_name = "development"
  name         = "dev-runner"
  description  = "Development project runner"
  tag_names    = "development,testing"
  
  assigned_projects = {
    "development" = "execute"
  }
  
  project_runner_as_node = {
    "development" = true
  }
}
```

This hybrid approach allows for both centralized and distributed runner management based on your organization's needs.
