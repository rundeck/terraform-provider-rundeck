# Terraform Provider Rundeck - TODO

Prioritized list of remaining work for the Rundeck Terraform Provider.

**Current Status**: v1.0.0 ready for release  
**Last Updated**: 2025-11-18

**Recent Accomplishments** (v1.0.0):
- ✅ **SDK Updated**: `rundeck-v2` updated to fix `ErrorResponse` bug
- ✅ **Semantic Equality**: Runner tags now use semantic equality (no more plan drift)
- ✅ **Issues Fixed**: #156 (EOF), #126 (delimiter), #198 (password state) all tested & verified
- ✅ **Full JSON**: All XML code eliminated, JSON-only API interactions
- ✅ **Runner Resources**: System and project runners fully migrated to Framework

---

## 🔴 High Priority (Before 1.1.0)

### 1. Complete Job Import for Complex Structures
**Effort**: Medium (2-3 days)  
**Why Important**: Users importing existing jobs will see drift on first `terraform plan` for complex configurations (commands, options, notifications). This creates confusion and reduces adoption.

**What's Missing**:
- `convertCommandsFromJSON()` - Parse commands array from API to Terraform state
- `convertOptionsFromJSON()` - Parse options array from API to Terraform state  
- `convertNotificationsFromJSON()` - Parse notifications map to Terraform state

**Files**: `rundeck/resource_job_converters.go`, `rundeck/resource_job_framework.go`

**Current Workaround**: Import works, but complex nested blocks added to `ImportStateVerifyIgnore`

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
**Why Important**: Reduces support burden and improves onboarding for new users upgrading from 0.x to 1.0.

**Priority Pages**:
1. **Import Guide** (`website/docs/guides/import.html.md`) - NEW
   - Step-by-step import workflow
   - Known limitations for complex structures
   - Post-import cleanup best practices

2. **Migration Guide** (`website/docs/guides/migration-v1.html.md`) - NEW
   - 0.x → 1.0 upgrade path
   - Breaking changes with examples
   - Rundeck version compatibility matrix

3. **Enterprise Features** (`website/docs/guides/enterprise.html.md`) - NEW
   - Feature comparison table (OSS vs Enterprise)
   - Runner management patterns
   - Project schedules usage

---

## 🟡 Medium Priority (1.x Releases)

### 4. Migrate Legacy SDK Resources to Framework
**Effort**: Large (1-2 weeks)  
**Why Important**: Removes dependency on legacy SDKv2, unifies implementation patterns, improves type safety.

**Resources**:
- `rundeck_project` - Most complex, has configuration maps
- `rundeck_acl_policy` - Simple, good starting point
- `rundeck_public_key` - Simple
- `rundeck_password` - Simple
- `rundeck_private_key` - Simple

**Approach**: Migrate simple resources first (ACL, keys), then project last.

**Benefits**:
- Consistent error handling across all resources
- Better plan diff accuracy
- Eventually remove SDKv2 dependency

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

### 7. Enterprise Test Automation & Enhancement
**Effort**: Medium (3-5 days)  
**Why Important**: 9 tests currently skip in CI/CD, manual testing is slow and error-prone.

**Skipped Tests**:
- 4 project schedule tests
- 5 runner tests (system + project)

**Comprehensive Test Improvements**:
- Add automated cleanup command (destroy + remove artifacts)
- Integrate drift validation into test flow
- Make build process more robust
- Add summary report generation
- Document for community contributors
- **File**: `test/enterprise/comprehensive.sh`

**CI/CD Options**:
1. GitHub Actions with Enterprise Docker image (requires license)
2. Scheduled manual test runs against Enterprise instance
3. Community-contributed test results

**Goal**: Increase automated test coverage from ~85% to 95% + easier community testing

---

## 🟢 Low Priority (Future)

### 8. Advanced Job Features
**Effort**: Large (2-3 weeks)  
**Why Important**: Completeness, but rarely used features.

**Features**:
- Global log filters (job-level, not command-level)
- Detailed log limit policies
- Retry with backoff strategies
- Custom orchestrator plugins
- Advanced error handler recursion

**Approach**: Implement on-demand based on user requests

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
- ✅ 23/23 OSS tests passing (100%)
- ⏭️ 9 Enterprise tests skipped (require manual setup)
- ⏭️ 2 validation tests skipped (future enhancement)
- ~85% code path coverage

**Known Limitations**:
- Import works but shows drift for complex nested structures
- No schema validation for duplicate notifications or empty choices
- Enterprise features require manual testing

**GitHub Issues Status**:
- ✅ [#156](https://github.com/rundeck/terraform-provider-rundeck/issues/156) - EOF error - **FIXED in v1.0.0** (tested & confirmed)
- ✅ [#126](https://github.com/rundeck/terraform-provider-rundeck/issues/126) - multi_value_delimiter - **FIXED in v1.0.0** (tested & confirmed)
- ✅ [#198](https://github.com/rundeck/terraform-provider-rundeck/issues/198) - password state corruption - **FIXED in v1.0.0** (tested & confirmed)
- 🟡 [#70](https://github.com/rundeck/terraform-provider-rundeck/issues/70) - extra_config merge - **Design decision** (Medium #6)
- 🟢 [#76](https://github.com/rundeck/terraform-provider-rundeck/issues/76) - SCM support - **Feature request** (Low #9)

---

## 🎯 Recommended Next Steps

### For 1.0.0 Release (Immediate)
- ✅ All critical features complete
- ✅ All OSS tests passing
- ✅ Documentation updated
- ✅ GitHub issues #156, #126, and #198 tested and confirmed fixed
- 🟡 Post test results to GitHub issues

### For 1.1.0 (Next Month)
1. Complete job import converters (High #1)
2. Add schema validation (High #2)
3. Write import/migration guides (High #3)

### For 2.0.0 (Future)
1. Migrate all resources to Framework (Medium #4) - Will address issue #70
2. Remove SDKv2 dependency
3. Implement data sources (Medium #5)
4. Consider SCM support if demand increases (Low #9) - GitHub issue #76

---

## 📞 Contributing

Want to tackle one of these? 

1. **Small tasks** (< 1 day): High #2 (schema validation), individual docs
2. **Medium tasks** (1-5 days): High #1 (import), High #3 (docs), Medium #6 (extra_config), Medium #7 (errors)
3. **Large tasks** (1-3 weeks): Medium #4 (Framework migrations), Medium #5 (data sources), Low #8 (advanced features), Low #9 (SCM)

Open an issue on GitHub to discuss your approach before starting.

**GitHub Issues**: Check [open issues](https://github.com/rundeck/terraform-provider-rundeck/issues) for community-reported bugs and feature requests.

---

**Last Updated**: 2025-11-17  
**Maintainer**: @fdevans  
**Repository**: https://github.com/terraform-providers/terraform-provider-rundeck
