# Example Terraform configuration for Rundeck Project Runner

terraform {
  required_providers {
    rundeck = {
      source = "rundeck/rundeck"
    }
  }
}

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

# Create a project runner associated with the project
resource "rundeck_project_runner" "example" {
  project_name = rundeck_project.example.name
  name         = "example-project-runner"
  description  = "Project-specific runner for example project"
  
  tag_names = "project,example,dev"
  
  installation_type = "linux"   # Valid values: "linux" (default), "windows", "kubernetes", "docker"
  replica_type      = "manual"  # Valid values: "manual" (default) or "ephemeral"
  
  # Node dispatch configuration
  runner_as_node_enabled = true
  remote_node_dispatch   = false
  runner_node_filter     = "tags: example"
}

# Output the runner details
output "project_runner_id" {
  value = rundeck_project_runner.example.runner_id
}

output "project_runner_token" {
  value     = rundeck_project_runner.example.token
  sensitive = true
}
