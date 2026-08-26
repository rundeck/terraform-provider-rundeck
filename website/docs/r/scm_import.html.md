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
    url    = "git@github.com:myorg/myproject-rundeck.git"
    branch = "main"
    format = "xml"
  }
}
```

## Argument Reference

* `project` - (Required, Forces new resource) Name of the project to configure SCM import for.
* `type` - (Required, Forces new resource) SCM plugin type name (e.g. `git-import`, `svn-import`). Changing this requires replacing the resource, since it amounts to reconfiguring from scratch.
* `config` - (Required) Plugin-specific configuration key/value pairs. The set of required/valid keys is dynamic per plugin type - check Rundeck's SCM plugin setup page in the UI, or the plugin's documentation, for the exact keys it expects. Reference external secret storage for credential-like values (e.g. SSH key paths, not raw key material) rather than embedding secrets directly.

## Attributes Reference

* `id` - The ID of this resource, in the form `"project:type"`.
* `enabled` - Whether the plugin is currently enabled for the project.

## Import

```
terraform import rundeck_scm_import.example my-project:git-import
```

## Notes

- There is no dedicated "delete" operation for SCM plugin configuration - destroying this resource disables the plugin (`ApiProjectDisable`), which is the closest available operation. The configuration may still persist server-side in a disabled state; there's no API to fully remove it.
- Rundeck has no validation/dry-run endpoint for SCM plugin config - invalid configuration is only caught at apply time, surfaced as a Rundeck-reported validation error.
- This resource only configures the import plugin; it does not trigger an import action (pulling jobs from the repository). Triggering SCM actions (e.g. import/commit/synch) is not currently supported by this provider.
