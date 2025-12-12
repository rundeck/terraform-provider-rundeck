# GitHub Copilot Instructions for terraform-provider-rundeck

## Project Overview

This is the official Terraform provider for Rundeck, enabling infrastructure-as-code management of Rundeck resources (jobs, projects, ACLs, runners, etc.).

**Current Version:** 1.0.0+  
**Framework:** Terraform Plugin Framework (SDKv2 removed in v1.0.0)  
**API:** JSON-only (XML support removed in v1.0.0)  
**Minimum Rundeck:** 5.0.0+ (API v46+)  
**Enterprise Features:** Runners require API v56+ (Rundeck Enterprise 5.17.0+)

## Architecture

### Core Components

1. **Provider** (`rundeck/provider_framework.go`)
   - Plugin Framework implementation
   - Authentication: API token or username/password
   - Default API version: 56 (was 46 pre-v1.0.0)

2. **Resources**
   - `rundeck_job` - Job definitions with commands, notifications, schedules
   - `rundeck_project` - Projects with resource models and configuration
   - `rundeck_system_runner` / `rundeck_project_runner` - Enterprise runners
   - `rundeck_acl_policy` - Access control policies
   - `rundeck_public_key` / `rundeck_private_key` / `rundeck_password` - Key storage

3. **API Interaction**
   - JSON-only via `go-rundeck` SDK v1.2.0+
   - Complex TO/FROM JSON converters for nested structures
   - Round-trip consistency critical (no plan drift)

## Critical Patterns & Requirements

### 1. Job Resource Command Types

The job resource has complex command types that must map correctly to Rundeck's API:

**Script Interpreter:**
```go
// API expects:
// "scriptInterpreter": "pwsh -f ${scriptfile}"  (string)
// "interpreterArgsQuoted": true                 (boolean at command level)

// NOT nested in an object!
```

**Script File Arguments:**
```go
// HCL: script_file_args
// API: "args" (not "scriptargs" or "script_file_args")
```

**Step & Node Step Plugins:**
```go
// API expects flat structure at command level:
// {
//   "type": "plugin-name",
//   "nodeStep": true,        // boolean for node step plugins
//   "configuration": {...}    // NOT "config"
// }

// NOT nested under "plugin" or "nodeStep" keys!
```

**Job References:**
```go
// Support both UUID (recommended) and name-based references
// Include node_step boolean and dispatch block for advanced options
```

**Error Handlers:**
```go
// keep_going_on_success belongs in error_handler block, not top-level command
```

**Execution Lifecycle Plugins:**
```go
// CRITICAL: API expects map, not array
// Correct:   {"killhandler": {"killChilds": "true"}}
// Wrong:     [{"type": "killhandler", "config": {...}}]
```

### 2. Notification System

**Critical Ordering:**
- Notifications MUST be sorted alphabetically by type (on_failure, on_start, on_success)
- Rundeck's API returns object format, not array
- Provider enforces deterministic ordering to prevent drift

**API Format Differences:**
- Create API: sends arrays `{"onsuccess": [...]}`
- Read API: returns objects `{"onsuccess": {...}}`
- Provider handles conversion in both directions

**Webhook Fields:**
- `format`, `http_method`, `webhook_urls` at top level, not nested

**Plugin Notifications:**
- Use `configuration` key, not `config`

### 3. Testing Requirements

**All code changes require:**
1. Acceptance tests that validate against real Rundeck API
2. Integration tests for complex features (jobs, orchestrators, runners)
3. Direct API validation where possible (not just Terraform state)

**Test Patterns:**
```go
// Standard acceptance test
func TestAccJob_feature(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: testAccJobConfig_feature(),
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("rundeck_job.test", "field", "value"),
                    // Add API validation where possible
                ),
            },
        },
    })
}

// Integration test with API validation
testAccJobValidateAPI(&jobID, func(jobData map[string]interface{}) error {
    // Verify actual API response, not just Terraform state
    if field := jobData["field"]; field != expected {
        return fmt.Errorf("API returned %v, expected %v", field, expected)
    }
    return nil
})
```

**Enterprise Features:**
- All Enterprise features require `RUNDECK_ENTERPRISE_TESTS=1` flag
- This includes: runners (system/project), project schedules, certain lifecycle plugins
- Runners specifically require API v56+ (Rundeck Enterprise 5.17.0+)
- Project schedules also require `RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1` (manual UI setup needed)
- Future Enterprise features should also use the `RUNDECK_ENTERPRISE_TESTS=1` flag pattern

### 4. Code Style & Quality

**Go Formatting:**
- Always run `make fmt` before committing
- CI/CD checks gofmt compliance
- No exceptions

**Converter Functions:**
- TO JSON: `convert*ToJSON()` - Terraform HCL → Rundeck API
- FROM JSON: `convert*FromJSON()` - Rundeck API → Terraform HCL
- Must be symmetrical (round-trip consistency)
- Only populate fields with actual values (avoid null/empty drift)

**Schema Definitions:**
- Use predefined object types (e.g., `commandObjectType`) for consistency
- Nested blocks for complex structures
- Optional vs Required clearly defined

### 5. Common Pitfalls to Avoid

**Field Naming:**
- Rundeck uses camelCase in API (e.g., `scriptInterpreter`, `enforcedValues`)
- Terraform uses snake_case in HCL (e.g., `script_interpreter`, `require_predefined_choice`)
- Always verify field names against actual API responses (not documentation)

**Boolean Handling:**
- Distinguish between `null`, `false`, and missing
- Use `types.BoolValue()` / `types.BoolNull()` appropriately
- Some fields infer defaults (e.g., `require_predefined_choice` infers `true` when `value_choices` present)

**Empty Blocks:**
- Don't create empty nested blocks - causes schema validation errors
- Only populate `cmdAttrs` map with non-null values

**Plan Drift:**
- Always test for drift: apply, refresh, plan (should show no changes)
- Normalization must be bidirectional (schedule cron, runner tags, etc.)

### 6. Documentation Standards

**No Emojis:**
- Official documentation (`website/docs/`) must not use emojis
- Use bold text and ALL CAPS for emphasis instead
- Example: `**ACTION REQUIRED:**` not `⚠️ Action Required:`

**Code Examples:**
- Provide both correct and incorrect examples when relevant
- Comment: `# Correct` and `# Incorrect` (no checkmarks/X marks)
- Always explain WHY something is required

**Migration Guides:**
- Place in `website/docs/guides/upgrading.html.markdown`
- Update navigation in `website/rundeck.erb`
- Link from affected resource docs

## Review Checklist for PRs

When reviewing PRs, check:

### Code Quality
- [ ] `make fmt` has been run
- [ ] No linter errors
- [ ] Proper error handling
- [ ] No hardcoded values (use test helpers/env vars)

### Testing
- [ ] Acceptance tests included for new features
- [ ] Tests validate against real API responses (not just state)
- [ ] Tests include both create and update scenarios
- [ ] Integration tests for complex features
- [ ] All tests pass locally and in CI/CD

### API Field Mapping
- [ ] Field names match actual API (verified against sample JSON, not just docs)
- [ ] TO/FROM JSON converters are symmetrical
- [ ] Boolean fields handle null vs false correctly
- [ ] Empty/null values don't cause drift

### Drift Prevention
- [ ] Round-trip test: apply → refresh → plan shows no changes
- [ ] Normalization matches Rundeck's behavior
- [ ] Deterministic ordering for collections

### Documentation
- [ ] No emojis in official docs
- [ ] Migration notes for breaking changes
- [ ] Code examples for new features
- [ ] CHANGELOG.md updated
- [ ] Resource docs updated if schema changes

### Backward Compatibility
- [ ] Breaking changes clearly documented
- [ ] Migration path provided
- [ ] Version requirements updated if needed

## Key Files to Understand

**Provider Core:**
- `rundeck/provider_framework.go` - Provider implementation
- `rundeck/provider_test.go` - Test helpers and configuration

**Job Resource (Most Complex):**
- `rundeck/resource_job_framework.go` - Schema and CRUD operations
- `rundeck/resource_job_converters.go` - TO/FROM JSON converters (2000+ lines)
- `rundeck/resource_job_command_schema.go` - Command nested block schema
- `rundeck/resource_job_test.go` - Standard tests
- `rundeck/resource_job_integration_test.go` - API validation tests
- `rundeck/resource_job_comprehensive_test.go` - Complex scenario tests

**Runner Resources (Enterprise):**
- `rundeck/resource_system_runner.go` / `resource_project_runner.go`
- `rundeck/types_runner_tags.go` - Custom semantic equality for tags

**Tests:**
- `rundeck/*_test.go` - Acceptance tests
- `test/enterprise/` - Manual testing scripts

## Known Limitations & Future Work

See `TODO.md` for current task list. Key areas:

1. **Project Schedule Support** - Requires additional API calls for validation
2. **Schema-level Validation** - Some validations done at runtime (could move to schema)
3. **Project extra_config Merge Behavior** - Current behavior is correct Terraform design

If a new PR will resolve something in TODO the file should also be updated.

The TODO.md should be forward looking only.  No need to document things that are complete.

Don't use numbering in the TODO.md so updates are focused to additions/removals.

## Getting Help

**For Complex Changes:**
1. Read existing converters for similar fields
2. Export sample job from Rundeck UI as JSON
3. Compare JSON structure to current converter logic
4. Add comprehensive tests with real API validation

**For Testing:**
1. Use `test/enterprise/test-custom.sh` for manual testing
2. Check existing integration tests for patterns
3. Always validate against real Rundeck instance

**For API Questions:**
1. Rundeck API docs: https://docs.rundeck.com/docs/api/
2. Export jobs from UI to see actual API format
3. Our integration tests query API directly - good reference

## Summary for AI Reviewers

**Most Important:**
1. No plan drift - round-trip consistency is critical
2. All code changes need acceptance tests with real API validation
3. Field mapping must match actual API (verify with sample JSON exports)
4. Follow existing patterns in converters (don't invent new approaches)
5. Run `make fmt` before committing
6. No emojis in official documentation

**Red Flags:**
- Hardcoded API versions (use test helpers)
- Array vs Map confusion (lifecycle plugins, notifications)
- Field name mismatches (API camelCase vs HCL snake_case)
- Missing round-trip tests
- Drift after apply → refresh → plan
- Empty nested blocks in schema

**Green Flags:**
- Direct API validation in tests
- Symmetric TO/FROM JSON converters
- Sample JSON exports referenced in comments
- Comprehensive test coverage
- Clear migration documentation for breaking changes

