# Enterprise Runner API - OpenAPI Spec Mismatch

## Summary

**RESOLVED: The issue was a missing feature flag, not an API bug.**

With the new Enterprise feature flags enabled, the Runner API now returns **lowercase** enum values for `installationType` and `replicaType`. However, the OpenAPI specification (`rundeck-api.yml`) defines these enums with **UPPERCASE** values, causing the generated Go SDK to fail when unmarshaling API responses.

## Root Cause: OpenAPI Spec vs. Actual API Mismatch

### OpenAPI Specification (rundeck-api.yml)

**Lines 13365-13371:**
```yaml
RunnerInstallationType:
  type: string
  enum:
    - LINUX      # UPPERCASE
    - WINDOWS
    - KUBERNETES
    - DOCKER
```

**Lines 13448-13452:**
```yaml
RunnerReplicaType:
  type: string
  enum:
    - ephemeral  # lowercase
    - manual
```

### Actual API Behavior (with new feature flags)

**Runner creation succeeds**, but API returns:
```json
{
  "installationType": "linux",     // lowercase (doesn't match spec)
  "replicaType": "manual"          // lowercase (matches spec)
}
```

### Go SDK Behavior

The generated Go SDK (`github.com/rundeck/go-rundeck/rundeck-v2`) uses strict enum types based on the OpenAPI spec:
- Expects `installationType` to be one of: `LINUX`, `WINDOWS`, `KUBERNETES`, `DOCKER`
- Receives: `linux` (lowercase)
- **Error:** `linux is not a valid RunnerInstallationType`

---

## Environment

- **Rundeck Version**: Enterprise 5.17.0 (with new feature flags **ENABLED**)
- **API Version**: 56
- **Endpoint Base**: `http://127.0.0.1:4440`
- **Authentication**: Token-based (`X-Rundeck-Auth-Token` header)
- **Go SDK**: `github.com/rundeck/go-rundeck/rundeck-v2` (needs update)

---

## Current Test Results (with new feature flags enabled)

### System Runner Tests

#### Step 1: Create System Runner (✅ SUCCESS)

**Endpoint:**
```
POST /api/56/runnerManagement/runner
```

**Request Headers:**
```
Content-Type: application/json
X-Rundeck-Auth-Token: ZqIzfkgqDl8FJgyVYXlxQW8VF9MUPAB1
```

**Request Body (corrected for new feature flags):**
```json
{
  "name": "test-system-runner",
  "description": "Test system runner",
  "tagNames": "test,terraform",
  "installationType": "linux",    // lowercase now required
  "replicaType": "manual"         // lowercase now required
}
```

**Response:** `200 OK`
```json
{
  "runnerId": "6c1cd1bc-091a-4d1e-ac90-c0be174c9140",
  "token": "...",
  "downloadTk": "..."
}
```

✅ **Runner Created Successfully**

---

#### Step 2: Read System Runner (❌ FAILS with Go SDK)

**Endpoint:**
```
GET /api/56/runnerManagement/runner/6c1cd1bc-091a-4d1e-ac90-c0be174c9140
```

**Request Headers:**
```
X-Rundeck-Auth-Token: mOW1ybfOkGqgoko3A0ABFJyCe0WJDBce
```

**Raw API Response:** `200 OK` ✅
```json
{
  "id": "6c1cd1bc-091a-4d1e-ac90-c0be174c9140",
  "name": "test-system-runner",
  "description": "Test system runner",
  "tagNames": ["test", "terraform"],
  "installationType": "linux",     // ❌ lowercase, SDK expects UPPERCASE
  "replicaType": "manual"
}
```

**Go SDK Error:** ❌
```
Error reading system runner: linux is not a valid RunnerInstallationType
```

**Root Cause:** The Go SDK's `RunnerInfo` struct has a typed enum field that only accepts uppercase values (`LINUX`, `WINDOWS`, etc.), but the API now returns lowercase.

---

### Project Runner Tests

#### Step 1: Create Project (✅ SUCCESS)

**Endpoint:**
```
POST /api/56/projects
```

**Request Body:**
```json
{
  "name": "terraform-acc-test-project-runner",
  "description": "Terraform Acceptance Tests Project for Runner",
  "config": {
    "project.resources.source.1.type": "local"
  }
}
```

**Response:** `201 Created`

✅ **Project Created Successfully**

---

#### Step 2: Create Project Runner (✅ SUCCESS)

**Endpoint:**
```
POST /api/56/project/terraform-acc-test-project-runner/runnerManagement/runner
```

**Request Headers:**
```
Content-Type: application/json
X-Rundeck-Auth-Token: ZqIzfkgqDl8FJgyVYXlxQW8VF9MUPAB1
```

**Request Body (corrected for new feature flags):**
```json
{
  "name": "test-project-runner",
  "description": "Test project runner",
  "newRunnerRequest": {
    "name": "test-project-runner",
    "description": "Test project runner",
    "tagNames": "test,terraform",
    "installationType": "linux",    // lowercase now required
    "replicaType": "manual"         // lowercase now required
  }
}
```

**Response:** `200 OK`
```json
{
  "runnerId": "ff85f15b-f479-4362-93b1-040ace15e2d1",
  "token": "...",
  "downloadTk": "..."
}
```

✅ **Project Runner Created Successfully**

---

#### Step 3: Read Project Runner (❌ FAILS with Go SDK)

**Endpoint (general RunnerInfo works, project-specific returns 404):**
```
GET /api/56/runnerManagement/runner/ff85f15b-f479-4362-93b1-040ace15e2d1
```

**Raw API Response:** `200 OK` ✅
```json
{
  "id": "ff85f15b-f479-4362-93b1-040ace15e2d1",
  "name": "test-project-runner",
  "description": "Test project runner",
  "tagNames": ["test", "terraform"],
  "installationType": "linux",     // ❌ lowercase, SDK expects UPPERCASE
  "replicaType": "manual"
}
```

**Go SDK Error:** ❌
```
failed to get runner info: linux is not a valid RunnerInstallationType
```

**Root Cause:** Same OpenAPI spec mismatch as system runners.

---

## Solution

### Required Action: Update Go SDK

The `github.com/rundeck/go-rundeck/rundeck-v2` SDK needs to be regenerated from an updated OpenAPI specification that reflects the new API behavior:

**OpenAPI Spec Changes Needed:**
```yaml
RunnerInstallationType:
  type: string
  enum:
    - linux        # Change from LINUX to linux
    - windows      # Change from WINDOWS to windows
    - kubernetes   # Change from KUBERNETES to kubernetes  
    - docker       # Change from DOCKER to docker
```

### Temporary Workaround

Until the Go SDK is updated, the Terraform Provider could:
1. Parse these fields as plain strings instead of using the typed SDK structs
2. Make raw HTTP calls for Runner operations instead of using the SDK
3. Wait for the updated SDK release

**Recommendation:** Wait for the updated Go SDK from the pending PR mentioned by the user.

---

## Expected Behavior

Based on the OpenAPI specification (`rundeck-api.yml`):

### RunnerInfo Endpoint (lines 9277-9295):
```yaml
/runnerManagement/runner/{runnerId}:
  get:
    tags:
      - Runner
    summary: Get Runner Info
    description: "Get runner information. Since: V41"
    operationId: runnerInfo
    parameters:
      - name: runnerId
        in: path
        description: Runner ID
        required: true
        schema:
          type: string
    responses:
      "200":
        description: Runner Info
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/RunnerInfo"
```

**Expected Response:** `200 OK` with RunnerInfo JSON

---

## Actual Behavior

**All read attempts return:** `500 Server Error`

No response body is returned with the 500 error (or it's empty).

---

## Impact

This prevents the Terraform Provider from:
1. Verifying that runners were created successfully
2. Reading runner state for Terraform refresh operations
3. Running acceptance tests for runner resources
4. Properly managing runner lifecycle

---

## Workaround Attempted

Tried using the project-specific `ProjectRunnerInfo` endpoint:
```
GET /api/56/project/{project}/runnerManagement/runner/{runnerId}
```

**Result:** Returns `404 Not Found` instead of `500`, but still cannot read the runner.

---

## Code References

### Provider Implementation
- System Runner Resource: `rundeck/resource_system_runner_framework.go`
- Project Runner Resource: `rundeck/resource_project_runner_framework.go`
- Test Helpers: `rundeck/resource_system_runner_test.go`, `rundeck/resource_project_runner_test.go`

### API Calls Made
```go
// Create System Runner (WORKS)
response, _, err := client.RunnerAPI.CreateRunner(apiCtx).
    CreateProjectRunnerRequest(*runnerRequest).
    Execute()

// Read System Runner (500 ERROR)
runnerInfo, apiResp, err := client.RunnerAPI.RunnerInfo(apiCtx, runnerId).
    Execute()

// Create Project Runner (WORKS)
response, _, err := client.RunnerAPI.CreateProjectRunner(apiCtx, projectName).
    CreateProjectRunnerRequest(*projectRunnerRequest).
    Execute()

// Read Project Runner via ProjectRunnerInfo (404 ERROR)
runnerInfo, apiResp, err := client.RunnerAPI.ProjectRunnerInfo(apiCtx, projectName, runnerId).
    Execute()

// Read Project Runner via general RunnerInfo (500 ERROR)
runnerInfo, apiResp, err := client.RunnerAPI.RunnerInfo(apiCtx, runnerId).
    Execute()
```

---

## Test Results Summary

| Test | Create | Read (API) | Read (SDK) | Status |
|------|--------|------------|------------|--------|
| System Runner | ✅ 200 OK | ✅ 200 OK | ❌ Enum error | Blocked on SDK |
| Project Runner | ✅ 200 OK | ✅ 200 OK | ❌ Enum error | Blocked on SDK |
| Project Runner + Node Dispatch | ✅ 200 OK | ✅ 200 OK | ❌ Enum error | Blocked on SDK |

**Conclusion:** Runner API works correctly. The issue is the Go SDK enum validation, which will be resolved when the SDK is regenerated from the updated OpenAPI specification.

---

## Reproduction Steps

1. Start Rundeck Enterprise instance
2. Create a runner via POST to `/api/56/runnerManagement/runner` (succeeds)
3. Note the `runnerId` from the response
4. Immediately attempt GET to `/api/56/runnerManagement/runner/{runnerId}`
5. Observe 500 Server Error

**This is reproducible 100% of the time across multiple test runs.**

---

## System Information

- **Test Framework**: Go acceptance tests using `github.com/rundeck/go-rundeck/rundeck-v2` (OpenAPI-generated client)
- **Go Version**: 1.24.10
- **Provider Version**: Development (modernization branch)
- **Test Command**: 
```bash
TF_ACC=1 RUNDECK_ENTERPRISE_TESTS=1 \
RUNDECK_URL=http://127.0.0.1:4440 \
RUNDECK_AUTH_TOKEN=ZqIzfkgqDl8FJgyVYXlxQW8VF9MUPAB1 \
RUNDECK_API_VERSION=56 \
go test -v -run "TestAccRundeckSystemRunner|TestAccRundeckProjectRunner" \
-timeout 30m ./rundeck
```

---

## Request for Engineering Team

Please investigate why the `RunnerInfo` endpoint returns 500 Server Error for runners that were just successfully created. Server logs and stack traces would be very helpful in diagnosing this issue.

Thank you!

