# Example 1: Basic system runner with legacy assigned_projects
resource "rundeck_system_runner" "basic" {
  name              = "basic-runner"
  description       = "Basic system runner"
  tag_names         = "production,default"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"

  # Legacy approach - simple project assignment without dispatch config
  assigned_projects = {
    "project-1" = "admin"
    "project-2" = "user"
  }
}

# Example 2: System runner with full per-project dispatch configuration
resource "rundeck_system_runner" "with_dispatch" {
  name              = "runner-with-dispatch"
  description       = "System runner with per-project dispatch settings"
  tag_names         = "production,runners"
  installation_type = "kubernetes"
  replica_type      = "ephemeral"

  # New approach - full dispatch configuration per project
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

    "project-3" = {
      access_level             = "execute"
      runner_as_node_enabled   = false
      remote_node_dispatch     = false
    }
  }
}

# Example 3: Mixed usage (assigned_projects_config takes precedence)
resource "rundeck_system_runner" "mixed" {
  name              = "mixed-runner"
  description       = "System runner with mixed configuration"
  tag_names         = "production"
  installation_type = "linux"
  replica_type      = "manual"

  # Projects without special dispatch needs
  assigned_projects = {
    "simple-project-1" = "admin"
    "simple-project-2" = "user"
  }

  # Projects with specific dispatch configuration
  # These will override any entry in assigned_projects for the same project
  assigned_projects_config = {
    "dr-project" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR,tags: RUNNER"
    }
  }
}

# Example 4: DR/HA use case - multiple runners with different dispatch configs
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

    "dr-project-2" = {
      access_level             = "admin"
      runner_as_node_enabled   = true
      remote_node_dispatch     = true
      runner_node_filter       = "tags: DR-PRIMARY"
    }
  }
}

resource "rundeck_system_runner" "dr_secondary" {
  name              = "dr-secondary-runner"
  description       = "Secondary DR runner"
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
