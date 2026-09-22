---
layout: "rundeck"
page_title: "Rundeck: rundeck_scm_import"
sidebar_current: "docs-rundeck-resource-scm-import"
description: |-
  The rundeck_scm_import resource configures a Rundeck project's SCM import plugin (e.g. git-import, svn-import).
---

# rundeck\_scm\_import

Configures and enables a project's SCM **import** plugin (e.g. `git-import`, `svn-import`) - job definitions pulled into Rundeck from a version control repository.

**Requirements:** none beyond the provider's own minimum. The SCM API endpoints this resource uses have shipped with core Rundeck (not Enterprise-gated) since API v15, well below the provider's documented overall minimum of v46 (Rundeck 5.0.0+), so there's no separate version constraint to configure for.

## Example Usage

```hcl
resource "rundeck_private_key" "scm" {
  path         = "terraform/scm_import_key"
  # Terraform's file() does not expand a leading "~" - use an absolute
  # path, or one relative to this module (e.g. "${path.module}/rundeck_scm_deploy_key").
  key_material = file("/home/youruser/.ssh/rundeck_scm_deploy_key")
}

resource "rundeck_project" "example" {
  name        = "example"
  description = "Example project"

  resource_model_source {
    type   = "local"
    config = {}
  }
}

resource "rundeck_scm_import" "example" {
  project = rundeck_project.example.name
  type    = "git-import"

  config = {
    url                   = "git@github.com:myorg/myproject-rundeck.git"
    dir                   = "/var/rundeck/scm/example"
    branch                = "main"
    pathTemplate          = "$${job.group}$${job.name}-$${job.id}.xml"
    format                = "xml"
    useFilePattern        = "false"
    sshPrivateKeyPath     = "keys/${rundeck_private_key.scm.path}"
    strictHostKeyChecking = "no"
  }
}
```

## Argument Reference

* `project` - (Required, Forces new resource) Name of the project to configure SCM import for.
* `type` - (Required, Forces new resource) SCM plugin type name (e.g. `git-import`, `svn-import`). Changing this requires replacing the resource, since it amounts to reconfiguring from scratch.
* `config` - (Required) Plugin-specific configuration key/value pairs. The set of required/valid keys is dynamic per plugin type - check Rundeck's SCM plugin setup page in the UI, or the plugin's documentation, for the exact keys it expects. Reference external secret storage for credential-like values (e.g. SSH key paths, not raw key material) rather than embedding secrets directly.

  For the bundled `git-import` plugin specifically, confirmed against a live instance's plugin input schema: `dir`, `pathTemplate`, `useFilePattern`, and `strictHostKeyChecking` are required in addition to `url`, `branch`, and `format` - the same requirements as `git-export`'s `dir`/`pathTemplate`/`strictHostKeyChecking`, plus `useFilePattern` (whether to only import files matching `filePattern`, which defaults to `.*\.xml`).

* `enabled` - (Optional) Whether the plugin should be enabled for the project. Defaults to `true`. Rundeck
  treats this as an operational toggle rather than a normal argument - it's common to disable a plugin
  out-of-band (during an incident, a migration, etc.) and expect it to stay disabled until someone
  re-enables it. Leaving this unset (the default) corrects that drift back to enabled on the next apply;
  set it explicitly to `false` to have Terraform respect and enforce a disabled state instead.

## Attributes Reference

* `id` - The ID of this resource, in the form `"project:type"`.

## Import

```
terraform import rundeck_scm_import.example my-project:git-import
```

## Notes

- There is no dedicated "delete" operation for SCM plugin configuration - destroying this resource disables the plugin (`ApiProjectDisable`), which is the closest available operation. The configuration may still persist server-side in a disabled state; there's no API to fully remove it.
- Rundeck has no validation/dry-run endpoint for SCM plugin config - invalid configuration is only caught at apply time, surfaced as a Rundeck-reported validation error.
- This resource only configures the import plugin; it does not trigger an import action (pulling jobs from the repository). Triggering SCM actions (e.g. import/commit/synch) is not currently supported by this provider.
