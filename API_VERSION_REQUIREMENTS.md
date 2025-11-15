# API Version Requirements by Resource

This document outlines the minimum Rundeck API versions required for each Terraform resource in the provider.

## Summary

**Recommended Minimum Rundeck Version**: **4.17.0** (API v44+)

This ensures all resources, including Job resources with JSON support, work correctly.

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

**Minimum API Version**: **v11**
- Storage Keys API introduced in Rundeck API v11

**Rundeck Version**: **2.6.0+** (released June 2016)

---

### 2. ACL Policy Resource

**Framework Resource**:
- `rundeck_acl_policy`

**API Endpoints Used**:
- `GET /api/{version}/system/acl/{policy}.aclpolicy` - Get ACL policy
- `POST /api/{version}/system/acl/{policy}.aclpolicy` - Create ACL policy
- `PUT /api/{version}/system/acl/{policy}.aclpolicy` - Update ACL policy
- `DELETE /api/{version}/system/acl/{policy}.aclpolicy` - Delete ACL policy

**Minimum API Version**: **v14**
- System ACL API introduced in Rundeck API v14

**Rundeck Version**: **2.6.0+** (released June 2016)

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

**Minimum API Version**: **v11**
- Project API introduced in Rundeck API v11
- Project config endpoint introduced in v11

**Rundeck Version**: **2.6.0+** (released June 2016)

---

### 4. Job Resource

**Framework Resource**:
- `rundeck_job`

**API Endpoints Used**:
- `GET /api/{version}/job/{id}?format=json` - Get job (uses v18+ for format parameter)
- `DELETE /api/{version}/job/{id}` - Delete job
- `POST /api/{version}/project/{project}/jobs/import` - Import job (JSON format)

**Minimum API Version**: **v44** (for JSON job import)
- Job import with JSON format requires API v44+
- XML job import supported since v1
- Job GET with format parameter since v18

**Rundeck Version**: **4.17.0+** (released September 2023)

**Note**: The provider automatically uses API v44+ for job import operations, even if the provider is configured with an older API version. This ensures JSON job definitions work correctly.

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

**Minimum API Version**: **v44** (Enterprise feature)
- Runner API is an Enterprise feature introduced in API v44

**Rundeck Version**: **Rundeck Enterprise 4.17.0+** (released September 2023)

**Status**: ⚠️ Currently blocked by OpenAPI spec mismatch (enum casing issue). Will be fully functional once Go SDK is updated.

---

## API Version Compatibility Matrix

| Resource | Min API Version | Min Rundeck Version | Status |
|----------|----------------|---------------------|--------|
| `rundeck_private_key` | v11 | 2.6.0+ | ✅ Working |
| `rundeck_public_key` | v11 | 2.6.0+ | ✅ Working |
| `rundeck_password` | v11 | 2.6.0+ | ✅ Working |
| `rundeck_acl_policy` | v14 | 2.6.0+ | ✅ Working |
| `rundeck_project` | v11 | 2.6.0+ | ✅ Working |
| `rundeck_job` | v44 | 4.17.0+ | ✅ Working |
| `rundeck_system_runner` | v44 | Enterprise 4.17.0+ | 🟡 Blocked on SDK |
| `rundeck_project_runner` | v44 | Enterprise 4.17.0+ | 🟡 Blocked on SDK |

---

## Provider API Version Configuration

The provider defaults to **API v14** for backward compatibility, but individual resources can use higher API versions as needed:

```hcl
provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "your-token"
  api_version = "56"  # Optional - defaults to "14"
}
```

### How API Versions Are Applied

1. **Storage Keys, ACL Policies, Projects**: Use the configured `api_version` (defaults to v14)
2. **Jobs**: Automatically use minimum v44 for import operations, regardless of configured version
3. **Runners**: Use the configured `api_version` (requires v44+)

---

## Recommendations

### For Community Edition Users

**Minimum Supported Version**: Rundeck **4.17.0+** (API v44)

While older resources (Storage Keys, ACL, Projects) work with Rundeck 2.6.0+, we recommend using Rundeck 4.17.0+ to ensure:
- ✅ All resources work correctly
- ✅ JSON job definitions (modern, maintainable format)
- ✅ Future-proof configuration

### For Enterprise Edition Users

**Minimum Supported Version**: Rundeck Enterprise **4.17.0+** (API v44)

Required for:
- ✅ All Community resources
- ✅ System and Project Runners (once SDK is updated)

### For Users on Older Rundeck Versions

If you must use Rundeck < 4.17.0:
- ⚠️ Job resources will **not work** (require v44+ for JSON import)
- ✅ Storage Keys, ACL Policies, and Projects **will work** with v11-v14

**Migration Path**:
1. Upgrade Rundeck to 4.17.0+
2. Update provider to this version
3. Existing Terraform configurations continue to work unchanged

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
| 4.8.0 | v44 | September 2023 | **JSON job import, Runner API** |
| 4.17.0 | v56 | Latest | Current version |

---

## Testing Configuration

The provider's CI/CD tests use the following configuration:

```bash
RUNDECK_URL=http://localhost:4440
RUNDECK_AUTH_TOKEN=<token>
RUNDECK_API_VERSION=14  # Default for backward compatibility
```

Even with API v14 configured, job resources automatically upgrade to v44 for import operations.

---

## Migration Notes

### From XML to JSON Job Definitions

The new Job resource uses JSON exclusively, but this is **fully backward compatible**:

**Old XML-based approach** (still works):
```hcl
resource "rundeck_job" "example" {
  project_name = "my-project"
  name         = "example"
  # XML was used internally
}
```

**New JSON-based approach** (same configuration):
```hcl
resource "rundeck_job" "example" {
  project_name = "my-project"
  name         = "example"
  # JSON is used internally - no config changes needed
}
```

Users see **no difference** - the provider handles the format internally.

---

## Future Considerations

### When Rundeck Deprecates API v14

If Rundeck deprecates API v14 in the future, the provider will need to:

1. Update the default `api_version` to the new minimum
2. Update documentation
3. Provide migration guide for users

This is not expected in the near future, as Rundeck maintains backward compatibility for older API versions.

---

## Questions?

For technical details about specific API endpoints, refer to the [Rundeck API Documentation](https://docs.rundeck.com/docs/api/).

For provider-specific questions, please open a GitHub issue.

