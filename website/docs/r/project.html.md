---
layout: "rundeck"
page_title: "Rundeck: rundeck_project"
sidebar_current: "docs-rundeck-resource-project"
description: |-
  The rundeck_project resource allows Rundeck projects to be managed by Terraform.
---

# rundeck\_project

Organize your automation environment. Projects are containers for jobs, node resources, and execution settings. Managing projects as code ensures consistent configuration, simplifies multi-environment deployments, and provides a clear audit trail of infrastructure changes.

**Use projects to:** Group related automation tasks, define node inventories, configure SSH settings, and control job execution permissions.

## Example Usage

```hcl
resource "rundeck_project" "terraform" {
  name        = "terraform"
  description = "Sample Application Created by Terraform Plan"
  ssh_key_storage_path = "${rundeck_private_key.terraform.path}"
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      # This path is interpreted on the Rundeck server.
      file = "/home/rundeck/resources.yaml"
      writable = "true"
      generateFileAutomatically = "true"
    }
  }
  extra_config = {
    "project.label" = "Terraform Example"
  }
}
```

Note that the above configuration assumes the existence of a ``resources.yaml`` file in the
filesystem on the Rundeck server. The Rundeck provider does not itself support creating such a file,
but one way to place it would be to use the ``file`` provisioner to copy a configuration file
from the module directory.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the project, used both in the UI and to uniquely identify
  the project. Must therefore be unique across a single Rundeck installation.

* `resource_model_source` - (Required) Nested block instructing Rundeck on how to determine the
  set of resources (nodes) for this project. The nested block structure is described below.

* `description` - (Optional) A description of the project, to be displayed in the Rundeck UI.
  Defaults to "Managed by Terraform".

* `default_node_file_copier_plugin` - (Optional) The name of a plugin to use to copy files onto
  nodes within this project. Defaults to `jsch-scp`, which uses the "Secure Copy" protocol
  to send files over SSH.

* `default_node_executor_plugin` - (Optional) The name of a plugin to use to run commands on
  nodes within this project. Defaults to `jsch-ssh`, which uses the SSH protocol to access the
  nodes.

* `ssh_authentication_type` - (Optional) When the SSH-based file copier and executor plugins are
  used, the type of SSH authentication to use. Defaults to `privateKey`.

* `ssh_key_storage_path` - (Optional) When the SSH-based file copier and executor plugins are
  used, the location within Rundeck's key store where the SSH private key can be found. Private
  keys can be uploaded to rundeck using the `rundeck_private_key` resource.

* `ssh_key_file_path` - (Optional) Like `ssh_key_storage_path` except that the key is read from
  the Rundeck server's local filesystem, rather than from the key store.

* `extra_config` - (Optional) Behind the scenes a Rundeck project is really an arbitrary set of
  key/value pairs. This map argument allows setting any configuration properties that aren't
  explicitly supported by the other arguments described above, but due to limitations of Terraform
  the key names must be written wrapped in double quotes. Do not use this argument to set
  properties that the above arguments set, or undefined behavior will result.

`resource_model_source` blocks have the following nested arguments:

* `type` - (Required) The name of the resource model plugin to use.

* `config` - (Required) Map of arbitrary configuration properties for the selected resource model
  plugin.

## Attributes Reference

The following attributes are exported:

* `name` - The unique name that identifies the project, as set in the arguments.
* `ui_url` - The URL of the index page for this project in the Rundeck UI.

## Import

Rundeck Project can be imported using the `name`, e.g.

```
$ terraform import rundeck_project.main project-name
```
