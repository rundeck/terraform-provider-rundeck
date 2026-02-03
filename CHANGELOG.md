## 1.1.2

**Bug Fixes**

### Job Resource
- **Fixed validation error with variable-based value_choices** ([#218](https://github.com/rundeck/terraform-provider-rundeck/issues/218)) - Job options with `require_predefined_choice = true` and `value_choices` set to a Terraform variable no longer fail validation during plan phase. The provider now correctly skips validation when values are unknown (e.g., from variables or computed values) and validates during apply phase when values are concrete. This fixes the regression introduced in v1.1.0 where users received "Missing value choices" errors even when variables were properly defined.

- **Fixed node_filter_exclude_query not working** - The `node_filter_exclude_query` field was not being sent to Rundeck correctly. The provider was using `excludeFilter` as the JSON field name, but Rundeck's API expects `filterExclude`. Jobs with node exclusion filters now work correctly:
  ```hcl
  resource "rundeck_job" "example" {
    # ...
    node_filter_query         = "tags: webserver"
    node_filter_exclude_query = "name: maintenance-*"
  }
  ```

---

## 1.1.1

**Bug Fixes**

### Job Resource
- **Fixed schedule day-of-month support** ([#215](https://github.com/rundeck/terraform-provider-rundeck/issues/215)) - Jobs can now be scheduled on specific days of the month using the day-of-month field in cron expressions (e.g., `schedule = "0 0 09 10 * ? *"` for 9am on the 10th of each month). Previously, the provider incorrectly replaced day-of-month values with `?`, causing "Provider produced inconsistent result" errors. The fix uses Rundeck's `crontab` schedule format to preserve the complete cron expression.

- **Fixed notification plugin parsing** - Plugin notifications now work correctly when the Rundeck API returns a single plugin as an object instead of an array. Previously, single plugin notifications were silently dropped, causing "block count changed from N to N-1" errors. The fix handles both API response formats:
  - Array format (multiple plugins): `"plugin": [{"type": "...", "configuration": {...}}]`
  - Single object format (one plugin): `"plugin": {"type": "...", "configuration": {...}}`

- **Fixed combined notification targets** - Notification blocks with multiple targets (e.g., email + plugin in the same block) now work correctly. Previously, the provider would read these as separate notification blocks, causing "block count changed from 1 to 2" errors. Now a single notification block with multiple targets is preserved correctly.

### Provider Authentication
- **Fixed excessive token creation** - The provider now reuses existing "terraform-token" entries instead of creating a new token on every Terraform run. Previously, each `terraform plan`, `apply`, or `refresh` would create a new "terraform-token" in Rundeck, leading to hundreds or thousands of unused tokens accumulating over time. The provider now checks for existing valid tokens before creating new ones. If multiple "terraform-token" entries exist (from previous versions), the provider will reuse the first valid one found. Users can manually clean up duplicate tokens via the Rundeck UI or API if desired.

---

## 1.1.0

**Important Changes**

### Breaking Changes

#### Project Resource - `extra_config` Format Change
- **`extra_config` key format** - Keys in `extra_config` now use dot notation (e.g., `"project.label"`) instead of slash notation (e.g., `"project/label"`). This matches the Rundeck API format directly and removes unnecessary conversion logic. **Migration:** Update any `extra_config` keys from slashes to dots. Example: `"project/label"` → `"project.label"`. This change affects users who used `extra_config` in v1.0.0.

### Job Resource - Notification Ordering Requirement
- **Notification ordering requirement** ([#209](https://github.com/rundeck/terraform-provider-rundeck/issues/209)) - Notifications must be defined in alphabetical order by type (`on_avg_duration`, `on_failure`, `on_retryable_failure`, `on_start`, `on_success`) to prevent plan drift. The Rundeck API returns notifications sorted alphabetically, so your Terraform configuration must match this order.

**Why:** The Rundeck API returns notifications sorted alphabetically by type. If your Terraform configuration defines them in a different order, Terraform will detect plan drift and attempt to reorder them, causing "Provider produced inconsistent result" errors. By defining notifications in alphabetical order, your configuration matches what the API returns, eliminating plan drift.

**Example:**
```hcl
# Correct - alphabetical order (on_failure before on_success)
notification {
  type = "on_failure"
  webhook_urls = ["https://example.com/webhook"]
}

notification {
  type = "on_success"
  email {
    recipients = ["user@example.com"]
  }
}
```

**Bug Fixes**

### Job Resource
- **Fixed notification ordering requirement** ([#209](https://github.com/rundeck/terraform-provider-rundeck/issues/209)) - The provider now sorts notifications alphabetically by type before sending to the API and after reading from the API, ensuring consistent state. This eliminates confusing "Provider produced inconsistent result" errors when notifications are defined in non-alphabetical order. Users must define notifications in alphabetical order to match the API's behavior.
- **Fixed lossy job imports** ([#213](https://github.com/rundeck/terraform-provider-rundeck/issues/213)) - Job imports now correctly extract all fields from the Rundeck API, including `continue_on_error`, `time_zone`, `node_filter_exclude_precedence`, `success_on_empty_node_filter`, and enhanced `node_step` handling for job references. Previously, many fields were missing from imported jobs, making imports unusable compared to v0.5.0.
- **Added UUID support for error_handler job references** ([#212](https://github.com/rundeck/terraform-provider-rundeck/issues/212)) - Error handler job references now support UUID-based references (like regular job references), making them immutable and resilient to job renames. Previously, error handler job references only supported name-based references which could break if the referenced job was renamed. This brings error handler job references to feature parity with regular job references, including support for `project_name`, `node_step`, `node_filters`, and other advanced options.
- **Added validation for empty choice values** - When `require_predefined_choice = true` for a job option, the provider now validates that `value_choices` contains at least one non-empty value and no empty strings. Empty strings cause plan drift because the Rundeck API filters them out. This validation catches the issue at plan time with a helpful error message.

### Documentation
- **Updated `extra_config` format** - `extra_config` keys now use dot notation (e.g., `"project.label"`) to match the Rundeck API format directly. Previously, keys used slash notation (e.g., `"project/label"`) which was unnecessarily converted. This change simplifies the implementation and matches the actual API format.
- Updated job resource documentation to clarify notification ordering requirement and add validation details

---

## 1.0.0

**Major Release - Provider Modernization**

This release modernizes the Terraform Provider to use the Terraform Plugin Framework exclusively and establishes JSON-only API interactions with Rundeck. This is a major milestone that eliminates all known plan drift issues and significantly improves reliability.

### Breaking Changes
- **Minimum Rundeck version:** 5.0.0 (API v46+)
- **Minimum Go version:** 1.24+
- **SDKv2 provider removed** - Now uses Plugin Framework exclusively
- **XML API interactions fully removed** - All operations use JSON

### Architecture Changes
- **Removed SDKv2 provider** - Simplified to Plugin Framework only for better maintainability
- **Removed Plugin Mux** - Single provider implementation eliminates schema inconsistency issues
- **Updated Go SDK versions:**
  - `github.com/rundeck/go-rundeck/rundeck` v1.2.0
  - `github.com/rundeck/go-rundeck/rundeck-v2` v1.2.0

### Enhancements

#### Job Resource
- **Migrated to Plugin Framework** - Modern implementation with full feature support
- **Native HCL nested blocks** - All job configurations use proper Terraform syntax
- **UUID support for job references** - Reference other jobs by immutable UUID instead of name
- **Fixed execution lifecycle plugins** - Now use correct map structure instead of array (CRITICAL FIX)
- **Complete notification support:**
  - Email notifications with `attach_log` support
  - Webhook notifications with `format`, `http_method`, and `webhook_urls`
  - Plugin notifications with proper configuration structure
  - Handles API format differences (Create: arrays, Read: objects)
  - Alphabetical ordering for deterministic behavior
- **Schedule normalization** - Automatically normalizes cron expressions to match Rundeck's format
- **Option enforcement inference** - Correctly infers `require_predefined_choice` when values are defined
- **Orchestrator support** - maxPercentage, subset, rankTiered configurations
- **Log limit support** - output, action, and status configuration
- **Global log filter support** - Plugin-based log filtering
- **Project schedules** - Enterprise feature for centralized scheduling

#### Runner Resources
- **Migrated to Plugin Framework** - system_runner and project_runner
- **Semantic equality for tags** - No more plan drift from Rundeck's tag normalization (lowercase/sorting)
- **Improved error messages** - Include API response details for troubleshooting

#### Provider Configuration
- **Username/password authentication** - Framework provider now supports auth_username/auth_password
- **Consistent schema** - URL and API version handling aligned across all resources

### Bug Fixes

#### Verified GitHub Issues
- ✅ [#156](https://github.com/rundeck/terraform-provider-rundeck/issues/156) - EOF error on `terraform apply` - FIXED
- ✅ [#126](https://github.com/rundeck/terraform-provider-rundeck/issues/126) - `multi_value_delimiter` not working - FIXED  
- ✅ [#198](https://github.com/rundeck/terraform-provider-rundeck/issues/198) - Password state corruption on connection errors - FIXED

#### Job Resource Plan Drift Fixes
- **Notifications:**
  - Fixed webhook field placement (top-level, not nested)
  - Fixed plugin configuration key (`configuration` not `config`)
  - Fixed notification ordering (alphabetical for determinism)
  - Fixed `attach_log` handling (null vs false)
  - Fixed multiple notification targets per event type
- **Schedules:**
  - Fixed cron normalization (`*` → `?` for day-of-month)
  - Implemented structured JSON format conversion
- **Options:**
  - Fixed `require_predefined_choice` inference from values presence
  - Fixed boolean field handling (null vs false)
- **Commands:**
  - Fixed script interpreter configuration (args_quoted, invocation_string)
  - Fixed script file arguments field mapping
  - Fixed step plugins and node step plugins structure
  - Fixed job reference configuration including node_step and dispatch options
  - Implemented complete round-trip JSON conversion for all command types
  - Added comprehensive test coverage for all command configurations
- **Orchestrator, Log Limit, Global Log Filter:**
  - Implemented TO/FROM JSON converters (were silently dropped before)

#### Runner Resource Plan Drift Fixes
- **Tag normalization** - Semantic equality prevents drift from Rundeck's tag sorting/lowercasing

### Testing

#### Test Coverage
- **38 passing acceptance tests** (100% pass rate)
  - 21 job tests (including integration and comprehensive tests)
  - 3 orchestrator tests
  - 5 runner tests (enterprise)
  - 9 other resource tests
- **Integration tests with API validation** - Direct API queries verify actual Rundeck storage
- **Comprehensive command type tests** - Validates all script, plugin, and job reference configurations


### Important Notes

**Execution Lifecycle Plugins:**
Previous versions sent incorrect format to API, causing plugins to be silently ignored. Jobs with lifecycle plugins should be recreated or updated after upgrading to ensure plugins are properly applied.

**Runner Tags:**
Tag normalization is now handled automatically via semantic equality. Existing configurations work as-is - tags can be specified in any order or case (e.g., `"production,api,test"` equals `"api,production,test"`).

**Notification Ordering:**
Notifications are returned in alphabetical order (onfailure, onstart, onsuccess, etc.) for deterministic behavior. Update your Terraform configurations to match this order to avoid unnecessary diffs.

**Option Enforcement:**
When an option has `value_choices` defined, Rundeck implicitly enforces values even if `enforcedValues` is not explicitly set in the API response. The provider now correctly infers `require_predefined_choice = true` in these cases.

### Compatibility
- Existing Terraform plan files from previous versions should work without modification
- Notification blocks may need reordering to match alphabetical sort
- Options with `value_choices` should explicitly set `require_predefined_choice = true`
- If you experience issues with existing plans, please open an issue on the repository

### Testing Your Configuration
We've provided test scripts to help validate your configurations:
- `test/enterprise/test-custom.sh` - Test your own Terraform plans with automatic build and drift detection
- `test/enterprise/comprehensive.sh` - Full enterprise feature validation
- See `test/enterprise/README.md` for detailed instructions

### Documentation
- Updated issue template with Rundeck version requirements and test script guidance
- Added troubleshooting section for common issues
- Documented notification ordering behavior
- Documented runner tag semantic equality behavior

---
## 0.5.5

**IMPORTANT**: 0.5.3 and 0.5.4 were not released publicly to Hashicorp. These versions were signed with a key that has been 
rotated for security purposes.

- No material changes in 0.5.5 except publicly making features from 0.5.4 and 0.5.3 avaialble in public registry.

## 0.5.4
- Support for Enterprise Runners (system_runner and project_runner resources)
- Introduces rundeck-v2 Go Client Library for runner management
- Refinement for Project Schedule tests (added RUNDECK_PROJECT_SCHEDULES_CONFIGURED flag)
- Updated README with comprehensive testing documentation

## 0.5.3
- Support for Project Schedules (Enterprise Feature)
- Support for Execution Lifecycle Plugins
- Fixed tests so all pass on latest Rundeck release and added ability to test on Enterprise version locally.  (See repo Readme)
- Fixed GitHub Workflow to properly run Docker and tests

## 0.5.2
- Adds two new command options for Rundeck jobs to support custom script file extensions and token expansion in external scripts.
- Introduces file_extension and expand_token_in_script_file to Terraform schema and mapping functions
- Introduces support for configuring webhook notifications in Rundeck jobs by adding the ability to specify payload formats and HTTP methods.

## 0.5.1
- Runner Selector support on Job Resource

## 0.5.0
- Added a fix which introduces validation in the CreateProject function of the Go client for the Rundeck provider to ensure a project with the same name does not already exist before attempting to create it.


## 0.4.9
- Added flags for using Job Reference steps.

## 0.4.8
- Added Job step option for `script_url`
- Added ability to reference jobs in other projects through the use of `project_name` in `job` command type.
- Added support for Log Filters on individual Job Steps
- Added Import capability for Job Definitions

## 0.4.7
- Added Password Resource
- Added Orchestrator option to Job Definition
- Added Hidden flag for Job Options

## 0.4.6
- Added Retry Delay setting to Job definition.
- Added ability to set Secure Options in Job Definition.
- Typos in provider descriptions.
- Minor fixes

## 0.4.5
* Added User/Password authentication

## 0.4.4
* Added ability to import project files.
* updated to use Go 1.19

## O.4.3
* Updated Documentation

## 0.4.1/0.4.2

* Community improvements and updates to modern Terraform model.

## 0.4.0 (August 02, 2019)

### Added:
* Job reference node filter override [#27](https://github.com/terraform-providers/terraform-provider-rundeck/pull/27)

### Fixed:
* Handle empty value options gracefully [#28](https://github.com/terraform-providers/terraform-provider-rundeck/pull/28)

## 0.3.0 (June 13, 2019)

### Added:

* **Terraform 0.12** update Terraform SDK to 0.12.1 ([#25](https://github.com/terraform-providers/terraform-provider-rundeck/pull/25))
* resource/job: Add attribute `notification` ([#24](https://github.com/terraform-providers/terraform-provider-rundeck/pull/24))

## 0.2.1 (June 12, 2019)

### Added:
* Job Schedule Enabled argument
* Job Execution Enabled argument

### FIXED:
* Executions and schedules getting disabled due to missing defaults

## 0.2.0 (May 13, 2019)

### Added:
* ACL Policy resource
* API Version provider option

### FIXED:
* Idempotency issue when node filter not set on job

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
