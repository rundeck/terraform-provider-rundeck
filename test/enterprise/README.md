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

### 1. Set Environment Variables

```bash
export RUNDECK_URL="http://localhost:4440"
export RUNDECK_TOKEN="your-api-token-here"
```

### 2. Run Comprehensive Test

```bash
cd test/enterprise
./comprehensive.sh
```

The script will:
- Clean up any previous test state
- Build the provider binary locally (portable, no absolute paths)
- Configure Terraform to use the local provider build
- Run `terraform plan` and `terraform apply`
- Verify nodefilters and lifecycle plugin structures via API
- Display a summary of what to verify in the UI

### 3. Review in UI

Open http://localhost:4440/project/comprehensive-test and review the created jobs.

---

## Testing Your Own Plans

### "Bring Your Own Plan" - `test-custom.sh`

**NEW**: Test your own Terraform configurations against the provider, including dev branches!

#### Use Cases
- ✅ Test your production configs before applying
- ✅ Reproduce issues for bug reports
- ✅ Validate dev branches before releases
- ✅ Early adopter testing

#### Quick Start

```bash
# 1. Set environment variables
export RUNDECK_URL="http://localhost:4440"
export RUNDECK_TOKEN="your-api-token"

# 2. Run with your plan directory
cd test/enterprise
./test-custom.sh /path/to/your/terraform/config

# Or test in current directory
cd /path/to/your/terraform/config
/path/to/test-custom.sh .

# Or run and auto-cleanup after
./test-custom.sh /path/to/your/terraform/config --destroy-after

# Cleanup only
./test-custom.sh /path/to/your/terraform/config destroy
```

#### What It Does

The script automatically:
1. ✅ Cleans up previous test artifacts
2. ✅ Builds the provider from source
3. ✅ Sets up dev overrides (uses local build)
4. ✅ Initializes Terraform (if needed)
5. ✅ Shows plan for review
6. ✅ Asks for confirmation before apply
7. ✅ Applies your configuration
8. ✅ **Checks for drift** (automatic validation)
9. ✅ (Optional) Auto-cleanup if `--destroy-after` used
10. ✅ Provides cleanup instructions

#### Example: Testing a Bug Report

```bash
# Customer reports an issue with their config
cd /tmp/customer-issue
cat > main.tf <<EOF
resource "rundeck_job" "test" {
  project_name = "test"
  name = "my-job"
  # ... their configuration
}
EOF

# Test it
export RUNDECK_TOKEN="your-token"
/path/to/test-custom.sh .

# If issue reproduces, report it with:
# - Your .tf files
# - Script output
# - Drift check results
```

#### Testing Dev Branches

```bash
# Early adopter wants to test a fix before release
cd terraform-provider-rundeck
git checkout fix/some-issue

# Test with their production config
cd test/enterprise
export RUNDECK_TOKEN="..."
./test-custom.sh /path/to/their/production/config

# Validates the fix works with real-world usage!
```

#### Output

The script provides:
- Clear step-by-step progress
- **Automatic drift detection**
- Summary with drift status
- Cleanup instructions
- Links to report issues

#### Troubleshooting

**Token Authorization Errors (403)**

If you see `unauthorized` or `StatusCode=403`:

```bash
# 1. Check if token is hardcoded in tfvars files
grep -r "rundeck_token" /path/to/your/config/*.tfvars

# 2. If found, update the file with your current token
# OR remove it and rely on environment variables only

# 3. Verify token works via API
curl -H "X-Rundeck-Auth-Token: $RUNDECK_TOKEN" \
     "$RUNDECK_URL/api/56/system/info"
```

**Common cause**: Old tokens hardcoded in `terraform.tfvars` or `app.tfvars` files. The script uses `-var-file` which takes precedence over environment variables.

**Provider Build Errors**

If build fails:
```bash
# Ensure you're in the repo root
cd /path/to/terraform-provider-rundeck

# Check Go version (requires 1.24+)
go version

# Try manual build
go build -o terraform-provider-rundeck
```

**"No RUNDECK_TOKEN environment variable" Error**

```bash
# Ensure token is exported BEFORE running script
export RUNDECK_TOKEN="your-token-here"
export RUNDECK_URL="http://localhost:4440"

# Then run
./test-custom.sh /path/to/config
```

**Drift Detected After Apply**

If drift is detected, this may indicate a provider bug:
1. Note which resource(s) show drift
2. Capture the drift output
3. Include in your GitHub issue report
4. This is valuable feedback for identifying issues!

**"Inconsistent dependency lock file" Error**

If you see this error:
```
Error: Inconsistent dependency lock file
The following dependency selections recorded in the lock file are inconsistent with the current configuration:
  - provider registry.terraform.io/rundeck/rundeck: required by this configuration but no version is selected
```

**Solution**: The script automatically handles this by running `terraform init` once to create/update `.terraform.lock.hcl`. If this fails:

```bash
# Manually create the lock file
cd /path/to/your/config
unset TF_CLI_CONFIG_FILE  # Temporarily disable dev overrides
terraform init -upgrade
# Then re-run test-custom.sh
```

**Note**: The lock file is required even with dev overrides. The script creates it automatically on first run.

---

## Portability & Community Sharing

This test is designed to be **fully portable** and community-friendly:

✅ **No hardcoded paths** - Provider binary is built into the test directory  
✅ **No hardcoded credentials** - Uses environment variables  
✅ **Self-contained** - All artifacts stay in the test directory  
✅ **Automatic cleanup** - Script handles cleanup of generated files  
✅ **Works anywhere** - Uses relative paths and dynamic path resolution  

**Generated files** (automatically cleaned up):
- `terraform-provider-rundeck` - Provider binary (built locally)
- `.terraformrc` - Dev overrides config (generated per run)
- `.terraform/` - Terraform state directory
- `terraform.tfstate*` - Terraform state files

All generated files are gitignored and can be safely removed with:
```bash
terraform destroy -auto-approve
rm -f terraform-provider-rundeck .terraformrc
```

## Files

| File | Purpose |
|------|---------|
| `comprehensive.tf` | Complete test configuration with 9 test jobs |
| `comprehensive.sh` | Automated comprehensive test runner |
| `test-custom.sh` | **NEW** - Test your own Terraform plans |
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
# Standard run (resources remain after test)
./comprehensive.sh

# Run and auto-cleanup after (good for CI/CD)
./comprehensive.sh --destroy-after

# Cleanup only
./comprehensive.sh destroy
```

**What it does:**
1. Cleans up previous test state
2. Builds the provider from source
3. Sets up dev overrides
4. Shows terraform plan
5. Applies configuration (auto-approve)
6. Verifies nodefilters structure via API
7. Verifies lifecycle plugins via API
8. Displays job UUIDs and summary
9. (Optional) Auto-cleanup if `--destroy-after` used

**Output includes:**
- ✅ Verification that dispatch is nested in nodefilters
- ✅ Verification that all 4 lifecycle plugins are present
- 📋 URLs to view resources in Rundeck UI
- 🔧 Instructions for testing import functionality
- 🧹 Cleanup instructions

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

### Automated Cleanup (Recommended)

```bash
cd test/enterprise
export RUNDECK_TOKEN="your-token"
./comprehensive.sh destroy
```

This handles everything: terraform destroy, API fallback if needed, and artifact removal.

### Manual Cleanup

**Via Terraform:**
```bash
cd test/enterprise
export TF_VAR_rundeck_url="http://localhost:4440"
export TF_VAR_rundeck_token="your-token"
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"
terraform destroy -auto-approve
rm -f terraform-provider-rundeck .terraformrc
```

**Via API (if terraform fails):**
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

