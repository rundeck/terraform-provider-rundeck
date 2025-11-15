# API Version Requirements by Resource

This document outlines the minimum Rundeck API versions required for each Terraform resource in the provider.

## Summary

**Minimum Supported Rundeck Version**: **5.0.0** (API v46+)

This provider requires Rundeck 5.0.0 or later to ensure all resources work correctly with JSON-only interactions and modern API features.

---

## Resource-by-Resource Breakdown

### 1. Storage Key Resources (Private Key, Public Key, Password)

**Framework Resources**:
- `rundeck_private_key`
- `rundeck_public_key`
- `rundeck_password`

**API Endpoints Used**:
- `GET /api/{version}/storage/keys/{path}` - Get key metadata
- `POST /api/{version}/storage/keys/{path}` - Create key
- `PUT /api/{version}/storage/keys/{path}` - Update key
- `DELETE /api/{version}/storage/keys/{path}` - Delete key

**API Version**: **v46** (provider default)
- Storage Keys API originally introduced in v11, provider uses v46

**Rundeck Version**: **5.0.0+** (provider minimum)

---

### 2. ACL Policy Resource

**Framework Resource**:
- `rundeck_acl_policy`

**API Endpoints Used**:
- `GET /api/{version}/system/acl/{policy}.aclpolicy` - Get ACL policy
- `POST /api/{version}/system/acl/{policy}.aclpolicy` - Create ACL policy
- `PUT /api/{version}/system/acl/{policy}.aclpolicy` - Update ACL policy
- `DELETE /api/{version}/system/acl/{policy}.aclpolicy` - Delete ACL policy

**API Version**: **v46** (provider default)
- System ACL API originally introduced in v14, provider uses v46

**Rundeck Version**: **5.0.0+** (provider minimum)

---

### 3. Project Resource

**Framework Resource**:
- `rundeck_project`

**API Endpoints Used**:
- `GET /api/{version}/project/{name}` - Get project
- `POST /api/{version}/projects` - Create project
- `DELETE /api/{version}/project/{name}` - Delete project
- `GET /api/{version}/project/{name}/config` - Get project config
- `PUT /api/{version}/project/{name}/config` - Update project config

**API Version**: **v46** (provider default)
- Project API originally introduced in v11, provider uses v46

**Rundeck Version**: **5.0.0+** (provider minimum)

---

### 4. Job Resource

**Framework Resource**:
- `rundeck_job`

**API Endpoints Used**:
- `GET /api/{version}/job/{id}?format=json` - Get job (JSON format available at v44+, default at v46+)
- `DELETE /api/{version}/job/{id}` - Delete job
- `POST /api/{version}/project/{project}/jobs/import?fileformat=json` - Import job (JSON format)

**API Version**: **v46** (provider default, enforced minimum for job operations)
- Job import with JSON format requires API v44+
- Job get with JSON format requires API v44+
- Provider automatically enforces v46 minimum for all job operations

**Rundeck Version**: **5.0.0+** (provider minimum)

**Note**: The job resource uses JSON exclusively for all job operations (create, read, update), providing a modern, maintainable format.

---

### 5. Runner Resources (Enterprise)

**Framework Resources**:
- `rundeck_system_runner`
- `rundeck_project_runner`

**API Endpoints Used**:
- `POST /api/{version}/runner` - Create system runner
- `POST /api/{version}/project/{project}/runner` - Create project runner
- `GET /api/{version}/runner/{id}` - Get runner info
- `PUT /api/{version}/runner/{id}` - Update system runner
- `PUT /api/{version}/project/{project}/runner/{id}` - Update project runner
- `DELETE /api/{version}/runner/{id}` - Delete system runner
- `DELETE /api/{version}/project/{project}/runner/{id}` - Delete project runner
- `PUT /api/{version}/project/{project}/runner/node-dispatch` - Configure node dispatch

**API Version**: **v56** (required for runners)
- Runner API is an Enterprise feature requiring API v56+

**Rundeck Version**: **Rundeck Enterprise 5.x** (with API v56+)

**Status**: ⚠️ Currently blocked by OpenAPI spec mismatch (enum casing issue). Will be fully functional once Go SDK is updated with corrected OpenAPI spec.

---

## API Version Compatibility Matrix

| Resource | API Version | Rundeck Version | Status |
|----------|-------------|-----------------|--------|
| `rundeck_private_key` | v46 | 5.0.0+ | ✅ Working |
| `rundeck_public_key` | v46 | 5.0.0+ | ✅ Working |
| `rundeck_password` | v46 | 5.0.0+ | ✅ Working |
| `rundeck_acl_policy` | v46 | 5.0.0+ | ✅ Working |
| `rundeck_project` | v46 | 5.0.0+ | ✅ Working |
| `rundeck_job` | v46 | 5.0.0+ | ✅ Working |
| `rundeck_system_runner` | v56 | Enterprise 5.x (v56+) | 🟡 Blocked on SDK |
| `rundeck_project_runner` | v56 | Enterprise 5.x (v56+) | 🟡 Blocked on SDK |

**Most resources require Rundeck 5.0.0+ (API v46). Runner resources require API v56+ (Enterprise).**

---

## Provider API Version Configuration

The provider defaults to **API v46** (Rundeck 5.0.0) for all operations:

```hcl
provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "your-token"
  api_version = "56"  # Optional - defaults to "46" (Rundeck 5.0.0)
}
```

### How API Versions Are Applied

1. **Storage Keys, ACL Policies, Projects**: Use the configured `api_version` (defaults to v46)
2. **Jobs**: Enforce minimum v46 for import operations
3. **Runners (Enterprise)**: Require API v56+ - provider will use v56 if configured version is lower

### Overriding the Default

If you have a newer Rundeck installation, you can specify a higher API version:

```hcl
provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "your-token"
  api_version = "56"  # Use latest API version
}
```

---

## Recommendations

### For Community Edition Users

**Required Version**: Rundeck **5.0.0+** (API v46)

This provider requires Rundeck 5.0.0 or later:
- ✅ All resources work correctly
- ✅ JSON-only job definitions (modern, maintainable format)
- ✅ Clean baseline for modern infrastructure-as-code
- ✅ Aligned with Rundeck 5.0 major release

### For Enterprise Edition Users

**Required Version**: Rundeck Enterprise **5.0.0+** (API v46)

Includes:
- ✅ All Community resources
- ✅ System and Project Runners (once SDK is updated)
- ✅ Enterprise-specific features and stability

### For Users on Older Rundeck Versions

If you are using Rundeck < 5.0.0:
- ⚠️ This provider version will **not work**
- ⚠️ Use an older version of this provider, or upgrade Rundeck

**Migration Path**:
1. Upgrade Rundeck to 5.0.0 or later
2. Update provider to this version
3. Existing Terraform configurations continue to work unchanged
4. Benefit from improved stability and modern API features

---

## API Version History

For reference, here are the key Rundeck releases and their API versions:

| Rundeck Version | API Version | Release Date | Notes |
|----------------|-------------|--------------|-------|
| 2.6.0 | v11 | June 2016 | Storage Keys, Projects API |
| 2.6.0 | v14 | June 2016 | System ACL API |
| 2.8.0 | v18 | June 2017 | Job format parameter |
| 3.3.0 | v31 | July 2020 | Job forecast |
| 3.4.0 | v36 | April 2021 | Config API |
| 4.0.0 | v40 | February 2022 | Execution result data |
| 4.8.0 | v44 | September 2023 | JSON job import, Runner API |
| **5.0.0** | **v46** | **Rundeck 5.0** | **Provider minimum version** |
| 5.x (latest) | v56+ | Current | Latest features |

---

## Testing Configuration

The provider's CI/CD tests use the following configuration:

```bash
RUNDECK_URL=http://localhost:4440
RUNDECK_AUTH_TOKEN=<token>
RUNDECK_API_VERSION=46  # Default (Rundeck 5.0.0)
```

The tests run against Rundeck 5.0.0+ to ensure full compatibility with all resources.

---

## Migration Notes

### From XML to JSON Job Definitions

The new Job resource uses JSON exclusively for all interactions with the Rundeck API:

**Job Configuration** (unchanged for users):
```hcl
resource "rundeck_job" "example" {
  project_name = "my-project"
  name         = "example"
  description  = "Example job"
  
  command {
    shell_command = "echo Hello"
  }
}
```

**Behind the scenes**:
- ✅ Provider uses JSON format for all job operations
- ✅ Clean, modern API interactions
- ✅ No user configuration changes needed

### Upgrading from Older Provider Versions

If you're upgrading from an older provider version that supported Rundeck < 5.0.0:

1. **Upgrade Rundeck first**: Ensure you're running Rundeck 5.0.0 or later
2. **Update provider version**: Upgrade to this provider version
3. **No config changes needed**: Your existing Terraform configurations will work unchanged
4. **Test in non-production**: Run `terraform plan` to verify no unexpected changes

---

## Future Considerations

### API Version Evolution

This provider uses **API v46** (Rundeck 5.0.0) as the baseline. Future considerations:

1. **Newer Rundeck versions**: Users can configure higher API versions to use newer features
2. **Deprecation**: If Rundeck deprecates v46, the provider will update to the new minimum
3. **Backward compatibility**: Major version bumps will be used for breaking changes

The v46 baseline provides a stable foundation aligned with Rundeck's major 5.0 release.

---

## Questions?

For technical details about specific API endpoints, refer to the [Rundeck API Documentation](https://docs.rundeck.com/docs/api/).

For provider-specific questions, please open a GitHub issue.

