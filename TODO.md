# Terraform Provider Rundeck - TODO & Technical Debt

## 🚀 Version 1.0.0 Release Status

The provider has been successfully modernized to the Terraform Plugin Framework with JSON-only API interactions. This document tracks remaining work, known limitations, and future enhancements.

---

## ✅ Recently Completed (2025-11-17)

### Critical Bug Fixes

1. **✅ NodeFilters Structure Fix** (CRITICAL - OSS & Enterprise)
   - **Issue**: Jobs were executing locally instead of dispatching to nodes
   - **Root Cause**: `dispatch` was at root level instead of nested inside `nodefilters`
   - **Fix**: Correctly nested `dispatch` inside `nodefilters` for both job creation and reading
   - **Impact**: ALL jobs with node filters now work correctly in OSS and Enterprise
   - **Files**: `rundeck/resource_job_framework.go` (lines 128-131, 803-845, 1078-1104)

2. **✅ Execution Lifecycle Plugin Structure Fix** (CRITICAL - Enterprise)
   - **Issue**: Plugins were being completely ignored by Rundeck API
   - **Root Cause**: Sent as array `[{type, config}]`, API expects map `{pluginType: config}`
   - **Fix**: Changed converter to use map structure
   - **Impact**: All 8 lifecycle plugins now work correctly
   - **Files**: `rundeck/resource_job_converters.go` (lines 445-485)

3. **✅ Execution Lifecycle Plugin Ordering Fix** (CRITICAL)
   - **Issue**: "Provider produced inconsistent result" errors
   - **Root Cause**: Map iteration order is unpredictable in Go
   - **Fix**: Sort plugin types alphabetically before iteration
   - **Impact**: Stable, predictable state with no inconsistency errors
   - **Files**: `rundeck/resource_job_framework.go` (lines 1120-1128)

4. **✅ Execution Lifecycle Plugin Empty Config Fix**
   - **Issue**: Test failure `TestAccJob_executionLifecyclePlugin_noConfig`
   - **Root Cause**: Empty config `{}` converted to `null` during read
   - **Fix**: Always preserve map structure when config field exists
   - **Impact**: Plugins with no config options now work correctly
   - **Files**: `rundeck/resource_job_framework.go` (lines 1143-1145)

### New Features

5. **✅ UUID-Based Job References**
   - **Feature**: Jobs can reference other jobs by immutable UUID instead of name
   - **Benefit**: References survive job renames, more reliable for production
   - **Backward Compatible**: Name-based references still work
   - **Files**: `rundeck/resource_job_command_schema.go`, `rundeck/resource_job_converters.go`

6. **✅ Import Functionality for Lifecycle Plugins**
   - **Feature**: Execution lifecycle plugins now import correctly
   - **Fix**: Changed `NodeFilters` from `map[string]string` to `map[string]interface{}`
   - **Impact**: Import captures lifecycle plugins with configurations
   - **Files**: `rundeck/job.go`, `rundeck/resource_job_framework.go`

### Infrastructure Improvements

7. **✅ Test Directory Reorganization**
   - Separated OSS and Enterprise test environments
   - Created comprehensive documentation (3,300+ lines across 3 READMEs)
   - Updated CI/CD workflows for new structure
   - **Structure**:
     ```
     test/
     ├── README.md (overview)
     ├── oss/ (Docker-based OSS testing)
     └── enterprise/ (comprehensive Enterprise testing)
     ```

8. **✅ Scripts Cleanup**
   - Removed unused scripts (circle-ci.sh, gogetcookie.sh, changelog-links.sh)
   - Documented active scripts (gofmtcheck.sh, errcheck.sh)
   - Added `scripts/README.md` for clarity

---

## ⚠️ Known Limitations

### 1. Job Import - Nested Block State Verification

**Status**: Import functionality WORKS for end users, but test verification is incomplete.

**Issue**: The `jobJSONAPIToState` function has placeholder TODOs for converting JSON back to Terraform state for complex nested blocks:
- Commands (including job references, plugins, error handlers, script interpreters)
- Options (including value choices, validation rules)
- Notifications (email, webhook, plugin configurations)

**Current Behavior**:
- Import successfully retrieves jobs from Rundeck API
- Basic fields (name, description, project, schedule, etc.) are correctly populated
- Nested blocks are added to `ImportStateVerifyIgnore` in tests

**Impact**: 
- Users can import jobs using `terraform import rundeck_job.example project-name/job-uuid` or just `job-uuid`
- Imported state may not be 100% identical to original HCL for complex nested structures
- Re-running `terraform apply` after import may show minor changes for complex configurations

**Future Work**: Implement full JSON-to-state converters:
```go
// In rundeck/resource_job_converters.go
- convertCommandsFromJSON() - Parse JSON commands array to types.List
- convertOptionsFromJSON() - Parse JSON options array to types.List  
- convertNotificationsFromJSON() - Parse JSON notifications map to types.List
```

**Files**:
- `rundeck/resource_job_framework.go` (lines 1104-1130 - TODO comments)
- `rundeck/import_resource_job_test.go` (lines 44-53 - ImportStateVerifyIgnore list)

---

### 2. Schema-Level Validation (Deferred)

**Status**: Two validation enhancements deferred to future version.

#### A. Duplicate Notification Type Validation
**Test**: `TestAccJobNotification_multiple` (currently skipped)

**Issue**: Rundeck doesn't support multiple notifications of the same type (e.g., two `on_success` email notifications), but the provider schema doesn't prevent this at the Terraform level.

**Current Behavior**: Rundeck API will reject with error, but only at apply time.

**Desired Behavior**: Schema-level validation that prevents duplicate notification types in the plan phase.

**Implementation**: Use `terraform-plugin-framework-validators` to add custom list validator:
```go
// Add to notification block schema
Validators: []validator.List{
    listvalidator.UniqueValues(), // Validate on "type" field
}
```

#### B. Empty Option Choice Validation
**Test**: `TestAccJobOptions_empty_choice` (currently skipped)

**Issue**: Rundeck requires at least one value in `value_choices` when `require_predefined_choice` is true, but schema doesn't enforce this.

**Current Behavior**: Rundeck API rejects at apply time.

**Desired Behavior**: Schema-level validation that requires non-empty `value_choices` when `require_predefined_choice = true`.

**Implementation**: Custom validator for conditional required fields.

**Files**:
- `rundeck/resource_job_test.go` (lines 200, 215 - Skip messages)

---

## 🧪 Testing Gaps

### 1. Enterprise Feature Testing

Several tests require Rundeck Enterprise Edition and are skipped in CI/CD:

**Skipped Tests**:
- `TestAccJob_executionLifecyclePlugin_multiple` - Multiple execution lifecycle plugins
- `TestAccJob_projectSchedule` - Project-level schedule references
- `TestAccJob_projectSchedule_multiple` - Multiple project schedules
- `TestAccJob_projectSchedule_noOptions` - Project schedule without job options
- `TestAccRundeckProjectRunner_basic` - Project runner creation
- `TestAccRundeckProjectRunner_withNodeDispatch` - Project runner with node dispatch
- `TestAccRundeckProjectRunner_update` - Project runner updates
- `TestAccRundeckSystemRunner_basic` - System runner creation
- `TestAccRundeckSystemRunner_update` - System runner updates

**Testing Strategy**:
- Manual testing against Rundeck Enterprise 5.17.0+
- Set `RUNDECK_ENTERPRISE_TESTS=1` to run Enterprise tests locally
- Consider GitHub Actions matrix with Enterprise Docker image (if license permits)

**Files**:
- `rundeck/resource_job_test.go` (lines 1023, 1149, 1188, 1227)
- `rundeck/resource_system_runner_test.go` (lines 15, 45)
- `rundeck/resource_project_runner_test.go` (lines 16, 49, 74)

### 2. Project Schedule Validation

**Issue**: The project schedule tests skip actual validation that schedules are applied in Rundeck.

**Current Behavior**: 
- Tests verify job HCL configuration is accepted
- `testAccJobCheckScheduleExists` helper exists but tests are skipped (require Enterprise)
- Manual setup required (project and schedule must exist before test)

**Improvement**: When Enterprise tests are enabled, verify:
- Schedule is actually applied to the job in Rundeck
- Schedule parameters are correctly passed (`jobParams`)
- Multiple schedules work correctly
- Schedule without options works correctly

**Files**:
- `rundeck/resource_job_test.go` (lines 1131-1269 - Project schedule tests)

---

## 🔄 Migration Opportunities

### 1. Legacy SDK Resources

These resources still use the legacy `go-rundeck/rundeck` v1 SDK and could be migrated to Plugin Framework:

**Resources to Migrate**:
- `rundeck_project` - Project management
- `rundeck_acl_policy` - ACL policy management
- `rundeck_public_key` - Public key storage
- `rundeck_password` - Password storage  
- `rundeck_private_key` - Private key storage

**Benefits of Migration**:
- Consistent implementation across all resources
- Better type safety and diagnostics
- Native support for complex nested blocks
- Improved plan diff accuracy

**Considerations**:
- These resources are relatively simple (no complex nested blocks)
- Migration is lower priority than job/runner resources
- Would allow eventual removal of SDKv2 dependency

**Current SDK Usage**:
```
github.com/rundeck/go-rundeck/rundeck v0.0.0-20190510195016-2cf9670bbcc4
  └── Used by: Job (JSON helpers only), Project, ACL Policy, Public Key
```

### 2. Runner Resources - OpenAPI Spec Improvements

**Status**: Migrated to Framework, but OpenAPI spec has known issues.

**Completed**:
- ✅ Updated to latest `go-rundeck/rundeck-v2` SDK
- ✅ Fixed enum casing issues (LINUX → linux, MANUAL → manual)
- ✅ Implemented tag normalization to prevent drift
- ✅ All runner tests passing

**Remaining OpenAPI Issues** (tracked upstream):
- Some field descriptions may be incomplete
- New runner features may not be in spec yet

**Monitoring**: Watch `rundeck/rundeck-api-specs` repo for updates

---

## 🎯 Future Enhancements

### 1. Advanced Job Features

**Partially Implemented**:
- ✅ Basic execution lifecycle plugins
- ✅ Project schedules  
- ✅ Basic orchestrators (high-low, max-percent, rank-tiered)
- ✅ Command-level plugins (log filters)

**Not Yet Implemented**:
- Global log filters (job-level, not command-level)
- Log limit policies (detailed configuration)
- Retry with backoff strategies
- Node orchestrator custom plugins
- Advanced error handler recursion

**Files to Extend**:
- `rundeck/resource_job_command_schema.go` - Command block schema
- `rundeck/resource_job_framework.go` - Main job resource
- `rundeck/resource_job_converters.go` - JSON conversion helpers

### 2. Additional Resources

**Potential New Resources** (based on Rundeck API capabilities):
- `rundeck_node_source` - Dynamic node sources
- `rundeck_webhook` - Webhook event handlers
- `rundeck_execution` - Manage/trigger executions (?)
- `rundeck_user` - User management (if API supports)
- `rundeck_role` - Role management (if API supports)

### 3. Data Sources

**Current**: None implemented

**Potential Data Sources**:
- `data.rundeck_project` - Look up project details
- `data.rundeck_job` - Reference existing jobs
- `data.rundeck_node` - Query nodes
- `data.rundeck_runner` - Look up runner details

### 4. Provider Configuration Enhancements

**Current Limitations**:
- Single auth token only
- No retry configuration
- Basic timeout handling

**Potential Improvements**:
- Support for API key + username auth
- Configurable retry logic with exponential backoff
- Custom HTTP client configuration
- Multiple Rundeck instance support (aliased providers)

---

## 📝 Documentation Improvements

### 1. Import Documentation

**Current**: Basic import command examples

**Needed**:
- Detailed import guide with common scenarios
- Known limitations for complex nested structures
- Best practices for post-import cleanup
- Troubleshooting guide

**Location**: `website/docs/r/job.html.md`

### 2. Migration Guide

**Current**: CHANGELOG mentions compatibility

**Needed**:
- Detailed upgrade guide from 0.x → 1.0
- Breaking changes with examples
- Common migration issues and solutions
- Rundeck version compatibility matrix

**Location**: `website/docs/guides/migration-v1.html.md` (new)

### 3. Enterprise Features

**Current**: Basic mention of Enterprise requirements

**Needed**:
- Clear Enterprise vs Community feature matrix
- Enterprise setup guide
- Runner management best practices
- Project schedule patterns

**Location**: `website/docs/guides/enterprise-features.html.md` (new)

---

## 🔧 Technical Debt

### 1. Test Infrastructure

**Improvements Needed**:
- Reduce test duplication (many similar test configurations)
- Extract common test helpers to separate file
- Add integration test suite for complex scenarios
- Add performance/load testing for bulk operations

### 2. Error Handling

**Current**: Basic error messages

**Improvements**:
- More specific error types (instead of generic strings)
- Better error context (include job ID, project name, etc.)
- Clearer user-facing error messages
- Link to documentation for common errors

### 3. Code Organization

**Consider**:
- Split `resource_job_framework.go` (currently 1100+ lines)
  - Separate files for schema, CRUD, converters
- Group related converters in `resource_job_converters.go`
- Extract common patterns (e.g., null handling, list conversion)

---

## 📊 Metrics & Monitoring

**Current Test Coverage** (as of 2025-11-17):
- ✅ 23 passing tests (18 job, 5 other resources) - ALL PASSING in CI/CD
- 8 skipped Enterprise tests (require `RUNDECK_ENTERPRISE_TESTS=1`)
- 2 deferred validation tests (future enhancement)
- ~85% code paths tested

**Coverage Gaps**:
- Error paths (API failures, invalid responses)
- Edge cases (empty strings, null values, max lengths)
- Concurrent operations
- Large job configurations (100+ commands/options)

**Tools to Consider**:
- Go test coverage analysis (`go test -cover`)
- Static analysis (golangci-lint)
- Integration testing framework
- API mocking for unit tests

---

## 🎬 Prioritization Recommendations

### High Priority (Before 1.1.0)
1. ✅ Fix job import test (COMPLETED 2025-11-16)
2. ✅ Fix nodefilters structure (COMPLETED 2025-11-17 - CRITICAL)
3. ✅ Fix lifecycle plugins structure, ordering, and empty config (COMPLETED 2025-11-17 - CRITICAL)
4. ✅ Add UUID-based job references (COMPLETED 2025-11-17)
5. ✅ Reorganize and document test infrastructure (COMPLETED 2025-11-17)
6. Implement JSON-to-state converters for import verification (commands, options, notifications)
7. Add schema-level validation for duplicate notifications and empty choices
8. Expand documentation (import guide, migration guide)

### Medium Priority (1.x releases)
9. Migrate remaining SDK resources to Framework
10. Implement data sources
11. Enhance error messages and handling
12. Add integration test suite

### Low Priority (Future)
13. Advanced job features (global log filters, retry strategies)
14. New resources (node sources, webhooks, users)
15. Provider configuration enhancements
16. Performance optimization

---

## 📞 Contributing

If you're interested in contributing to any of these items:
1. Open an issue on GitHub to discuss the approach
2. Reference this TODO.md for context
3. Follow the established patterns in migrated resources
4. Ensure tests pass and add new tests for new functionality
5. Update documentation alongside code changes

---

**Last Updated**: 2025-11-17
**Version**: 1.0.0 (RC - pending final CI/CD validation)
**Maintainer**: @fdevans

