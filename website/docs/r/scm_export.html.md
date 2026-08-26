---
layout: "rundeck"
page_title: "Rundeck: rundeck_scm_export"
sidebar_current: "docs-rundeck-resource-scm-export"
description: |-
  The rundeck_scm_export resource configures a Rundeck project's SCM export plugin (e.g. git-export, svn-export).
---

# rundeck\_scm\_export

Configures and enables a project's SCM **export** plugin (e.g. `git-export`, `svn-export`) - job definitions committed from Rundeck out to a version control repository.

**Requirements:** none beyond the provider's own minimum. The SCM API endpoints this resource uses have shipped with core Rundeck (not Enterprise-gated) since API v15, well below the provider's documented overall minimum of v46 (Rundeck 5.0.0+), so there's no separate version constraint to configure for.

## Example Usage

```hcl
resource "rundeck_private_key" "scm" {
  path         = "terraform/scm_export_key"
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

resource "rundeck_scm_export" "example" {
  project = rundeck_project.example.name
  type    = "git-export"

  config = {
    url                   = "git@github.com:myorg/myproject-rundeck.git"
    dir                   = "/var/rundeck/scm/example"
    branch                = "main"
    createBranch          = "true" # only needed if "main" doesn't exist yet on the remote
    committerName         = "rundeck"
    committerEmail        = "rundeck@example.com"
    pathTemplate          = "$${job.group}$${job.name}-$${job.id}.xml"
    format                = "xml"
    sshPrivateKeyPath     = "keys/${rundeck_private_key.scm.path}"
    strictHostKeyChecking = "no"
  }
}
```

## Argument Reference

* `project` - (Required, Forces new resource) Name of the project to configure SCM export for.
* `type` - (Required, Forces new resource) SCM plugin type name (e.g. `git-export`, `svn-export`). Changing this requires replacing the resource, since it amounts to reconfiguring from scratch.
* `config` - (Required) Plugin-specific configuration key/value pairs. The set of required/valid keys is dynamic per plugin type - check Rundeck's SCM plugin setup page in the UI, or the plugin's documentation, for the exact keys it expects. Reference external secret storage for credential-like values (e.g. SSH key paths, not raw key material) rather than embedding secrets directly.

  For the bundled `git-export` plugin specifically, confirmed against a live instance: `dir` (a local checkout directory on the Rundeck server) is required in addition to `url`; an SSH key referenced via `sshPrivateKeyPath` must use the **full** Rundeck key storage path including the `keys/` prefix (e.g. `keys/${rundeck_private_key.example.path}`, not just `${rundeck_private_key.example.path}`); and if `branch` doesn't already exist on the remote, you also need `createBranch = "true"`. `createBranch` creates the new branch based on `baseBranch` (default `master`) - a genuinely empty repository (zero commits) has no branch for it to base off of at all, so the remote repository needs at least one existing commit on some branch before `createBranch` can work.

## Attributes Reference

* `id` - The ID of this resource, in the form `"project:type"`.
* `enabled` - Whether the plugin is currently enabled for the project.

## Import

```
terraform import rundeck_scm_export.example my-project:git-export
```

## Notes

- There is no dedicated "delete" operation for SCM plugin configuration - destroying this resource disables the plugin (`ApiProjectDisable`), which is the closest available operation. The configuration may still persist server-side in a disabled state; there's no API to fully remove it.
- Rundeck has no validation/dry-run endpoint for SCM plugin config - invalid configuration is only caught at apply time, surfaced as a Rundeck-reported validation error.
- This resource only configures the export plugin; it does not trigger an export action (committing job definitions out to the repository). Triggering SCM actions (e.g. export/commit/synch) is not currently supported by this provider - a successful `terraform apply` means the plugin is configured and enabled, not that anything has been committed.
