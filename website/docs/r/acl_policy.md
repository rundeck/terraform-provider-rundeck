---
layout: "rundeck"
page_title: "Rundeck: rundeck_acl_policy"
sidebar_current: "docs-rundeck-resource-acl-policy"
description: |-
  The rundeck_acl_policy resource allows Rundeck ACLs to be managed by Terraform.
---

# rundeck\_acl_policy

The acl_policy resource allows Rundeck projects to be managed by Terraform.

## Example Usage

```hcl
data "local_file" "acl" {
  filename = "${path.module}/acl.yaml"
}

resource "rundeck_acl_policy" "example" {
  name = "ExampleAcl.aclpolicy"

  policy = "${data.local_file.acl.content}"
}
```

Note that the above configuration assumes the existence of an ``acl.yaml`` file in the
project directory. This resource passes the raw YAML policy string to Rundeck which stores
and returns it as-is. A future ``acl_policy_document`` data source is planned to allow defining
the policy in terraform configuration.

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the policy. Must end with `.aclpolicy`.

* `policy` - (Required) The name of the job, used to describe the job in the Rundeck UI.
