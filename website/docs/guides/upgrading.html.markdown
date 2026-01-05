---
layout: "rundeck"
page_title: "Upgrading the Rundeck Provider"
sidebar_current: "docs-rundeck-guide-upgrading"
description: |-
  Guide for upgrading to newer versions of the Terraform Rundeck Provider
---

# Upgrading the Rundeck Provider

This guide covers important changes and migration steps for major version upgrades.

---

## Upgrading to v1.1.0

Version 1.1.0 includes important bug fixes. **Please review these changes before upgrading.**

### Important Changes

#### Notification Ordering Requirement

**ACTION REQUIRED:** Notifications must be defined in alphabetical order by type to prevent plan drift. The Rundeck API returns notifications sorted alphabetically, so your Terraform configuration must match this order.

**Valid notification types (in alphabetical order):**
- `on_avg_duration`
- `on_failure`
- `on_retryable_failure`
- `on_start`
- `on_success`

```hcl
# Correct syntax (v1.1.0+ - alphabetical order)
resource "rundeck_job" "example" {
  name         = "Example Job"
  project_name = "my-project"
  
  # on_failure comes before on_success alphabetically
  notification {
    type = "on_failure"
    webhook_urls = ["https://example.com/webhook"]
  }
  
  notification {
    type = "on_success"
    email {
      recipients = ["user@example.com"]
      attach_log = true
    }
  }
  
  command {
    shell_command = "echo hello"
  }
}
```

**Why:** The Rundeck API returns notifications sorted alphabetically by type. If your Terraform configuration defines them in a different order, Terraform will detect plan drift and attempt to reorder them, causing "Provider produced inconsistent result" errors. By defining notifications in alphabetical order, your configuration matches what the API returns, eliminating plan drift.

**Migration Steps:**
1. Review all `notification {` blocks in your `.tf` files
2. Reorder them alphabetically by `type` (`on_avg_duration`, `on_failure`, `on_retryable_failure`, `on_start`, `on_success`)
3. Run `terraform plan` to verify no unexpected changes

### Bug Fixes

- **Fixed notification ordering requirement** ([#209](https://github.com/rundeck/terraform-provider-rundeck/issues/209)) - The provider now sorts notifications alphabetically by type before sending to the API and after reading from the API, ensuring consistent state. This eliminates confusing "Provider produced inconsistent result" errors. Users must define notifications in alphabetical order to match the API's behavior.
- **Fixed lossy job imports** ([#213](https://github.com/rundeck/terraform-provider-rundeck/issues/213)) - Job imports now correctly extract all fields from the Rundeck API.
- **Added UUID support for error_handler job references** ([#212](https://github.com/rundeck/terraform-provider-rundeck/issues/212)) - Error handler job references now support UUID-based references.

---

## Upgrading to v1.0.0 (Not Recommended)

**Note:** v1.0.0 introduced a notification ordering limitation that was fixed in v1.1.0. **We recommend upgrading directly to v1.1.0 by following the steps above** to avoid the notification ordering requirement and the subsequent syntax change.

If you plan to upgrade to v1.0.0 and stop there (not upgrading to v1.1.0+), please review these migration steps:

### Overview

v1.0.0 brings:
- **Zero plan drift** for all resource types
- **100% test pass rate** (38 acceptance tests)
- Plugin Framework migration (SDKv2 removed)
- JSON-only API (XML support removed)
- All command types fully functional
- Multiple critical bug fixes

### Breaking Changes

#### Minimum Versions

- **Rundeck:** 5.0.0+ (API v46+) required
- **Rundeck Enterprise:** 5.17.0+ (API v56) required for Enterprise Runner resources
- **Go:** 1.24+ required for building from source
- **Terraform:** 0.12+ (tested through 1.9)

#### Removed Features

- **SDKv2 Provider:** Now uses Plugin Framework exclusively
- **XML API Support:** All interactions use JSON
- **Plugin Mux:** Single provider implementation for better stability

### Required Configuration Changes

#### 1. Notification Ordering Requirement

**ACTION REQUIRED:** Notifications must be defined in alphabetical order by type. This requirement exists in v1.0.0 and continues in v1.1.0+. The Rundeck API returns notifications sorted alphabetically, so your Terraform configuration must match this order to prevent plan drift.

**Valid notification types (in alphabetical order):**
- `on_avg_duration`
- `on_failure`
- `on_retryable_failure`
- `on_start`
- `on_success`

```hcl
# Correct syntax (v1.0.0+ - alphabetical order)
resource "rundeck_job" "example" {
  name         = "Example Job"
  project_name = "my-project"
  
  # on_failure comes before on_success alphabetically
  notification {
    type = "on_failure"
    webhook_urls = ["https://example.com/webhook"]
  }
  
  notification {
    type = "on_success"
    email {
      recipients = ["user@example.com"]
      attach_log = true
    }
  }
  
  command {
    shell_command = "echo hello"
  }
}
```

**Why:** The Rundeck API returns notifications sorted alphabetically by type. If your Terraform configuration defines them in a different order, Terraform will detect plan drift and attempt to reorder them, causing "Provider produced inconsistent result" errors. By defining notifications in alphabetical order, your configuration matches what the API returns, eliminating plan drift.

**Migration Steps:**
1. Review all `notification {` blocks in your `.tf` files
2. Reorder them alphabetically by `type` (`on_avg_duration`, `on_failure`, `on_retryable_failure`, `on_start`, `on_success`)
3. Run `terraform plan` to verify no unexpected changes

**Note:** v1.1.0 adds validation to catch ordering issues at plan time with helpful error messages. v1.0.0 will only show the error during apply.

#### 2. Execution Lifecycle Plugins

**CRITICAL:** Jobs with execution lifecycle plugins in previous versions **did not apply correctly** due to an API format bug (array vs. map).

**Action Required:**
1. After upgrading, run `terraform plan` on jobs with lifecycle plugins
2. You **will** see drift - this is expected and correct
3. Run `terraform apply` to properly configure the plugins in Rundeck for the first time

**Example:**
```hcl
resource "rundeck_job" "example" {
  name         = "Job with Lifecycle Plugin"
  project_name = "my-project"
  
  execution_lifecycle_plugin {
    type = "killhandler"
    config = {
      killChilds = "true"
    }
  }
  
  command {
    shell_command = "long-running-process"
  }
}
```

**What was broken:** Previous versions sent `[{type: "killhandler", config: {...}}]` (array) but Rundeck expects `{killhandler: {killChilds: "true"}}` (map). Your configuration was correct, but the plugins were never actually applied in Rundeck.

### Recommended Configuration Changes

#### 3. Option Enforcement (Optional but Recommended)

**Recommended:** Explicitly set `require_predefined_choice` for options with predefined values.

```hcl
resource "rundeck_job" "example" {
  name         = "Example Job"
  project_name = "my-project"
  
  option {
    name          = "environment"
    value_choices = ["dev", "staging", "prod"]
    require_predefined_choice = true  # Explicitly set for clarity
  }
  
  command {
    shell_command = "deploy.sh ${option.environment}"
  }
}
```

**Note:** The provider automatically infers `true` when `value_choices` is present but the API doesn't explicitly return the field. Being explicit improves configuration readability and prevents confusion.

**Migration Steps:**
1. Optional - add `require_predefined_choice = true` to options with `value_choices`
2. If omitted, provider will infer correctly (no drift)

### No Action Required

The following are handled automatically by the provider:

#### Schedule Normalization

Cron schedules are automatically normalized to match Rundeck's format.

```hcl
# You write this:
schedule = "0 0 12 * * * *"

# Rundeck normalizes to:
# "0 0 12 ? * * *" (asterisk becomes question mark for day-of-month)

# Provider handles this automatically - no drift!
```

#### Runner Tag Normalization

Tags are compared semantically (case-insensitive, sorted).

```hcl
# These are semantically equal - no drift:
tag_names = "Production,API,Test"    # Your input
tag_names = "api,production,test"    # Rundeck's normalized response
```

#### Command Type Fixes

All command types now work correctly:

- **Script interpreter:** `args_quoted` properly handled as boolean
- **Script file arguments:** Arguments now map to correct API field (`args`)
- **Step & Node Step Plugins:** Structure corrected (flat at command level)
- **Job References:** `node_step` and `dispatch` options now supported
- **Error Handlers:** `keep_going_on_success` now properly supported

**No action needed** - these bugs are fixed and will work correctly on upgrade.

### Testing Before Upgrade

We **strongly recommend** testing your configuration before upgrading production:

```bash
# 1. Clone the provider repo
git clone https://github.com/rundeck/terraform-provider-rundeck.git
cd terraform-provider-rundeck

# 2. Checkout v1.0.0 branch (or PR for pre-release testing)
git checkout provider-modernization  # For PR #203
# OR after release:
# git checkout tags/v1.0.0

# 3. Test your configuration
export RUNDECK_URL="http://your-rundeck:4440"
export RUNDECK_AUTH_TOKEN="your-api-token"
cd test/enterprise
./test-custom.sh /path/to/your/terraform/config
```

This will:
- Build the new provider locally
- Run `terraform plan` with your actual configuration
- Show you any drift before you upgrade
- Help identify notification ordering issues

### What's Fixed in v1.0.0

#### Plan Drift Eliminated

- **Notifications:** Complete rewrite - webhooks, plugins, email all work correctly
- **Schedules:** Automatic normalization matches Rundeck's format
- **Options:** `require_predefined_choice` correctly inferred
- **Commands:** All command types validated against real API responses
- **Runner tags:** Semantic equality prevents false drift

#### Critical Bug Fixes

**[#156](https://github.com/rundeck/terraform-provider-rundeck/issues/156) - EOF Error on Apply**  
FIXED - Updated Go SDK to v1.2.0 with proper error handling

**[#126](https://github.com/rundeck/terraform-provider-rundeck/issues/126) - multi_value_delimiter Not Working**  
FIXED - Corrected field mapping in option converter

**[#198](https://github.com/rundeck/terraform-provider-rundeck/issues/198) - Password State Corruption**  
FIXED - Updated SDK error handling

#### Command Type Fixes

**Script Interpreter:**
- Fixed `args_quoted` to properly handle boolean values
- Fixed `invocation_string` mapping to API's `scriptInterpreter` field

**Script Options:**
- Fixed `script_file_args` to map to API's `args` field (was incorrectly `scriptargs`)
- Fixed `file_extension` handling
- Fixed `expand_token_in_script_file` for script files and URLs

**Plugins:**
- Fixed step plugin structure (flat at command level, not nested)
- Fixed node step plugin structure
- Fixed `nodeStep` boolean handling
- Fixed `config` vs `configuration` field naming

**Job References:**
- Added `node_step` support (run job once per node)
- Added `dispatch` block support for thread control, keep_going, rank options

**Error Handlers:**
- Added `keep_going_on_success` support
- Fixed all script-related fields in error handlers

**Execution Lifecycle Plugins:**
- Fixed API format (map instead of array) - **critical fix**

### Upgrade Checklist

Use this checklist when upgrading:

- [ ] Review Rundeck version (5.0.0+ required)
- [ ] Review jobs with multiple notifications
- [ ] Reorder notifications alphabetically in `.tf` files
- [ ] Identify jobs with execution lifecycle plugins
- [ ] Update provider version in `required_providers` block
- [ ] Run `terraform init -upgrade`
- [ ] Run `terraform plan` to review changes
- [ ] Review drift for lifecycle plugins (expected)
- [ ] Apply changes in non-production first
- [ ] Verify jobs execute correctly in Rundeck UI
- [ ] Repeat for production

### Post-Upgrade

After successfully upgrading:

1. Verify jobs execute correctly
2. Check notifications are being delivered
3. Verify lifecycle plugins are active (check job execution behavior)
4. Remove any workarounds for old bugs (e.g., manual lifecycle plugin configuration)

---

## Getting Help

If you encounter issues during an upgrade:

1. **Check the [CHANGELOG](https://github.com/rundeck/terraform-provider-rundeck/blob/main/CHANGELOG.md)** for detailed changes
2. **Review [open issues](https://github.com/rundeck/terraform-provider-rundeck/issues)** for known problems
3. **Test with `test-custom.sh`** before reporting bugs
4. **Include your Terraform configuration** (sanitized) when reporting issues
5. **Specify Rundeck version** and whether it's OSS or Enterprise

**Questions?** Open an issue on [GitHub](https://github.com/rundeck/terraform-provider-rundeck/issues) or ask in the [Rundeck Community Discussions](https://github.com/rundeck/rundeck/discussions).

