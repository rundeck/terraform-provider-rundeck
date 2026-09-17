---
layout: "rundeck"
page_title: "Rundeck: rundeck_runners"
sidebar_current: "docs-rundeck-datasource-runners"
description: |-
  The rundeck_runners data source lists Rundeck Enterprise runners, optionally filtered by project, tags, status, or a generic filter string.
---

# rundeck\_runners

Lists Rundeck Enterprise runners, optionally scoped to a project and filtered by tags, status, or a generic filter string.

**Requirements:** Requires Rundeck Enterprise 5.17.0+ (API v56). Configure the provider with `api_version = "56"` or higher.

## Example Usage

```hcl
# All system runners tagged "production"
data "rundeck_runners" "production" {
  tags = "production"
}

# Runners assigned to a specific project
data "rundeck_runners" "project_scoped" {
  project_name = "my-project"
}

output "runner_ids" {
  value = [for r in data.rundeck_runners.production.runners : r.id]
}
```

## Argument Reference

* `project_name` - (Optional) If set, list runners assigned to this project instead of all system runners.
* `tags` - (Optional) Comma separated list of tags to filter runners by.
* `status` - (Optional) Filter runners by status.
* `filter` - (Optional) Generic filter string passed through to the Rundeck API.
* `local_only` - (Optional) If `true`, only include the local runner.

There is no pagination on this endpoint — the full matching result set is always returned.

## Attributes Reference

* `runners` - List of runners matching the given filters. Each entry has:
  * `id` - ID of the runner.
  * `name` - Name of the runner.
  * `description` - Description of the runner.
  * `status` - Current status of the runner.
  * `version` - Runner software version.
  * `associated_projects` - Number of projects this runner is associated with.
  * `last_checkin` - Timestamp of the runner's last check-in.
  * `runner_as_node_enabled` - Whether the runner acts as a node.
  * `tag_names` - Comma separated tags for the runner.
  * `runner_node_filter` - Node filter string for the runner.
  * `remote_node_dispatch` - Whether remote node dispatch is enabled.
  * `hostname` - Hostname the runner is running on.
  * `os_family` - Operating system family of the runner host.
  * `replica_type` - Replica type of the runner (`manual` or `ephemeral`).
  * `installation_type` - Installation type of the runner.
  * `providers` - List of plugin providers (node/step/etc) this runner handles, each with `provider`, `service_name`, and `plugin_name`.
  * `runner_replicas` - Number of replicas registered for this runner.
  * `healthy_runner_replicas` - Number of healthy replicas registered for this runner.
  * `running_operations` - Number of operations currently running on the runner.
  * `uptime` - Runner uptime, in seconds.
  * `date_created` - Timestamp the runner was created, in RFC3339 format.
  * `last_updated` - Timestamp the runner was last updated, in RFC3339 format.
