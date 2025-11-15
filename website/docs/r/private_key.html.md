---
layout: "rundeck"
page_title: "Rundeck: rundeck_private_key"
sidebar_current: "docs-rundeck-resource-private-key"
description: |-
  The rundeck_private_key resource allows private keys to be stored in Rundeck's key store.
---

# rundeck\_private\_key

Securely manage SSH credentials for node access. Private keys stored in Rundeck's key store enable secure, automated authentication to target nodes without embedding credentials in job definitions. Managing keys as code ensures credential lifecycle is tracked and auditable.

## Example Usage

```hcl
resource "rundeck_private_key" "terraform" {
  path         = "terraform/id_rsa"
  key_material = "$${file(\"id_rsa.pub\")}"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path within the key store where the key will be stored.

* `key_material` - (Required) The private key material to store, serialized in any way that is
  accepted by OpenSSH.

The key material is hashed before it is stored in the state file, so sharing the resulting state
will not disclose the private key contents.

## Attributes Reference

Rundeck does not allow stored private keys to be retrieved via the API, so this resource does not
export any attributes.
