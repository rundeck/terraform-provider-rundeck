## 1.0.1

**Bug Fixes**

### Job Resource
- **Fixed notification ordering requirement** ([#209](https://github.com/rundeck/terraform-provider-rundeck/issues/209)) - Notifications can now be defined in any order in your Terraform configuration. The provider automatically sorts them alphabetically before sending to Rundeck's API, eliminating confusing "Provider produced inconsistent result" errors. Previously, notifications had to be manually arranged in alphabetical order to avoid plan drift.
- **Added UUID support for error_handler job references** ([#212](https://github.com/rundeck/terraform-provider-rundeck/issues/212)) - Error handler job references now support UUID-based references (like regular job references), making them immutable and resilient to job renames. Previously, error handler job references only supported name-based references which could break if the referenced job was renamed. This brings error handler job references to feature parity with regular job references, including support for `project_name`, `node_step`, `node_filters`, and other advanced options.

### Documentation
- **Fixed `extra_config` example in project resource** ([#210](https://github.com/rundeck/terraform-provider-rundeck/issues/210)) - Corrected documentation examples to use `"project/label"` instead of `"project.label"`. Rundeck uses forward slashes as separators in project configuration keys, not dots. Using dots causes plan drift as Rundeck normalizes them to forward slashes.
- Updated job resource documentation to reflect that notification ordering is now handled automatically (v1.0.1+)

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
