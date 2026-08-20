---
layout: "rundeck"
page_title: "Rundeck: rundeck_local_role"
sidebar_current: "docs-rundeck-resource-local-role"
description: |-
  The rundeck_local_role resource manages a Rundeck Enterprise local role and its membership.
---

# rundeck\_local\_role

Manages a Rundeck Enterprise local role, including its member usernames. Local roles are referenced by name (`authority`) from ACL policies (`rundeck_acl_policy`) to determine what a role is authorized to do; this resource manages the role and its membership, not what it's authorized to do.

**Requirements:** Requires Rundeck Enterprise with the **local user store** authentication realm enabled, and API v44+. Configure the provider with `api_version = "44"` or higher. If your instance authenticates users via LDAP/SSO/PAM instead of the local user store, this resource will not work — Rundeck has no REST-manageable role concept for those realms.

**Note:** This resource manages roles and role membership only. It does not create local user accounts — see the note below on `rundeck_local_user`.

## Example Usage

```hcl
provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "your-token"
  api_version = "44"
}

resource "rundeck_local_role" "operators" {
  authority   = "operators"
  description = "Operators role, granted execute access via ACL policy"

  members = [
    "alice",
    "bob",
  ]
}

resource "rundeck_acl_policy" "operators" {
  name = "operators.aclpolicy"

  policy = <<-YAML
    description: Operator access
    context:
      project: '.*'
    for:
      job:
        - allow: [read, run]
    by:
      group: ${rundeck_local_role.operators.authority}
  YAML
}
```

## Argument Reference

* `authority` - (Required) Name of the role, referenced as a group name in ACL policies.
* `description` - (Optional) Description of the role.
* `members` - (Optional) Set of local usernames assigned to this role. Membership is managed by resolving each username to its numeric user ID and diffing against the role's current membership on every apply — usernames that don't match an existing local user produce an error.

## Attributes Reference

* `id` - The numeric ID of the role, assigned by Rundeck.

## Note on membership and authorization

There is no "list members of a role" endpoint - determining a role's actual
membership requires listing every local user and checking each one's role
assignments. Depending on how your Rundeck instance's authorization is
configured, the token used may be able to manage roles but not list all
local users (a 403 on that specific call). If that happens, this provider
degrades gracefully: it emits a warning and leaves `members` as its last
known value (or an empty set if there's no prior value yet) rather than
failing the whole read/apply. If you rely on `members`, verify your token
has permission to list local users (`GET /secure/users`), not just manage
roles.

## Import

Local roles can be imported using the numeric role ID:

```
terraform import rundeck_local_role.example 3
```

## A note on `rundeck_local_user`

The Rundeck Enterprise API also exposes local **user** CRUD
(`/secure/users*`), but the underlying OpenAPI spec defines no request body
schema for creating or editing a user — the vendored Go SDK has no way to
send a username/password/roles payload to those two endpoints. Because of
that gap, this provider does not (yet) offer a `rundeck_local_user`
resource; `members` above must reference usernames that already exist,
provisioned some other way (Rundeck UI, direct API call, etc).
