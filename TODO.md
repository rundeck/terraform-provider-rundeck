# Terraform Provider Rundeck - TODO

Forward-looking tasks for the Rundeck Terraform Provider.

**Current Status**: 1.4.0 merged to `master` (not yet tagged) - local roles, system execution mode, runner data sources, plus job and system runner fixes. See `CHANGELOG.md` for full details.  
**Last Updated**: 2026-08-27 (1.4.0 merged)

---

## ✅ Completed in 1.4.0

- **`rundeck_local_role`** - Create/read/update/delete for Enterprise local user store roles, including membership management ([#291](https://github.com/rundeck/terraform-provider-rundeck/pull/291)). `rundeck_local_user` remains blocked - see the note under "New Resources (Other)" below.
- **`rundeck_system_execution_mode`** - Controls whether a server executes jobs (`active`/`passive`) ([#287](https://github.com/rundeck/terraform-provider-rundeck/pull/287)).
- **Runner data sources** - `rundeck_runner`, `rundeck_runners`, `rundeck_runner_tags`, closing the long-standing `data.rundeck_runner` item below ([#290](https://github.com/rundeck/terraform-provider-rundeck/pull/290)).
- **`rundeck_system_runner` project detachment fix** - `Update` now explicitly clears per-project dispatch settings when a project is removed, instead of relying on unreliable overwrite semantics ([#290](https://github.com/rundeck/terraform-provider-rundeck/pull/290)).
- **Job resource enhancements** - `node_intersect` on job reference dispatch blocks, `values_list_delimiter` for option choices, `notify_avg_duration_threshold` for `on_avg_duration` notifications ([#284](https://github.com/rundeck/terraform-provider-rundeck/pull/284), [#285](https://github.com/rundeck/terraform-provider-rundeck/pull/285)).
- **SCM resources implemented, held from this release** - `rundeck_scm_import`/`rundeck_scm_export` are built and reviewed ([#292](https://github.com/rundeck/terraform-provider-rundeck/pull/292)) but not merged: acceptance testing surfaced a GitHub/SSHJ SSH handshake incompatibility that's on Rundeck's server side, not the provider. Held pending a Rundeck 6.2.0 fix; see the "SCM Integration Support" entry below for details.

---

## 🔴 High Priority

---

### Documentation Improvements
**Effort**: Small (1 day)  
**Status**: Import Guide complete! Enterprise guide still deferred  
**Why Important**: Reduces support burden and improves onboarding for new users.

**Completed in v1.2.0:**
- ✅ **Import Guide** (`website/docs/guides/import.html.md`)
  - Comprehensive step-by-step import workflow for all resources
  - Resource-specific examples (projects, jobs, webhooks, ACLs, keys, runners)
  - Bulk import scripts and common patterns
  - Troubleshooting section with solutions

**Nice-to-have guide page:**
- **Enterprise Features** (`website/docs/guides/enterprise.html.md`)
  - Feature comparison table (OSS vs Enterprise)
  - Runner management patterns
  - Project schedules usage

---

## 🟡 Medium Priority

### Implement Data Sources
**Effort**: Medium (1 week)  
**Why Important**: Enables referencing existing Rundeck resources without managing them, common pattern in Terraform.

**Completed in v1.4.0**:
- ✅ `data.rundeck_runner`, `data.rundeck_runners`, `data.rundeck_runner_tags` - Look up and enumerate Enterprise runners

**Remaining, priority order**:
- `data.rundeck_project` - Look up project details, most requested
- `data.rundeck_job` - Reference existing jobs by name/UUID
- `data.rundeck_node` - Query nodes (lower priority)

**Use Cases**:
- Reference existing projects created outside Terraform
- Build job dependencies without hardcoding UUIDs

---

### Project-Level ACL Policy Resource
**Effort**: Medium (1-2 days)  
**Why Important**: Customers are migrating from system ACLs to project-level ACLs for better isolation and want to manage them via Terraform.  
**Customer Request**: Received via email - customer managing system ACLs successfully, wants same workflow for project ACLs.

**Current Limitation**:
- `rundeck_acl_policy` only supports **system-level ACLs** (uses `/api/VERSION/system/acl/` endpoints)
- Project-level ACLs use different API endpoints: `/api/VERSION/project/[PROJECT]/acl/`
- Workaround exists (system ACL with project context) but stores in different location

**Proposed Resource**: `rundeck_project_acl_policy`

**Requirements**:
1. Verify `go-rundeck` SDK v1.2.0+ has `ProjectACLPolicy*` methods (Create, Get, Update, Delete)
2. Create new resource with schema:
   - `project` (Required) - Project name for ACL scope
   - `name` (Required) - ACL policy file name
   - `policy` (Required) - YAML formatted ACL policy string
3. Implement CRUD operations using project ACL API endpoints
4. Add acceptance tests (create, update, import, delete scenarios)
5. Documentation with examples showing project-level vs system-level ACLs
6. Update provider resources list

**Example Usage**:
```hcl
resource "rundeck_project_acl_policy" "project_admin" {
  project = "my-project"
  name    = "ProjectAdmins.aclpolicy"
  policy  = file("${path.module}/project-admins.aclpolicy")
}

# Batch management like current system ACLs
resource "rundeck_project_acl_policy" "policies" {
  for_each = fileset(path.module, "project-acls/*.aclpolicy")
  project  = "my-project"
  name     = each.value
  policy   = file("${path.module}/project-acls/${each.value}")
}
```

**Benefits**:
- True project-level ACL management (stored in project scope, not system)
- Maintains Git-based workflow customers already use for system ACLs
- Consistent with Rundeck best practices (project ACLs for project-specific permissions)
- Enables per-project ACL versioning and deployment

**Investigation Needed**:
- Check if `go-rundeck` SDK has the necessary API methods, or if they need to be added
- Verify API version requirements (likely v14+)

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

**Recommendation**: Document current behavior, consider `rundeck_project_config_item` for future release if demand is high.

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

**Skipped in CI/CD** (15 tests):
- 1 execution lifecycle plugin test (multiple plugins)
- 3 project schedule tests (API not available yet)
- 3 project runner tests (Enterprise feature)
- 2 system runner tests (Enterprise feature)
- 6 webhook Enterprise plugin tests (advanced-run-job, datadog-run-job, pagerduty-run-job, pagerduty-V3-run-job, github-webhook, aws-sns-webhook)

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
**Effort**: Large (1-2 weeks) - **implemented, held from release**  
**Why Important**: Users want to manage SCM configurations via Terraform.  
**GitHub Issue**: [#76](https://github.com/rundeck/terraform-provider-rundeck/issues/76)  
**PR**: [#292](https://github.com/rundeck/terraform-provider-rundeck/pull/292)

**Status**: `rundeck_scm_import` and `rundeck_scm_export` are implemented, reviewed, and passed acceptance testing for the resource logic itself. They're held out of 1.4.0 because that same acceptance testing (a Git-export config against a real GitHub remote) hit a GitHub/SSHJ SSH handshake incompatibility - confirmed via server logs to be on Rundeck's side (the SSHJ client disconnects before authentication completes), not a provider or SDK bug. A fix is going into Rundeck 6.2.0. Once that ships, re-run the SCM acceptance test against a real GitHub remote and merge.

**Resources** (as implemented):
- `rundeck_scm_import` - Configure Git/SVN import for a project
- `rundeck_scm_export` - Configure Git/SVN export for a project
- `config` is a generic string map since valid keys are plugin-specific and discovered at runtime
- Gated at API v15+ (ships with core Rundeck, not Enterprise-only)

---

### New Resources (Other)
**Effort**: Large (varies by resource)  
**Why Important**: Expands provider capabilities, but low user demand currently.

**Candidates**:
- `rundeck_node_source` - Dynamic node sources (Medium priority)
- `rundeck_local_user` - Enterprise local-user-store user management.
  `rundeck_local_role` (role CRUD + membership) is implemented; users are
  NOT, and can't be with the current SDK: the vendored client has no
  request-body support at all for the create/edit user endpoints
  (`PUT /secure/users/create`, `POST /secure/user/{id}`) - a gap in
  Rundeck's own published OpenAPI spec. Blocked until that's fixed
  upstream or the requests are hand-built, bypassing the generated client
  for those two calls.
- `rundeck_execution` - Trigger/manage executions (questionable use case)

**Completed in v1.4.0**:
- ✅ `rundeck_local_role` - Enterprise local user store role CRUD + membership management

**Completed in v1.2.0**:
- ✅ `rundeck_webhook` - Webhook event handlers (fully implemented with all 8 plugin types)

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

## Contributing

Want to tackle one of these tasks? Open an issue on GitHub to discuss your approach before starting.

**GitHub Issues**: Check [open issues](https://github.com/rundeck/terraform-provider-rundeck/issues) for community-reported bugs and feature requests.

**Maintainer**: Rundeck Team  
**Repository**: https://github.com/rundeck/terraform-provider-rundeck
