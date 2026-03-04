---
layout: "rundeck"
page_title: "Upgrading the Rundeck Provider"
sidebar_current: "docs-rundeck-guides-upgrading"
description: |-
  Guide for upgrading between major versions of the Rundeck Terraform Provider
---

# Upgrading the Rundeck Provider

This guide covers upgrading between major versions of the Rundeck Terraform Provider.

## Version Compatibility

| Provider Version | Minimum Rundeck Version | Minimum Terraform Version |
|-----------------|------------------------|---------------------------|
| v1.x            | 5.0.0 (API v46+)       | 0.13.x                    |
| v0.5.x          | 3.x+                   | 0.12.x                    |

## Upgrading from v0.5.x to v1.x

Version 1.0.0 was a major rewrite from Terraform SDKv2 to the Plugin Framework with several breaking changes. Follow these steps to upgrade safely.

### Step 1: Backup Your State

Before upgrading, backup your Terraform state:

```bash
terraform state pull > terraform.tfstate.backup
```

### Step 2: Update Provider Configuration

Update your `required_providers` block to use the new provider namespace:

**Before (v0.5.x):**
```hcl
terraform {
  required_providers {
    rundeck = {
      source  = "terraform-providers/rundeck"
      version = "~> 0.5.0"
    }
  }
}
```

**After (v1.x):**
```hcl
terraform {
  required_providers {
    rundeck = {
      source  = "rundeck/rundeck"
      version = "~> 1.2.0"
    }
  }
}
```

### Step 3: Migrate Provider Namespace in State

Run the following command to update the provider namespace in your state file:

```bash
terraform state replace-provider registry.terraform.io/terraform-providers/rundeck registry.terraform.io/rundeck/rundeck
```

This updates the state without modifying any resources.

### Step 4: Update Provider Version

Update your provider version and reinitialize:

```bash
terraform init -upgrade
```

### Step 5: Review Configuration Changes

#### Notification Ordering (Required)

Notifications must be defined in **alphabetical order** by type. Rundeck's API returns notifications sorted alphabetically, and the provider enforces this to prevent drift.

**Before:**
```hcl
notification {
  type = "on_success"
  email {
    recipients = ["team@example.com"]
  }
}

notification {
  type = "on_failure"
  email {
    recipients = ["team@example.com"]
  }
}
```

**After (alphabetical order):**
```hcl
notification {
  type = "on_failure"
  email {
    recipients = ["team@example.com"]
  }
}

notification {
  type = "on_success"
  email {
    recipients = ["team@example.com"]
  }
}
```

Valid notification types in order:
- `on_avg_duration`
- `on_failure`
- `on_retryable_failure`
- `on_start`
- `on_success`

#### Project Extra Config (v1.1.0+)

If you use `extra_config` in project resources, update key format from slash to dot notation:

**Before:**
```hcl
resource "rundeck_project" "example" {
  extra_config = {
    "project/label"       = "My Project"
    "project/description" = "Example project"
  }
}
```

**After:**
```hcl
resource "rundeck_project" "example" {
  extra_config = {
    "project.label"       = "My Project"
    "project.description" = "Example project"
  }
}
```

#### Schedule Format (Optional)

Cron expressions are now normalized to include seconds. This is cosmetic and won't trigger replacement:

- `"0 30 * * * ? *"` becomes `"0 30 * * * * *"`

You can update your configuration to include the seconds field to match, but it's not required.

### Step 6: Run Plan and Review

Run a plan to see what changes will occur:

```bash
terraform plan
```

**Expected changes:**
- Default value updates (cosmetic, no replacement)
- Field additions like `success_on_empty_node_filter` (cosmetic)
- Schedule format normalization (cosmetic)
- Notification reordering if not alphabetical (update in-place)

**Unexpected changes (indicating issues):**
- Resources showing as "must be replaced"
- ID changing to `(known after apply)`

If you see unexpected replacements, verify:
1. You ran the `terraform state replace-provider` command in Step 3
2. Your `required_providers` block uses `source = "rundeck/rundeck"`
3. Your Terraform state has the correct provider namespace

### Step 7: Apply Changes

Once you've verified the plan looks reasonable:

```bash
terraform apply
```

## Common Issues and Solutions

### Issue: All Resources Show as "Must Be Replaced"

**Symptom:** Resources show `id = "..." -> (known after apply)` and "must be replaced"

**Cause:** Provider namespace mismatch in state file

**Solution:** Run the state migration command:
```bash
terraform state replace-provider registry.terraform.io/terraform-providers/rundeck registry.terraform.io/rundeck/rundeck
```

### Issue: Plan Shows Notification Changes

**Symptom:** Plan wants to update notification blocks

**Cause:** Notifications not in alphabetical order

**Solution:** Reorder notification blocks alphabetically by type (see Step 5)

### Issue: Runner Selector Fields Being Removed

**Symptom:** Plan shows `runner_selector_filter_mode` and `runner_selector_filter_type` being removed

**Cause:** These fields now default to empty/null when not explicitly set

**Solution:** This is cosmetic. The fields will be removed from state but functionality remains intact. If you want to keep them in state, explicitly set them:
```hcl
runner_selector_filter       = "tags: production"
runner_selector_filter_mode  = "TAGS"
runner_selector_filter_type  = "TAG_FILTER_AND"
```

### Issue: Execution Lifecycle Plugins Not Working

**Symptom:** Plugins defined before v1.0.0 have no effect in Rundeck

**Cause:** Pre-1.0.0 versions sent incorrect API format (array instead of map), causing plugins to be silently ignored

**Solution:** After upgrading, re-apply jobs with lifecycle plugins. The v1.0.0+ format will work correctly.

### Issue: Jobs With Empty Choice Values

**Symptom:** Validation error: "value_choices contains empty values"

**Cause:** v1.1.0+ validates that `value_choices` contains no empty strings (Rundeck API filters them out causing drift)

**Solution:** Remove empty strings from `value_choices`:
```hcl
# Before
value_choices = ["", "dev", "prod"]

# After
value_choices = ["dev", "prod"]
```

## Feature Changes

### New in v1.0.0+

- **UUID-based job references** - Jobs can reference other jobs by UUID for immutability
- **Full notification support** - Email, webhook, and plugin notifications work correctly
- **Execution lifecycle plugins** - Proper API format ensures plugins are applied
- **Better error messages** - API responses included in error messages for troubleshooting

### New in v1.1.0+

- **Improved validation** - Catches configuration errors at plan time
- **Better import support** - Job imports extract all fields correctly

### New in v1.2.0+

- **Webhook resource** - Manage Rundeck webhooks as code
- **User-Agent tracking** - Provider includes User-Agent header for analytics

## Getting Help

If you encounter issues during upgrade:

1. **Check the [CHANGELOG](https://github.com/rundeck/terraform-provider-rundeck/blob/main/CHANGELOG.md)** for detailed release notes
2. **Review existing issues** on [GitHub](https://github.com/rundeck/terraform-provider-rundeck/issues)
3. **Open a new issue** with:
   - Your current provider version
   - Target provider version
   - Rundeck version
   - Full plan output (sanitized)
   - Provider configuration

## Best Practices

- **Test in non-production first** - Upgrade in dev/staging before production
- **Review the plan carefully** - Ensure no unexpected replacements
- **Backup state files** - Keep backups before major version changes
- **Update gradually** - Don't skip multiple major versions if possible
- **Read the CHANGELOG** - Each release includes detailed notes on changes

## Version-Specific Notes

### v1.0.0 Breaking Changes

- Minimum Rundeck version increased to 5.0.0
- SDKv2 implementation removed (Plugin Framework only)
- XML API support removed (JSON only)

### v1.1.0 Breaking Changes

- `extra_config` key format changed from slash to dot notation

### v1.2.0 Changes

- No breaking changes from v1.1.x
- Adds webhook resource
- Bug fixes for import and option enforcement
