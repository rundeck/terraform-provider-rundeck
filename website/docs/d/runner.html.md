---
layout: "rundeck"
page_title: "Rundeck: rundeck_runner"
sidebar_current: "docs-rundeck-datasource-runner"
description: |-
  The rundeck_runner data source retrieves information about a single Rundeck Enterprise runner.
---

# rundeck\_runner

Retrieves information about a single Rundeck Enterprise runner (system or project scoped) by its runner ID.

**Requirements:** Requires Rundeck Enterprise 5.17.0+ (API v56). Configure the provider with `api_version = "56"` or higher.

## Example Usage

```hcl
data "rundeck_runner" "example" {
  runner_id = "12345678-1234-1234-1234-123456789abc"
}

output "runner_status" {
  value = data.rundeck_runner.example.status
}
```

## Argument Reference

* `runner_id` - (Required) ID of the runner to look up.

## Attributes Reference

* `name` - Name of the runner.
* `description` - Description of the runner.
* `status` - Current status of the runner.
* `version` - Runner software version.
* `tag_names` - Comma separated tags for the runner.
* `runner_as_node_enabled` - Whether the runner acts as a node.
* `remote_node_dispatch` - Whether remote node dispatch is enabled for the runner.
* `runner_node_filter` - Node filter string for the runner.
* `hostname` - Hostname the runner is running on.
* `os_family` - Operating system family of the runner host.
* `replica_type` - Replica type of the runner (`manual` or `ephemeral`).
* `installation_type` - Installation type of the runner (`linux`, `windows`, `kubernetes`, `docker`).
* `date_created` - Timestamp the runner was created, in RFC3339 format.
* `last_updated` - Timestamp the runner was last updated, in RFC3339 format.
* `last_checkin` - Timestamp of the runner's last check-in.
* `uptime` - Runner uptime, in seconds.
* `running_operations` - Number of operations currently running on the runner.
* `project_associations` - List of per-project dispatch settings associated with this runner. Each entry has:
  * `project_name` - Name of the associated project.
  * `node_filter` - Node filter configured for this project association.
  * `runner_as_node_enabled` - Whether the runner acts as a node for this project.
  * `remote_node_dispatch` - Whether remote node dispatch is enabled for this project.
  * `runner_node_filter` - Runner node filter configured for this project.
