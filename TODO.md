# Terraform Provider Rundeck - TODO

Prioritized list of remaining work for the Rundeck Terraform Provider.

**Current Status**: v1.0.0 ready for release  
**Last Updated**: 2025-11-20

**Recent Accomplishments** (v1.0.0):
- ✅ **SDKv2 Removed**: Single Plugin Framework implementation, mux eliminated
- ✅ **SDK Updated**: Both SDKs now at v1.2.0 with proper module versioning
- ✅ **Semantic Equality**: Runner tags use semantic equality (no more plan drift)
- ✅ **Issues Fixed**: #156 (EOF), #126 (delimiter), #198 (password state) all tested & verified
- ✅ **Full JSON**: All XML code eliminated, JSON-only API interactions
- ✅ **All 8 Resources Migrated**: Job, runners (system/project), project, ACL, keys (public/private), password - 100% on Framework
- ✅ **Complete Notification Support**: Email, webhook, plugin with proper API format handling
- ✅ **Schedule Normalization**: Automatic cron normalization prevents drift
- ✅ **Option Enforcement**: Correct inference from value_choices
- ✅ **FROM JSON Converters**: Schedule, orchestrator, log limit, global log filter, notifications, options all implemented
- ✅ **Orchestrator Support**: All types (maxPercentage, subset, rankTiered, custom plugins)
- ✅ **Log Limit Support**: Full implementation (output, action, status)
- ✅ **Global Log Filter Support**: Job-level log filtering with plugin config
- ✅ **Integration Tests**: API validation tests verify actual Rundeck storage (2 new tests)
- ✅ **Username/Password Auth**: Framework provider now supports auth_username/auth_password
- ✅ **Zero Plan Drift**: All resources stable across all tested scenarios
- ✅ **Comprehensive Testing**: 26/26 OSS tests passing, enterprise test validated (10 resources)

---

## 🔴 High Priority (Before 1.1.0)

### 1. Complete Job Import - Command Parsing
**Effort**: Small-Medium (1-2 days)  
**Status**: 90% Complete  
**Why Important**: Commands are the only remaining structure not parsed during import.

**What's Complete** ✅:
- ✅ `convertScheduleObjectToCron()` - Parse schedule from API
- ✅ `convertOrchestratorFromJSON()` - Parse orchestrator config
- ✅ `convertLogLimitFromJSON()` - Parse log limit config
- ✅ `convertOptionsFromJSON()` - Parse options array from API to Terraform state  
- ✅ `convertNotificationsFromJSON()` - Parse notifications map to Terraform state

**What's Missing**:
- `convertCommandsFromJSON()` - Parse commands array from API to Terraform state
  - Complex due to variety: shell, script, job reference (name/uuid), plugin, etc.
  - Currently has TODO at line 1259 in resource_job_framework.go

**Files**: `rundeck/resource_job_converters.go`, `rundeck/resource_job_framework.go`

**Impact**: Low - Commands don't drift, import still works (you just manually add commands to config)

---

### 2. Schema-Level Validation
**Effort**: Small (4-6 hours)  
**Why Important**: Prevents user errors at plan time instead of apply time, improving UX and reducing failed API calls.

#### A. Duplicate Notification Validation
- Prevent multiple notifications of the same type (e.g., two `on_success` blocks)
- Use `terraform-plugin-framework-validators`
- **Test**: `TestAccJobNotification_multiple` (currently skipped)

#### B. Empty Choice Validation
- Require at least one `value_choices` when `require_predefined_choice = true`
- Custom conditional validator
- **Test**: `TestAccJobOptions_empty_choice` (currently skipped)

**Files**: `rundeck/resource_job_option_schema.go`, `rundeck/resource_job_notification_schema.go`

---

### 3. Documentation Improvements
**Effort**: Medium (1-2 days)  
**Status**: Partially Complete (core docs done, guides deferred)  
**Why Important**: Reduces support burden and improves onboarding for new users upgrading from 0.x to 1.0.

**Completed in v1.0.0** ✅:
- ✅ **CHANGELOG.md** - Comprehensive v1.0.0 entry with breaking changes, enhancements, bug fixes
- ✅ **Job Resource Docs** (`website/docs/r/job.html.md`) - Notification ordering, webhook format notes
- ✅ **Runner Resource Docs** - Tag normalization behavior documented
- ✅ **Issue Template** (`.github/ISSUE_TEMPLATE.md`) - Enhanced with test script guidance, version requirements
- ✅ **Test Documentation** (`test/enterprise/README.md`, `test/oss/README.md`) - Portable examples, troubleshooting
- ✅ **PR Description** - Complete migration guide with before/after examples

**Deferred to v1.1.0** (nice-to-have guide pages):
1. **Import Guide** (`website/docs/guides/import.html.md`) - NEW
   - Step-by-step import workflow
   - Known limitations for complex structures
   - Post-import cleanup best practices

2. **Migration Guide** (`website/docs/guides/migration-v1.html.md`) - NEW
   - 0.x → 1.0 upgrade path (covered in CHANGELOG/PR description for now)
   - Breaking changes with examples
   - Rundeck version compatibility matrix

3. **Enterprise Features** (`website/docs/guides/enterprise.html.md`) - NEW
   - Feature comparison table (OSS vs Enterprise)
   - Runner management patterns
   - Project schedules usage

---

## 🟡 Medium Priority (1.x Releases)

### 4. ~~Migrate SDK Resources to Framework~~ ✅ COMPLETE
**Status**: ✅ **100% Complete - All 8 resources migrated + dead code cleaned up**  
**Completed**: v1.0.0

**All Resources Migrated to Framework** ✅:
- ✅ `rundeck_job` - Complete with all features
- ✅ `rundeck_system_runner` - With semantic equality for tags
- ✅ `rundeck_project_runner` - With semantic equality for tags
- ✅ `rundeck_project` - Full migration complete
- ✅ `rundeck_acl_policy` - Full migration complete
- ✅ `rundeck_public_key` - Full migration complete
- ✅ `rundeck_password` - Full migration complete
- ✅ `rundeck_private_key` - Full migration complete

**Benefits Achieved**:
- ✅ Consistent error handling across all resources
- ✅ Better plan diff accuracy
- ✅ SDKv2 dependency completely removed
- ✅ Unified testing patterns
- ✅ Single provider implementation (no mux)
- ✅ Dead SDKv2 code deleted (6 files removed)

---

### 5. Implement Data Sources
**Effort**: Medium (1 week)  
**Why Important**: Enables referencing existing Rundeck resources without managing them, common pattern in Terraform.

**Priority Order**:
1. `data.rundeck_project` - Look up project details, most requested
2. `data.rundeck_job` - Reference existing jobs by name/UUID
3. `data.rundeck_runner` - Look up runner details (Enterprise)
4. `data.rundeck_node` - Query nodes (lower priority)

**Use Cases**:
- Reference existing projects created outside Terraform
- Build job dependencies without hardcoding UUIDs
- Dynamic runner assignment

---

### 6. Project extra_config Merge Behavior
**Effort**: Small-Medium (1-2 days)  
**Why Important**: Users want additive configuration changes, not full replacements.  
**GitHub Issue**: [#70](https://github.com/rundeck/terraform-provider-rundeck/issues/70)

**Problem**:
Terraform's default behavior replaces the entire `extra_config` map. Users want to add new keys without removing existing ones that may have been set outside Terraform.

**Current Behavior**:
```hcl
# State has: {existing_key = "keep", another = "preserve"}
extra_config = {
  new_key = "add_this"
}
# Result: Only {new_key = "add_this"} remains
```

**Desired Behavior**:
```hcl
# Merge new keys with existing
# Result: {existing_key = "keep", another = "preserve", new_key = "add_this"}
```

**Challenges**:
- This is **Terraform's design** - resources are declarative, not additive
- Would break standard Terraform behavior expectations
- Requires special state management logic

**Options**:
1. **Accept as Terraform design** (Recommended)
   - Document this behavior clearly
   - Show workaround using `terraform import` + merge in config
   
2. **Custom merge logic**
   - Read existing config from API before apply
   - Merge with plan config
   - Could surprise users expecting standard Terraform behavior
   
3. **Separate resource for individual keys**
   - `rundeck_project_config_item` resource
   - Manages single key-value pairs
   - More Terraform-idiomatic

**Recommendation**: Document current behavior, consider `rundeck_project_config_item` for v2.0.0 if demand is high.

**Related**: Will be reconsidered during project resource Framework migration (Medium #4)

---

### 7. Enhanced Error Handling
**Effort**: Medium (3-5 days)  
**Why Important**: Poor error messages waste user time and create support tickets.

**Improvements**:
- Structured error types (not generic strings)
- Include context (job ID, project name, API version)
- Link to docs for common errors
- Better API error parsing

**Example**:
```
Before: Error creating job: 400 Bad Request
After:  Error creating job "my-job" in project "prod": Rundeck returned validation error - 
        Job name already exists in this project. See: https://docs.../job-naming
```

---

### 7. Enterprise Test Automation & CI/CD
**Effort**: Small-Medium (2-3 days)  
**Status**: Mostly Complete  
**Why Important**: 9 tests currently skip in CI/CD (require Enterprise Rundeck).

**Completed** ✅:
- ✅ `test/enterprise/comprehensive.sh` - Full enterprise validation
  - Automated cleanup (destroy + remove artifacts)
  - Drift validation integrated
  - Portable (no hardcoded paths)
  - Summary report generation
- ✅ `test/enterprise/test-custom.sh` - "Bring your own plan" testing
  - Automatic provider build
  - Drift detection
  - Easy community testing
- ✅ Conditional project schedule tests (skip gracefully if not set up)
- ✅ Documentation in `test/enterprise/README.md`

**Remaining**:
- Set up CI/CD with Enterprise Rundeck instance
  - Option 1: GitHub Actions with Enterprise Docker (requires license)
  - Option 2: Scheduled runs on internal Enterprise instance
  - Option 3: Manual verification before releases

**Skipped in CI/CD** (9 tests):
- 1 execution lifecycle plugin test (multiple plugins)
- 3 project schedule tests (API not available yet)
- 3 project runner tests (Enterprise feature)
- 2 system runner tests (Enterprise feature)

---

## 🟢 Low Priority (Future)

### 8. Advanced Job Features
**Effort**: Large (2-3 weeks)  
**Status**: 60% Complete (3 of 5 done in v1.0.0)  
**Why Important**: Completeness, but rarely used features.

**Completed in v1.0.0** ✅:
- ✅ **Global log filters** (job-level) - Full implementation with TO/FROM JSON converters
  - `convertGlobalLogFilterToJSON()` implemented
  - `convertGlobalLogFilterFromJSON()` implemented
  - Supports plugin type and config
- ✅ **Detailed log limit policies** - Complete implementation
  - `convertLogLimitToJSON()` implemented
  - `convertLogLimitFromJSON()` implemented
  - Supports output, action, and status
- ✅ **Orchestrator plugins** - All types supported
  - `convertOrchestratorToJSON()` implemented
  - `convertOrchestratorFromJSON()` implemented
  - Supports maxPercentage, subset, rankTiered, and custom plugins

**Remaining**:
- Retry with backoff strategies (partially done - basic retry exists, backoff strategies not implemented)
- Advanced error handler recursion

**Approach**: Remaining features on-demand based on user requests

---

### 9. SCM Integration Support
**Effort**: Large (1-2 weeks)  
**Why Important**: Users want to manage SCM configurations via Terraform.  
**GitHub Issue**: [#76](https://github.com/rundeck/terraform-provider-rundeck/issues/76)

**Feature Request**:
Add support for Rundeck's Source Control Management (SCM) integration, allowing Terraform to configure Git import/export for projects.

**Scope**:
Rundeck supports two SCM integrations per project:
- **SCM Import**: Pull job definitions from Git into Rundeck
- **SCM Export**: Push job definitions from Rundeck to Git

**Proposed Resources**:
1. `rundeck_scm_import` - Configure Git import for a project
2. `rundeck_scm_export` - Configure Git export for a project

**Configuration Examples**:
```hcl
resource "rundeck_scm_import" "project_import" {
  project     = rundeck_project.main.name
  plugin_type = "git-import"
  
  config = {
    dir             = "/var/rundeck/projects/${rundeck_project.main.name}/scm"
    url             = "https://github.com/org/rundeck-jobs.git"
    branch          = "main"
    strictHostKeyChecking = "yes"
    sshPrivateKeyPath     = "/var/rundeck/.ssh/id_rsa"
  }
}
```

**API Support**:
- Rundeck API v14+ supports SCM endpoints
- `POST /api/14/project/{project}/scm/{integration}/plugin/{type}/setup`
- `GET /api/14/project/{project}/scm/{integration}/config`

**Complexity**:
- Multiple plugin types (git-import, git-export)
- Complex configuration schema varies by plugin
- Authentication methods (SSH keys, tokens)
- State management for SCM actions (import/export/sync)

**Priority Justification**:
- Low demand (only 1 GitHub issue in 3 years)
- Complex implementation
- Workaround exists (manual SCM setup in Rundeck UI)
- Not blocking any workflows

**Recommendation**: Consider for v1.x if user demand increases

---

### 10. New Resources (Other)
**Effort**: Large (varies by resource)  
**Why Important**: Expands provider capabilities, but low user demand currently.

**Candidates**:
- `rundeck_node_source` - Dynamic node sources (Medium priority)
- `rundeck_webhook` - Webhook event handlers (Low priority)
- `rundeck_user` / `rundeck_role` - User management (if API supports)
- `rundeck_execution` - Trigger/manage executions (questionable use case)

**Approach**: Validate demand before implementation

---

### 11. Provider Configuration Enhancements
**Effort**: Medium (1 week)  
**Why Important**: Nice-to-have features, not blocking users.

**Features**:
- Configurable retry logic with exponential backoff
- Custom HTTP client configuration (timeout, TLS settings)
- Multiple Rundeck instance support (aliased providers)
- Support for API key + username auth

---

### 12. Code Organization Refactor
**Effort**: Medium (3-5 days)  
**Why Important**: Maintainability, but no user impact.

**Tasks**:
- Split `resource_job_framework.go` (1100+ lines) into multiple files
- Group related converters in `resource_job_converters.go`
- Extract common patterns (null handling, list conversion)
- Add integration test suite

---

## 📊 Current Metrics

**Test Coverage**:
- ✅ 26/26 OSS tests passing (100%)
  - Includes 2 new integration tests with API validation
  - TestAccJob_ComplexIntegration - validates orchestrator, log limit, schedule, options, notifications via API
  - TestAccJob_NotificationIntegration - validates email and webhook structures via API
- ⏭️ 9 Enterprise tests skipped (require Enterprise Rundeck instance)
- ⏭️ 2 validation tests skipped (future enhancement: duplicate notifications, empty choices)
- ✅ Comprehensive enterprise test validated (10 resources, zero drift)
- ~90% code path coverage

**Known Limitations**:
- Command import not yet implemented (commands don't drift, low priority)
- No schema validation for duplicate notifications or empty choices (UX enhancement)
- Project schedules require manual creation (no API available yet)

**GitHub Issues Status**:
- ✅ [#156](https://github.com/rundeck/terraform-provider-rundeck/issues/156) - EOF error - **FIXED in v1.0.0** (tested & confirmed)
- ✅ [#126](https://github.com/rundeck/terraform-provider-rundeck/issues/126) - multi_value_delimiter - **FIXED in v1.0.0** (tested & confirmed)
- ✅ [#198](https://github.com/rundeck/terraform-provider-rundeck/issues/198) - password state corruption - **FIXED in v1.0.0** (tested & confirmed)
- 🟡 [#70](https://github.com/rundeck/terraform-provider-rundeck/issues/70) - extra_config merge - **Design decision** (Medium #6)
- 🟢 [#76](https://github.com/rundeck/terraform-provider-rundeck/issues/76) - SCM support - **Feature request** (Low #9)

---

## 🎯 Recommended Next Steps

### For 1.0.0 Release (✅ READY)
- ✅ All critical features complete
- ✅ All 26 OSS tests passing (100%)
- ✅ Comprehensive enterprise test validated
- ✅ Integration tests with API validation
- ✅ Documentation updated (CHANGELOG, job docs, PR description)
- ✅ GitHub issues #156, #126, and #198 tested and confirmed fixed
- ✅ Zero plan drift for all resources
- 🟡 Post test results to GitHub issues after release

### For 1.1.0 (Next 1-2 Months)
1. Complete command import parser (High #1) - ~1-2 days
2. Add schema validation (High #2) - ~1 day
3. Write dedicated guide pages (High #3) - ~2 days
   - Import guide (import.html.md)
   - Migration guide (migration-v1.html.md)
   - Enterprise features guide (enterprise.html.md)
4. Project schedule resource (if API becomes available)

### For 1.2.0+ (Next 3-6 Months)
1. Migrate remaining SDKv2 resources to Framework (Medium #4) - ~1-2 weeks
2. Implement data sources (Medium #5) - ~1 week
3. Enhanced error handling (Medium #7) - ~3-5 days

### For 2.0.0 (Future - 6-12 Months)
1. ~~Complete framework migration~~ ✅ Done in v1.0.0
2. ~~Remove SDKv2 dependency~~ ✅ Done in v1.0.0
3. Clean up dead SDKv2 code files
4. Advanced job features (Low #8)
5. Consider SCM support if demand increases (Low #9) - GitHub issue #76

---

## 📞 Contributing

Want to tackle one of these? 

1. **Small tasks** (< 1 day): High #2 (schema validation), individual docs
2. **Medium tasks** (1-5 days): High #1 (import), High #3 (docs), Medium #6 (extra_config), Medium #7 (errors)
3. **Large tasks** (1-3 weeks): Medium #4 (Framework migrations), Medium #5 (data sources), Low #8 (advanced features), Low #9 (SCM)

Open an issue on GitHub to discuss your approach before starting.

**GitHub Issues**: Check [open issues](https://github.com/rundeck/terraform-provider-rundeck/issues) for community-reported bugs and feature requests.

---

**Last Updated**: 2025-11-20  
**Maintainer**: Rundeck Team  
**Repository**: https://github.com/rundeck/terraform-provider-rundeck
