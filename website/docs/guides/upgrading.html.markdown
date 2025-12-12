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

## Upgrading to v1.0.0

Version 1.0.0 is a major release with significant improvements and some breaking changes. **Please review these migration steps before upgrading.**

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
- **Rundeck Enterprise:** 5.17.0+ (API v56) required for runner resources
- **Go:** 1.24+ required for building from source
- **Terraform:** 0.12+ (tested through 1.9)

#### Removed Features

- **SDKv2 Provider:** Now uses Plugin Framework exclusively
- **XML API Support:** All interactions use JSON
- **Plugin Mux:** Single provider implementation for better stability

### Required Configuration Changes

#### 1. Notification Ordering

**ACTION REQUIRED:** Notifications must be ordered alphabetically by type to prevent plan drift.

```hcl
# Correct (alphabetical order)
resource "rundeck_job" "example" {
  name         = "Example Job"
  project_name = "my-project"
  
  notification { type = "on_failure" ... }
  notification { type = "on_start" ... }
  notification { type = "on_success" ... }
  
  command {
    shell_command = "echo hello"
  }
}

# Incorrect (will show drift)
resource "rundeck_job" "example" {
  name         = "Example Job"
  project_name = "my-project"
  
  notification { type = "on_success" ... }
  notification { type = "on_failure" ... }  # Wrong order!
  
  command {
    shell_command = "echo hello"
  }
}
```

**Why:** Rundeck's API returns notifications as an object (not array). The provider sorts them alphabetically for deterministic state management.

**Migration Steps:**
1. Review all jobs with multiple notifications
2. Reorder them alphabetically in your `.tf` files
3. Run `terraform plan` to verify no unexpected changes

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

