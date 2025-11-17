# Enterprise Rundeck Test Environment

This directory contains comprehensive test configurations for Rundeck Enterprise Edition (commercial version).

## Overview

- **Rundeck Version**: 5.17.0+
- **Edition**: Enterprise / Process Automation
- **API Version**: 56 (for runner features)
- **Deployment**: User-managed (not Docker)
- **Purpose**: Comprehensive feature validation including Enterprise-only functionality

## Prerequisites

### Required
- Running Rundeck Enterprise instance (5.17.0 or later)
- Valid API token with admin/sufficient permissions
- Go 1.24+ installed
- jq installed (`brew install jq` on macOS)

### Enterprise Features Tested
- ✅ Execution lifecycle plugins
- ✅ Project schedules
- ✅ System runners
- ✅ Project runners
- ✅ UUID-based job references
- ✅ All OSS features

## Quick Start

### 1. Update API Token

Edit `comprehensive.tf` and update the `auth_token`:

```hcl
provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "YOUR_ENTERPRISE_API_TOKEN_HERE"  # UPDATE THIS
  api_version = "56"
}
```

### 2. Run Comprehensive Test

```bash
cd test/enterprise
./comprehensive.sh
```

### 3. Review in UI

Open http://localhost:4440/project/comprehensive-test and review the created jobs.

## Files

| File | Purpose |
|------|---------|
| `comprehensive.tf` | Complete test configuration with 9 test jobs |
| `comprehensive.sh` | Automated test runner script |
| `README.md` | This file |

## Test Configuration

The `comprehensive.tf` file creates a comprehensive test environment with:

### Resources Created

**1 Project:**
- `comprehensive-test` - Test project with resource model source

**9 Jobs:**
1. **Node Dispatch Test** - Verifies nodefilters/dispatch structure fix
2. **Lifecycle Plugins Test** - Verifies execution lifecycle plugins work correctly
3a. **Target Job** - For UUID reference testing
3b. **Caller Job (UUID)** - Tests UUID-based job references (immutable)
4. **Complex Job** - Tests multiple features: schedules, options, notifications, log limits
5. **Local Execution** - Tests jobs without node dispatch
6. **Orchestrator Test** - Tests orchestrator configuration
7. **Caller Job (Name)** - Tests traditional name-based job references
8. **Import Test** - Minimal job for testing import functionality

### Critical Fixes Validated

The test configuration specifically validates recent critical fixes:

✅ **NodeFilters Structure Fix**
- Dispatch settings are correctly nested INSIDE nodefilters
- Jobs dispatch to nodes (not "Execute Locally")
- Structure: `nodefilters.dispatch` (not `dispatch` at root)

✅ **Execution Lifecycle Plugins Fix**
- Plugins use correct map structure (not array)
- Multiple plugins can be configured
- Plugin ordering is consistent (alphabetically sorted)

✅ **UUID Job References**
- Jobs can reference other jobs by immutable UUID
- Backward compatible with name-based references

## Running the Test

### Automated Run

The `comprehensive.sh` script performs a full test cycle:

```bash
./comprehensive.sh
```

**What it does:**
1. Cleans up previous test state
2. Builds the provider from source
3. Sets up dev overrides
4. Shows terraform plan
5. Prompts for confirmation
6. Applies configuration
7. Verifies nodefilters structure via API
8. Verifies lifecycle plugins via API
9. Displays job UUIDs and summary

**Output includes:**
- ✅ Verification that dispatch is nested in nodefilters
- ✅ Verification that all 4 lifecycle plugins are present
- 📋 URLs to view resources in Rundeck UI
- 🔧 Instructions for testing import functionality

### Manual Run

If you prefer manual steps:

```bash
# Build provider
cd /Users/forrest/Documents/GitHub/terraform-provider-rundeck
go build -o terraform-provider-rundeck

# Setup dev overrides
cat > .terraformrc <<EOF
provider_installation {
  dev_overrides {
    "terraform-providers/rundeck" = "$PWD"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"

# Navigate to test directory
cd test/enterprise

# Plan and apply
terraform plan
terraform apply
```

## Verification Steps

After running the test, verify in the Rundeck UI:

### 1. Node Dispatch Test (Job #1)
- Open job in UI
- Check "Nodes" tab
- Should show node filter: `tags: web`
- Should NOT be "Execute Locally"
- Thread count should be: 10
- Rank order should be: ascending

### 2. Lifecycle Plugins Test (Job #2)
- Open job in UI
- Click "Edit Job"
- Scroll to "Execution Plugins" section
- Should see 4 plugins:
  - Retry-Failed-Nodes
  - killhandler (with killChilds=true)
  - refreshHealthCheckerCache (with enabled=true)
  - resume (with onRetry=true)

### 3. UUID Job Reference (Job #3b)
- Open "03b-Caller-Job-UUID" in UI
- Check workflow steps
- Step 2 should show job reference by UUID
- Should reference "03a-Target-Job"

### 4. Complex Job (Job #4)
- Should have schedule defined (disabled)
- Should have 2 options (environment, verbose)
- Should have webhook notifications (success/failure)
- Should have log limit configured

## Testing Import Functionality

The test includes a dedicated job for testing Terraform import:

```bash
# Get the job UUID
JOB_ID=$(terraform output -json job_ids | jq -r '.import_test')

# Remove from state
terraform state rm rundeck_job.import_test

# Import it back
terraform import rundeck_job.import_test $JOB_ID

# Verify no drift
terraform plan
# Should show: "No changes. Your infrastructure matches the configuration."
```

## API Verification

The test script automatically verifies the API responses:

### NodeFilters Structure

```bash
curl -H "X-Rundeck-Auth-Token: YOUR_TOKEN" \
     "http://localhost:4440/api/56/job/JOB_ID?format=json" | \
     jq '.[0].nodefilters'
```

**Expected structure:**
```json
{
  "nodefilters": {
    "dispatch": {
      "threadcount": "10",
      "keepgoing": true,
      "rankOrder": "ascending",
      ...
    },
    "filter": "tags: web"
  }
}
```

### Lifecycle Plugins Structure

```bash
curl -H "X-Rundeck-Auth-Token: YOUR_TOKEN" \
     "http://localhost:4440/api/56/job/JOB_ID?format=json" | \
     jq '.[0].plugins.ExecutionLifecycle'
```

**Expected structure:**
```json
{
  "Retry-Failed-Nodes": {},
  "killhandler": {
    "killChilds": "true"
  },
  "refreshHealthCheckerCache": {
    "enabled": "true"
  },
  "resume": {
    "onRetry": "true"
  }
}
```

## Cleanup

### Remove Test Resources

```bash
cd test/enterprise
terraform destroy -auto-approve
```

### Delete Project via API

```bash
curl -X DELETE \
     -H "X-Rundeck-Auth-Token: YOUR_TOKEN" \
     "http://localhost:4440/api/56/project/comprehensive-test"
```

## Customization

### Using Different Rundeck Instance

Update `comprehensive.tf`:

```hcl
provider "rundeck" {
  url         = "https://your-rundeck-instance.com"  # Change this
  auth_token  = "your-api-token"                      # Change this
  api_version = "56"
}
```

### Adding More Test Cases

The `comprehensive.tf` file is structured to be easily extended:

```hcl
resource "rundeck_job" "my_new_test" {
  project_name = rundeck_project.comprehensive_test.name
  name         = "09-My-New-Test"
  description  = "Description of what this test validates"
  
  # ... job configuration
}
```

Add corresponding output:

```hcl
output "job_ids" {
  value = {
    # ... existing jobs
    my_new_test = rundeck_job.my_new_test.id
  }
}
```

## Troubleshooting

### "Project already exists" Error

```bash
# Delete the project
curl -X DELETE \
     -H "X-Rundeck-Auth-Token: YOUR_TOKEN" \
     "http://localhost:4440/api/56/project/comprehensive-test"

# Or use different project name in comprehensive.tf
```

### "Provider produced inconsistent result" Error

This was a bug that's now fixed. If you see this:
1. Ensure you're running the latest provider code
2. Check that the fix for plugin ordering is present
3. Rebuild the provider: `go build`

### API Token Unauthorized

```bash
# Test token
curl -H "X-Rundeck-Auth-Token: YOUR_TOKEN" \
     "http://localhost:4440/api/56/system/info"

# Should return system info, not 403
```

### Enterprise Features Not Available

Ensure you're running Rundeck Enterprise 5.17.0+:

```bash
curl -H "X-Rundeck-Auth-Token: YOUR_TOKEN" \
     "http://localhost:4440/api/56/system/info" | \
     jq '.system.rundeck | {version, build}'
```

## Test Results

After a successful run, you should see:

```
✅ VERIFIED: dispatch is nested inside nodefilters
✅ VERIFIED: All 4 execution lifecycle plugins present

Comprehensive Test Complete!
```

**What this confirms:**
- Provider correctly structures nodefilters with nested dispatch
- Execution lifecycle plugins use correct map format
- All job configurations are applied correctly
- No state inconsistency errors
- Import functionality works

## Notes

- **API Version**: Uses v56 to test runner features
- **Token Permissions**: Requires admin or equivalent permissions
- **Resource Files**: Jobs use placeholder resource file path
- **State Management**: Test creates `terraform.tfstate` - don't commit
- **Repeatable**: Can be run multiple times (with cleanup)

## Next Steps

1. Run test and verify all jobs created successfully
2. Execute jobs in UI to test runtime behavior
3. Test import functionality
4. Customize configuration for your specific test scenarios
5. Report any issues found during testing

## Contributing

When modifying the Enterprise test:

1. Keep test cases focused and well-documented
2. Add comments explaining what each test validates
3. Update this README if adding new tests
4. Ensure cleanup works properly
5. Test against Rundeck Enterprise 5.17.0+

