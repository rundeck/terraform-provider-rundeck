# Enterprise Runner API Bug Report

## Summary

When creating runners via the Enterprise API (v56), the creation succeeds and returns a valid runner ID. However, subsequent attempts to read the runner information via the `RunnerInfo` endpoint consistently return **500 Server Error**.

## Environment

- **Rundeck Version**: Enterprise (with API v56)
- **API Version**: 56
- **Endpoint Base**: `http://127.0.0.1:4440`
- **Authentication**: Token-based (`X-Rundeck-Auth-Token` header)

---

## Issue 1: System Runner Creation + Read Failure

### Step 1: Create System Runner (✅ SUCCESS)

**Endpoint:**
```
POST /api/56/runnerManagement/runner
```

**Request Headers:**
```
Content-Type: application/json
X-Rundeck-Auth-Token: ZqIzfkgqDl8FJgyVYXlxQW8VF9MUPAB1
```

**Request Body:**
```json
{
  "name": "test-system-runner",
  "description": "Test system runner",
  "tagNames": "test,terraform",
  "installationType": "LINUX",
  "replicaType": "MANUAL"
}
```

**Response:** `200 OK`
```json
{
  "runnerId": "8122aa3e-57f5-4aa7-9e62-b287a925ec0d",
  "token": "...",
  "downloadTk": "..."
}
```

✅ **Runner Created Successfully** - ID: `8122aa3e-57f5-4aa7-9e62-b287a925ec0d`

---

### Step 2: Read System Runner (❌ FAILS)

**Endpoint:**
```
GET /api/56/runnerManagement/runner/8122aa3e-57f5-4aa7-9e62-b287a925ec0d
```

**Request Headers:**
```
X-Rundeck-Auth-Token: ZqIzfkgqDl8FJgyVYXlxQW8VF9MUPAB1
```

**Response:** `500 Server Error`

❌ **Error:** Cannot read back the runner that was just created

---

## Issue 2: Project Runner Creation + Read Failure

### Step 1: Create Project (✅ SUCCESS)

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

### Step 2: Create Project Runner (✅ SUCCESS)

**Endpoint:**
```
POST /api/56/project/terraform-acc-test-project-runner/runnerManagement/runner
```

**Request Headers:**
```
Content-Type: application/json
X-Rundeck-Auth-Token: ZqIzfkgqDl8FJgyVYXlxQW8VF9MUPAB1
```

**Request Body:**
```json
{
  "name": "test-project-runner",
  "description": "Test project runner",
  "newRunnerRequest": {
    "name": "test-project-runner",
    "description": "Test project runner",
    "tagNames": "test,terraform",
    "installationType": "LINUX",
    "replicaType": "MANUAL"
  }
}
```

**Response:** `200 OK`
```json
{
  "runnerId": "95324ab5-1186-41af-82ff-d52f1a9dfbb9",
  "token": "...",
  "downloadTk": "..."
}
```

✅ **Project Runner Created Successfully** - ID: `95324ab5-1186-41af-82ff-d52f1a9dfbb9`

---

### Step 3: Read Project Runner (❌ FAILS)

**Attempt 1 - Project-specific endpoint:**
```
GET /api/56/project/terraform-acc-test-project-runner/runnerManagement/runner/95324ab5-1186-41af-82ff-d52f1a9dfbb9
```

**Response:** `404 Not Found`

❌ **Error:** Runner not found via project-specific endpoint

---

**Attempt 2 - General RunnerInfo endpoint:**
```
GET /api/56/runnerManagement/runner/95324ab5-1186-41af-82ff-d52f1a9dfbb9
```

**Response:** `500 Server Error`

❌ **Error:** Same 500 error as system runners

---

## Additional Test Cases

Multiple test runs with different configurations all show the same pattern:

### Example Runner IDs Created Successfully:
- `8122aa3e-57f5-4aa7-9e62-b287a925ec0d` (System Runner)
- `95324ab5-1186-41af-82ff-d52f1a9dfbb9` (Project Runner)
- `92b2a68a-6253-4157-b217-360c477f3913` (Project Runner with Node Dispatch)
- `d14d13e6-774e-4447-a9e5-02a7404265a9` (Project Runner Update Test)

**All return 500 Server Error when attempting to read back via:**
```
GET /api/56/runnerManagement/runner/{runnerId}
```

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

## Questions for Engineering Team

1. **Is there a different endpoint we should use to read runner information?**
2. **Are there any prerequisites or waiting periods after runner creation?**
3. **Is the `RunnerInfo` endpoint supported in Enterprise API v56?**
4. **Should project runners be read via `ProjectRunnerInfo` or general `RunnerInfo`?**
5. **Are there any known issues with the Runner API in this version?**
6. **What logs can we check on the Rundeck server to diagnose the 500 error?**

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

