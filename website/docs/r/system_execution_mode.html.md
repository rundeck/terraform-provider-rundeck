---
layout: "rundeck"
page_title: "Rundeck: rundeck_system_execution_mode"
sidebar_current: "docs-rundeck-resource-system-execution-mode"
description: |-
  The rundeck_system_execution_mode resource controls whether a Rundeck server executes jobs.
---

# rundeck\_system\_execution\_mode

Controls whether a Rundeck server executes jobs. In `passive` mode no job runs at all — neither scheduled ones nor those started by hand — which makes it the switch to reach for during a migration, a maintenance window, or a database operation.

This is **server-wide runtime state**, not per-project configuration. To stop only the scheduled runs of a project while still allowing manual ones, set `project.disable.schedule` on that project instead.

Requires `api_version` 36 or higher: below that, the status endpoint this resource reads from does not reliably report passive mode.

## Example Usage

```hcl
resource "rundeck_system_execution_mode" "this" {
  mode = "active"
}
```

Holding a server passive while its catalogue is being built up:

```hcl
resource "rundeck_system_execution_mode" "staging" {
  mode = "passive"
}
```

## Argument Reference

* `mode` - (Required) Either `active` (jobs run) or `passive` (no job runs).

## Attributes Reference

* `id` - Always `system`. A Rundeck server has a single execution mode, so there is only ever one of these resources per server.

## Relationship with the `rundeck.executionMode` property

Rundeck reads a `rundeck.executionMode` property from its configuration file **at startup**, and keeps the mode in its database afterwards. The two answer different questions:

* the property decides the mode a **restarted** server comes back in;
* this resource decides the mode the server runs in **from the next apply onwards**.

Keeping them consistent is usually what you want: otherwise a restart silently reverts the mode until someone runs Terraform again. Note that when the property is absent entirely, Rundeck starts **passive** — `ConfigurationService` treats anything other than the literal `active` as inactive.

## Behaviour on destroy

Removing this resource stops Terraform tracking the mode and **deliberately leaves the server as it is**. There is nothing to delete — a server always has an execution mode — and flipping it on `terraform destroy` would either halt a live server or start jobs nobody asked to start. A warning is emitted to make the no-op explicit. Set the mode you want before removing the resource.

## Import

The execution mode always exists, so it can be imported with the constant id:

```
terraform import rundeck_system_execution_mode.this system
```
