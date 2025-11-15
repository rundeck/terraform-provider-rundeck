---
layout: "rundeck"
page_title: "Provider: Rundeck"
sidebar_current: "docs-rundeck-index"
description: |-
  The Rundeck provider configures projects, jobs, ACLs and keys in Rundeck.
---

# Rundeck Provider

Manage your Rundeck runbook automation infrastructure as code. The Rundeck provider enables teams to version control, peer review, and consistently deploy Rundeck projects, jobs, access controls, and credentials across environments using Terraform.

## Key Benefits

**Infrastructure as Code:** Define your entire Rundeck configuration in version-controlled Terraform files. Track changes, roll back mistakes, and maintain consistency across development, staging, and production.

**Repeatable Deployments:** Eliminate manual configuration and human error. Deploy identical Rundeck environments programmatically, ensuring your runbook automation is consistent and reliable.

**Team Collaboration:** Review Rundeck changes through pull requests before deployment. Your runbook automation becomes part of your standard infrastructure workflow.

**Disaster Recovery:** Rebuild your complete Rundeck configuration from code in minutes, not hours. Your automation setup is documented, versioned, and instantly recoverable.

## What You Can Manage

- **Projects:** Create and configure Rundeck projects with resource models and settings
- **Jobs:** Define job workflows, commands, schedules, and notifications
- **ACL Policies:** Control access and permissions across your Rundeck instance
- **Credentials:** Manage SSH keys and passwords in Rundeck's key storage
- **Runners:** Configure Enterprise runners for distributed job execution (Enterprise only)

## Requirements

- Rundeck 5.0.0 or later (API v46+)
- Rundeck Enterprise 5.17.0+ required for runner resources (API v56)

## Configuration

The provider configuration block accepts the following arguments:

* ``url`` - (Required) The root URL of a Rundeck server. May alternatively be set via the
  ``RUNDECK_URL`` environment variable.

* ``api_version`` - (Optional) The API version of the server. Defaults to `46`, the
  minimum supported version. May alternatively be set via the ``RUNDECK_API_VERSION``
  environment variable.

* ``auth_token`` - The API auth token to use when making requests. May alternatively
  be set via the ``RUNDECK_AUTH_TOKEN`` environment variable. (RECOMMENDED)

**OR**

* ``auth_username`` - Local Login User Name.  
* ``auth_password`` - Local Login Password.

> Note: Username and Password auth will not work with SSO configured systems.  It relies on local Rundeck accounts. Please be sensitive to keeping passwords in your plan files!

Use the navigation to the left to read about the available resources.

## Example Usage

A full Example Exercise is included on the [Rundeck Learning site](https://docs.rundeck.com/docs/learning/howto/use-terraform-provider.html).

For those familiar with Terraform and Rundeck use the contents below.

```hcl
terraform {
  required_providers {
    rundeck = {
      source  = "rundeck/rundeck"
      version = "~> 1.0"
    }
  }
}

provider "rundeck" {
  url         = "http://rundeck.example.com:4440/"
  api_version = "46"
  auth_token  = "abcd1234"
}

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

resource "rundeck_job" "bounceweb" {
  name              = "Bounce All Web Servers"
  project_name      = "${rundeck_project.terraform.name}"
  node_filter_query = "tags: web"
  description       = "Restart the service daemons on all the web servers"

  command {
    shell_command = "sudo service anvils restart"
  }
}

resource "rundeck_public_key" "public_key" {
  path         = "terraform/id_rsa.pub"
  key_material = "ssh-rsa yada-yada-yada"
}

resource "rundeck_private_key" "private_key" {
  path         = "terraform/id_rsa"
  key_material = "$${file(\"id_rsa.pub\")}"
}

resource "rundeck_password" "password" {
  path         = "terraform/some_password"
  password = "qwerty"
}

data "local_file" "acl" {
  filename = "${path.cwd}/acl.yaml"
}

resource "rundeck_acl_policy" "example" {
  name = "ExampleAcl.aclpolicy"

  policy = "${data.local_file.acl.content}"
}
```

> Note: This example uses an ACL Policy file stored at the current working directory named `acl.yaml`.  Valid contents for that file are shown below.

```
by:
  group: terraform
description: Allow terraform Key Storage Access
for:
  storage:
  - allow:
    - read
context:
  application: rundeck
---
by:
  group: terraform
description: Allow Terraform Group [read] for all projects
for:
  project:
  - allow:
    - read
context:
  application: rundeck
---
by:
  group: terraform
description: Terraform Project Full Admin
for:
  project:
  - allow:
    - admin
    match:
      name: terraform
context:
  application: rundeck
```
