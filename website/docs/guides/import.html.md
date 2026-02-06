---
layout: "rundeck"
page_title: "Import Guide"
sidebar_current: "docs-rundeck-guide-import"
description: |-
  Guide for importing existing Rundeck resources into Terraform.
---

# Importing Existing Rundeck Resources

This guide provides step-by-step instructions for importing existing Rundeck resources into your Terraform configuration.

## Table of Contents

- [Why Import?](#why-import)
- [General Import Workflow](#general-import-workflow)
- [Resource-Specific Import Instructions](#resource-specific-import-instructions)
  - [Projects](#projects)
  - [Jobs](#jobs)
  - [Webhooks](#webhooks)
  - [ACL Policies](#acl-policies)
  - [Key Storage (Passwords & Keys)](#key-storage-passwords--keys)
  - [Runners (Enterprise)](#runners-enterprise)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)

---

## Why Import?

Importing allows you to bring existing Rundeck resources under Terraform management without having to recreate them. This is useful when:

- Migrating an existing Rundeck instance to Terraform
- Managing resources that were created manually or by other tools
- Recovering from state file loss
- Gradually adopting Terraform for Rundeck management

---

## General Import Workflow

The basic workflow for importing any Rundeck resource follows these steps:

### 1. Create the Resource Block

First, create an empty resource block in your Terraform configuration with the correct resource type and a local name:

```hcl
resource "rundeck_job" "my_job" {
  # Configuration will be filled in after import
}
```

### 2. Run Terraform Import

Use the `terraform import` command with the appropriate import ID format:

```bash
terraform import rundeck_job.my_job <IMPORT_ID>
```

### 3. Generate Configuration

After import, run `terraform plan` to see the differences, then populate your resource block with the actual values shown in the plan output.

Alternatively, use `terraform show` to view the imported state:

```bash
terraform show -no-color | grep -A 50 'resource "rundeck_job" "my_job"'
```

### 4. Verify and Apply

Once your configuration matches the imported state, run `terraform plan` to verify there are no changes, then proceed with your Terraform workflow.

---

## Resource-Specific Import Instructions

### Projects

**Import ID Format:** `project-name`

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_project" "production" {
  name        = "production"
  description = "Production environment"
}
EOF

# Import the project
terraform import rundeck_project.production production
```

**Important Notes:**
- The import ID is the project name
- After import, the `name` attribute should match the import ID
- Project configuration (`extra_config`) will be imported if it exists
- Default executor and node executor settings are included in the import

**Common Configuration:**

```hcl
resource "rundeck_project" "production" {
  name        = "production"
  description = "Production environment"
  
  resource_model_source {
    type = "file"
    config = {
      file                = "/var/rundeck/projects/production/etc/resources.xml"
      generateFileAutomatically = "true"
      includeServerNode = "true"
    }
  }
  
  extra_config = {
    "project.ssh-keypath" = "/var/rundeck/.ssh/id_rsa"
  }
}
```

---

### Jobs

**Import ID Format:** 
- Recommended: `job-uuid`
- Legacy format: `project-name/job-uuid`

**Finding Job UUIDs:**

```bash
# List jobs in a project via API
curl -H "X-Rundeck-Auth-Token: $RUNDECK_AUTH_TOKEN" \
  "$RUNDECK_URL/api/56/project/production/jobs" | jq '.[] | {name, id}'

# From the Rundeck UI: Job Actions → Edit this Job → URL contains the UUID
```

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_job" "deploy_app" {
  name         = "Deploy Application"
  project_name = "production"
  description  = "Deploy application to production servers"
  
  command {
    shell_command = "echo 'Job imported'"
  }
}
EOF

# Import using job UUID (recommended)
terraform import rundeck_job.deploy_app a4e5f6g7-8901-2345-6789-0abcdef12345

# Or using legacy format
terraform import rundeck_job.deploy_app production/a4e5f6g7-8901-2345-6789-0abcdef12345
```

**Important Notes:**
- **Job UUIDs are stable** - they don't change when the job is modified
- Job names can change, but UUIDs cannot
- Complex jobs with multiple commands, options, and plugins require careful configuration matching
- Notifications are imported but need to be structured correctly in HCL
- Job schedules are imported with their cron expressions

**Post-Import Tips:**
- Compare the imported job definition with the Rundeck UI to ensure all settings are captured
- Test complex jobs (with plugins, error handlers, notifications) in a non-production environment first
- Job option defaults and value choices must match exactly to avoid drift

---

### Webhooks

**Import ID Format:** `project-name/webhook-id`

**Finding Webhook IDs:**

```bash
# List webhooks in a project via API
curl -H "X-Rundeck-Auth-Token: $RUNDECK_AUTH_TOKEN" \
  "$RUNDECK_URL/api/56/webhook/admin?project=production" | jq '.[] | {name, id}'

# From the Rundeck UI: Project Settings → Webhooks → Click webhook → URL contains the ID
```

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_webhook" "github_deploy" {
  project      = "production"
  name         = "GitHub Deploy Webhook"
  user         = "admin"
  roles        = "admin"
  enabled      = true
  event_plugin = "webhook-run-job"
  
  config {
    job_id = "\${rundeck_job.deploy_app.id}"
  }
}
EOF

# Import the webhook
terraform import rundeck_webhook.github_deploy production/123
```

**Important Notes:**
- **Auth tokens are NOT imported** - The `auth_token` will be regenerated on the first `terraform apply`
- After import, the webhook URL will change due to the new auth token
- The `config` block structure varies by plugin type (webhook-run-job, advanced-run-job, etc.)
- Enterprise plugins (DataDog, PagerDuty, GitHub, AWS SNS) have additional configuration fields
- See the [webhook resource documentation](../r/webhook.html.md) for detailed `config` schema per plugin

**Post-Import Cleanup:**
1. Update any systems using the old webhook URL with the new URL (shown in `terraform apply` output)
2. Verify the `config` block matches the webhook plugin type
3. For advanced plugins with `rules`, `job_options`, or `conditions`, ensure nested blocks are correctly structured

---

### ACL Policies

**Import ID Format:** `policy-name.aclpolicy` (filename)

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_acl_policy" "api_access" {
  name   = "api-access.aclpolicy"
  policy = file("\${path.module}/policies/api-access.aclpolicy")
}
EOF

# Import the policy
terraform import rundeck_acl_policy.api_access api-access.aclpolicy
```

**Important Notes:**
- The import ID is the policy filename (must end in `.aclpolicy`)
- The `policy` content is imported as YAML
- Rundeck validates ACL syntax on import
- System-level ACLs only (project-level ACLs are not yet supported by this provider)

**Managing Multiple Policies:**

```hcl
# Import all ACL policies from a directory
resource "rundeck_acl_policy" "policies" {
  for_each = fileset(path.module, "policies/*.aclpolicy")
  
  name   = each.value
  policy = file("${path.module}/policies/${each.value}")
}
```

Then import each policy:

```bash
for policy in policies/*.aclpolicy; do
  filename=$(basename "$policy")
  resource_name=$(echo "$filename" | sed 's/.aclpolicy$//' | tr '-' '_')
  terraform import "rundeck_acl_policy.policies[\"$filename\"]" "$filename"
done
```

---

### Key Storage (Passwords & Keys)

Key storage resources have special handling for sensitive data during import.

#### Passwords

**Import ID Format:** `keys/path/to/password`

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_password" "db_password" {
  path     = "keys/production/db_password"
  password = "placeholder" # Will need to be updated after import
}
EOF

# Import the password
terraform import rundeck_password.db_password keys/production/db_password
```

**Important Notes:**
- **The actual password value cannot be retrieved** from Rundeck's key storage
- A placeholder hash is used during import
- You **must update the `password` attribute** after import with the actual value
- The provider will detect the difference and update the password on the next `terraform apply`

**Post-Import Workflow:**

```bash
# After import, update the password in your Terraform config
# Option 1: Direct value (not recommended for production)
password = "actual-password"

# Option 2: Environment variable
password = var.db_password

# Option 3: External secret management
password = data.vault_generic_secret.db.data["password"]

# Then apply to update Rundeck with the correct value
terraform apply
```

#### Private Keys

**Import ID Format:** `keys/path/to/private-key`

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_private_key" "ssh_key" {
  path         = "keys/production/ssh_key"
  key_material = file("\${path.module}/keys/id_rsa") # Update after import
}
EOF

# Import the private key
terraform import rundeck_private_key.ssh_key keys/production/ssh_key
```

**Important Notes:**
- Same limitation as passwords - **key material cannot be retrieved**
- Update `key_material` after import with the actual private key content
- The provider will update Rundeck's key storage on the next `terraform apply`

#### Public Keys

**Import ID Format:** `keys/path/to/public-key`

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_public_key" "ssh_pubkey" {
  path         = "keys/production/ssh_pubkey"
  key_material = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDy..."
}
EOF

# Import the public key
terraform import rundeck_public_key.ssh_pubkey keys/production/ssh_pubkey
```

**Important Notes:**
- Public key content **can be retrieved** and is populated during import
- Verify the `key_material` matches your actual public key after import

---

### Runners (Enterprise)

Runners are an Enterprise feature available in Rundeck 5.17.0+.

#### System Runners

**Import ID Format:** `runner-id`

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_system_runner" "primary_runner" {
  runner_id   = "primary-runner-01"
  description = "Primary runner for job execution"
  
  tags = {
    environment = "production"
    region      = "us-east-1"
  }
}
EOF

# Import the system runner
terraform import rundeck_system_runner.primary_runner primary-runner-01
```

#### Project Runners

**Import ID Format:** `project-name:runner-id`

**Example:**

```bash
# Create the resource block
cat >> main.tf <<EOF
resource "rundeck_project_runner" "production_runner" {
  project     = "production"
  runner_id   = "runner-prod-01"
  description = "Production project runner"
}
EOF

# Import the project runner
terraform import rundeck_project_runner.production_runner production:runner-prod-01
```

**Important Notes:**
- The colon (`:`) separator is required for project runners
- Runner IDs must match what's registered with Rundeck
- Tags are imported and preserved

---

## Common Patterns

### Bulk Import Script

When migrating an entire Rundeck instance to Terraform, you may need to import many resources. Here's a helper script:

```bash
#!/bin/bash
# bulk-import.sh - Import all jobs from a Rundeck project

PROJECT="production"
RUNDECK_URL="${RUNDECK_URL:-http://localhost:4440}"
AUTH_TOKEN="${RUNDECK_AUTH_TOKEN}"

# Fetch all jobs in the project
jobs=$(curl -s -H "X-Rundeck-Auth-Token: $AUTH_TOKEN" \
  "$RUNDECK_URL/api/56/project/$PROJECT/jobs" | jq -r '.[] | @base64')

# Import each job
for job in $jobs; do
  _jq() {
    echo "$job" | base64 --decode | jq -r "$1"
  }
  
  name=$(_jq '.name')
  id=$(_jq '.id')
  safe_name=$(echo "$name" | tr ' ' '_' | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_]//g')
  
  echo "Importing job: $name (ID: $id)"
  
  # Create resource block if it doesn't exist
  if ! grep -q "resource \"rundeck_job\" \"$safe_name\"" main.tf 2>/dev/null; then
    cat >> main.tf <<EOF

resource "rundeck_job" "$safe_name" {
  name         = "$name"
  project_name = "$PROJECT"
  description  = "Imported job"
  
  command {
    shell_command = "echo 'Update this job configuration'"
  }
}
EOF
  fi
  
  # Import the job
  terraform import "rundeck_job.$safe_name" "$id"
done

echo "Import complete. Review main.tf and update configurations as needed."
```

### Gradual Migration

If you have a large Rundeck installation, consider a gradual migration approach:

1. **Start with infrastructure resources:** Projects, ACL policies
2. **Import critical jobs:** High-priority or frequently-run jobs
3. **Import supporting resources:** Key storage, webhooks
4. **Import remaining jobs:** Less critical jobs over time

This approach allows you to:
- Test Terraform changes in a controlled manner
- Minimize risk of disruption
- Learn best practices before managing all resources

### Using Modules

Organize imported resources into reusable modules:

```hcl
# modules/rundeck-project/main.tf
resource "rundeck_project" "project" {
  name        = var.project_name
  description = var.description
  
  resource_model_source {
    type   = "file"
    config = var.node_source_config
  }
}

# Import into the module
terraform import module.production.rundeck_project.project production
```

---

## Troubleshooting

### Import ID Not Found

**Error:**
```
Error: Cannot import non-existent remote object
```

**Solution:**
- Verify the resource exists in Rundeck
- Check your import ID format for the resource type
- Ensure your Rundeck credentials and URL are correct
- Verify API version compatibility (v56+ recommended)

### Configuration Drift After Import

**Error:**
```
Terraform will perform the following actions:
  # rundeck_job.my_job will be updated in-place
```

**Solution:**
- The imported state doesn't match your Terraform configuration
- Run `terraform show` to see the exact imported values
- Update your Terraform configuration to match
- Pay special attention to:
  - Nested blocks (commands, options, notifications)
  - Default values that differ from Rundeck's defaults
  - Computed attributes that shouldn't be set in config

### Complex Job Import Issues

**Problem:** Jobs with multiple commands, plugins, or orchestrators are hard to import correctly.

**Solution:**
1. Export the job definition from Rundeck:
   ```bash
   curl -H "X-Rundeck-Auth-Token: $AUTH_TOKEN" \
     "$RUNDECK_URL/api/56/job/$JOB_ID" | jq . > job-export.json
   ```

2. Use the exported JSON as a reference to build your Terraform config

3. Compare Rundeck's JSON schema with the [Terraform job schema](../r/job.html.md)

4. Test in a development environment before importing production jobs

### Webhook Auth Token Changed

**Problem:** After importing a webhook, the webhook URL changes.

**Expected Behavior:** This is normal - auth tokens cannot be imported for security reasons.

**Solution:**
1. Note the new webhook URL from `terraform apply` output
2. Update any external systems (GitHub, DataDog, etc.) with the new URL
3. Consider using Terraform outputs to track webhook URLs:
   ```hcl
   output "github_webhook_url" {
     value     = rundeck_webhook.github_deploy.url
     sensitive = true
   }
   ```

### Key Storage Import with Missing Values

**Problem:** Passwords and private keys show placeholder values after import.

**Expected Behavior:** Rundeck's key storage API doesn't allow retrieving secret values.

**Solution:**
1. After import, update your Terraform config with the actual secret values
2. Use secure methods to manage secrets:
   - Terraform variables marked as `sensitive`
   - Environment variables
   - External secret management (Vault, AWS Secrets Manager, etc.)
3. Run `terraform apply` to update Rundeck with the correct values

### Import Fails with "Invalid Import ID"

**Error:**
```
Error: Invalid Import ID
Expected import identifier in format 'X', got: 'Y'
```

**Solution:**
- Each resource type has a specific import ID format (see [Resource-Specific Import Instructions](#resource-specific-import-instructions))
- Double-check you're using the correct format for the resource type
- Common mistakes:
  - Using job name instead of job UUID
  - Missing the project name for webhooks or project runners
  - Forgetting the `.aclpolicy` extension for ACL policies
  - Incorrect separator (`:` vs `/`)

---

## Best Practices

1. **Always test imports in a development environment first**
2. **Use version control for your Terraform configurations**
3. **Document non-obvious configuration decisions** (especially for complex jobs)
4. **Import related resources together** (e.g., project + jobs + webhooks)
5. **Validate with `terraform plan`** before applying changes
6. **Keep a backup of your Rundeck instance** before making bulk changes
7. **Use consistent naming conventions** for Terraform resource names
8. **Consider using `terraform_remote_state`** for large, multi-team Rundeck installations

---

## Additional Resources

- [Rundeck API Documentation](https://docs.rundeck.com/docs/api/)
- [Terraform Import Documentation](https://www.terraform.io/docs/cli/commands/import.html)
- [Provider Configuration](../index.html.markdown)
- [Resource Documentation](../r/)

For issues or questions, please visit:
- [GitHub Issues](https://github.com/rundeck/terraform-provider-rundeck/issues)
- [Rundeck Community](https://docs.rundeck.com/docs/introduction/getting-help.html)
