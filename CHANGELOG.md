## 1.0.0

**Major Release - Provider Modernization**

This release modernizes the Terraform Provider to use the Terraform Plugin Framework and establishes JSON-only API interactions with Rundeck.

**Breaking Changes:**
- **Minimum Rundeck version:** 5.0.0 (API v46+)
- **Minimum Go version:** 1.24+
- XML API interactions fully removed - all operations use JSON

**Enhancements:**
- Migrated Job resource from Plugin SDK to modern Plugin Framework
- Implemented native HCL nested blocks for all job configurations
- Added full support for execution lifecycle plugins
- Implemented project schedule functionality with validation
- Eliminated all plan drift issues for Job resource
- All API calls now use JSON format exclusively

**Compatibility:**
- Existing Terraform plan files from previous versions should work without modification
- If you experience issues with existing plans, please open an issue on the repository

**Testing:**
- All acceptance tests passing (23 passed, 2 validation enhancements deferred to future version)
- Job tests: 18 passed
- Runner tests: 5 passed (system_runner, project_runner)
- Enterprise features fully tested and working

---

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
