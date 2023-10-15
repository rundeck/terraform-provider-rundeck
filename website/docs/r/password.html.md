---
layout: "rundeck"
page_title: "Rundeck: rundeck_password"
sidebar_current: "docs-rundeck-resource-password"
description: |-
  The rundeck_password resource allows passwords to be stored in Rundeck's key store.
---

# rundeck\_password

The password resource allows passwords to be stored into Rundeck's key store.
The key store is where Rundeck keeps credentials that are needed to access the nodes on which
it runs commands.

## Example Usage

```hcl
resource "rundeck_password" "terraform" {
  path         = "terraform/some_password"
  password = "qwerty"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path within the key store where the key will be stored.

* `password` - (Required) The password to store.

The password is hashed before it is stored in the state file, so sharing the resulting state
will not disclose the private key contents.

## Attributes Reference

Rundeck does not allow stored passwords to be retrieved via the API, so this resource does not
export any attributes.
