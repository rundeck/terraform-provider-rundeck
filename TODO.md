# Terraform Provider Rundeck - TODO

Forward-looking tasks for the Rundeck Terraform Provider.

**Current Status**: v1.0.0 ready for release  
**Last Updated**: 2025-11-21

---

## 🔴 High Priority

### Schema-Level Validation
**Effort**: Small (4-6 hours)  
**Why Important**: Prevents user errors at plan time instead of apply time, improving UX and reducing failed API calls.

#### Duplicate Notification Validation
- Prevent multiple notifications of the same type (e.g., two `on_success` blocks)
- Use `terraform-plugin-framework-validators`
- **Test**: `TestAccJobNotification_multiple` (currently skipped)

#### Empty Choice Validation
- Require at least one `value_choices` when `require_predefined_choice = true`
- Custom conditional validator
- **Test**: `TestAccJobOptions_empty_choice` (currently skipped)

**Files**: `rundeck/resource_job_option_schema.go`, `rundeck/resource_job_notification_schema.go`

---

### Documentation Improvements
**Effort**: Medium (1-2 days)  
**Status**: Core docs complete, guides deferred  
**Why Important**: Reduces support burden and improves onboarding for new users upgrading from 0.x to 1.0.

**Deferred to v1.1.0** (nice-to-have guide pages):
- **Import Guide** (`website/docs/guides/import.html.md`)
  - Step-by-step import workflow
  - Known limitations for complex structures
  - Post-import cleanup best practices

- **Migration Guide** (`website/docs/guides/migration-v1.html.md`)
  - 0.x → 1.0 upgrade path (covered in CHANGELOG/PR description for now)
  - Breaking changes with examples
  - Rundeck version compatibility matrix

- **Enterprise Features** (`website/docs/guides/enterprise.html.md`)
  - Feature comparison table (OSS vs Enterprise)
  - Runner management patterns
  - Project schedules usage

---

## 🟡 Medium Priority

### Implement Data Sources
**Effort**: Medium (1 week)  
**Why Important**: Enables referencing existing Rundeck resources without managing them, common pattern in Terraform.

**Priority Order**:
- `data.rundeck_project` - Look up project details, most requested
- `data.rundeck_job` - Reference existing jobs by name/UUID
- `data.rundeck_runner` - Look up runner details (Enterprise)
- `data.rundeck_node` - Query nodes (lower priority)

**Use Cases**:
- Reference existing projects created outside Terraform
- Build job dependencies without hardcoding UUIDs
- Dynamic runner assignment

---

### Project extra_config Merge Behavior
**Effort**: Small-Medium (1-2 days)  
**Why Important**: Users want additive configuration changes, not full replacements.  
**GitHub Issue**: [#70](https://github.com/rundeck/terraform-provider-rundeck/issues/70)

**Problem**:
Terraform's default behavior replaces the entire `extra_config` map. Users want to add new keys without removing existing ones that may have been set outside Terraform.

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

---

### Enhanced Error Handling
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

### Enterprise Test Automation & CI/CD
**Effort**: Small-Medium (2-3 days)  
**Status**: Mostly Complete  
**Why Important**: 9 tests currently skip in CI/CD (require Enterprise Rundeck).

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

### Code Organization Refactor
**Effort**: Small (1-2 days remaining)  
**Why Important**: Maintainability, but no user impact.

**Remaining**:
- Split `resource_job_framework.go` (1313 lines) into logical modules:
  - Schema definition
  - CRUD operations
  - Helper functions
- Extract common patterns (null handling, list conversion) into utilities

---

## 🟢 Low Priority

### Advanced Job Features
**Effort**: Medium (1-2 weeks)  
**Why Important**: Completeness, but rarely used features.

**Remaining Tasks**:
- Retry with backoff strategies (basic retry exists, backoff strategies not implemented)
- Advanced error handler recursion

**Approach**: On-demand based on user requests

---

### SCM Integration Support
**Effort**: Large (1-2 weeks)  
**Why Important**: Users want to manage SCM configurations via Terraform.  
**GitHub Issue**: [#76](https://github.com/rundeck/terraform-provider-rundeck/issues/76)

**Feature Request**:
Add support for Rundeck's Source Control Management (SCM) integration, allowing Terraform to configure Git import/export for projects.

**Proposed Resources**:
- `rundeck_scm_import` - Configure Git import for a project
- `rundeck_scm_export` - Configure Git export for a project

**API Support**:
- Rundeck API v14+ supports SCM endpoints
- Complex configuration schema varies by plugin
- Authentication methods (SSH keys, tokens)

**Priority Justification**:
- Low demand (only 1 GitHub issue in 3 years)
- Complex implementation
- Workaround exists (manual SCM setup in Rundeck UI)

**Recommendation**: Consider for v1.x if user demand increases

---

### New Resources (Other)
**Effort**: Large (varies by resource)  
**Why Important**: Expands provider capabilities, but low user demand currently.

**Candidates**:
- `rundeck_node_source` - Dynamic node sources (Medium priority)
- `rundeck_webhook` - Webhook event handlers (Low priority)
- `rundeck_user` / `rundeck_role` - User management (if API supports)
- `rundeck_execution` - Trigger/manage executions (questionable use case)

**Approach**: Validate demand before implementation

---

### Provider Configuration Enhancements
**Effort**: Medium (1 week)  
**Why Important**: Nice-to-have features, not blocking users.

**Features**:
- Configurable retry logic with exponential backoff
- Custom HTTP client configuration (timeout, TLS settings)
- Multiple Rundeck instance support (aliased providers)
- Support for API key + username auth

---

## 📞 Contributing

Want to tackle one of these? 

- **Small tasks** (< 1 day): Schema validation, individual docs
- **Medium tasks** (1-5 days): Docs, extra_config, errors, test automation
- **Large tasks** (1-3 weeks): Data sources, SCM, advanced features

Open an issue on GitHub to discuss your approach before starting.

**GitHub Issues**: Check [open issues](https://github.com/rundeck/terraform-provider-rundeck/issues) for community-reported bugs and feature requests.

---

**Last Updated**: 2025-11-21  
**Maintainer**: Rundeck Team  
**Repository**: https://github.com/rundeck/terraform-provider-rundeck
