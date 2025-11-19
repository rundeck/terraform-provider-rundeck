#!/bin/bash
set -e

# Check for required environment variables
if [ -z "$RUNDECK_TOKEN" ]; then
    echo "ERROR: RUNDECK_TOKEN environment variable must be set"
    echo "Usage: RUNDECK_TOKEN=your-token ./comprehensive.sh"
    exit 1
fi

RUNDECK_URL="${RUNDECK_URL:-http://localhost:4440}"

echo "=========================================="
echo "Comprehensive Enterprise Test"
echo "=========================================="
echo "Rundeck URL: $RUNDECK_URL"
echo

# Clean up any previous test state
echo "1. Cleaning up previous test state..."
rm -rf .terraform .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
rm -f terraform-provider-rundeck .terraformrc
echo "   ✓ Clean"
echo

# Build the provider and place binary in this test directory
echo "2. Building provider..."
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
echo "   Building from: $REPO_ROOT"
cd "$REPO_ROOT"
go clean -cache
go build -o "$PWD/test/enterprise/terraform-provider-rundeck"
cd - > /dev/null  # Return to test directory
echo "   ✓ Provider built: ./terraform-provider-rundeck"
echo

# Create dev overrides config (using current directory for portability)
echo "3. Setting up dev overrides..."
TEST_DIR="$PWD"
cat > .terraformrc <<EOF
provider_installation {
  dev_overrides {
    "rundeck/rundeck" = "$TEST_DIR"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"
echo "   ✓ Dev overrides configured (pointing to current directory)"
echo

# Export Terraform variables
export TF_VAR_rundeck_url="$RUNDECK_URL"
export TF_VAR_rundeck_token="$RUNDECK_TOKEN"

# Show the plan
echo "4. Showing Terraform plan..."
echo "=========================================="
terraform plan
echo "=========================================="
echo

# Apply the configuration
echo "5. Applying Terraform configuration..."
echo "=========================================="
terraform apply -auto-approve
echo "=========================================="
echo

# Show the outputs
echo
echo "6. Test Summary:"
echo "=========================================="
terraform output -raw test_summary
echo "=========================================="
echo

# Fetch one of the jobs to verify structure
echo "7. Verifying nodefilters structure in created job..."
JOB_ID=$(terraform output -json job_ids | jq -r '.node_dispatch')
echo "   Job ID: $JOB_ID"
echo
curl -s -H "X-Rundeck-Auth-Token: ${RUNDECK_TOKEN}" \
     -H "Accept: application/json" \
     "${RUNDECK_URL}/api/56/job/$JOB_ID?format=json" | \
     jq '.[0] | {name, nodefilters}' | tee /tmp/enterprise_nodefilters_check.json
echo

if cat /tmp/enterprise_nodefilters_check.json | jq -e '.nodefilters.dispatch' > /dev/null 2>&1; then
    echo "   ✅ VERIFIED: dispatch is nested inside nodefilters"
else
    echo "   ⚠️  WARNING: dispatch structure may be incorrect"
fi
echo

# Verify lifecycle plugins
echo "8. Verifying execution lifecycle plugins structure..."
PLUGIN_JOB_ID=$(terraform output -json job_ids | jq -r '.lifecycle_plugins')
echo "   Job ID: $PLUGIN_JOB_ID"
echo
curl -s -H "X-Rundeck-Auth-Token: ${RUNDECK_TOKEN}" \
     -H "Accept: application/json" \
     "${RUNDECK_URL}/api/56/job/$PLUGIN_JOB_ID?format=json" | \
     jq '.[0].plugins.ExecutionLifecycle' | tee /tmp/enterprise_plugins_check.json
echo

PLUGIN_COUNT=$(cat /tmp/enterprise_plugins_check.json | jq 'length')
echo "   Plugin count: $PLUGIN_COUNT"
if [ "$PLUGIN_COUNT" -eq 4 ]; then
    echo "   ✅ VERIFIED: All 4 execution lifecycle plugins present"
else
    echo "   ⚠️  WARNING: Expected 4 plugins, found $PLUGIN_COUNT"
fi
echo

echo "=========================================="
echo "Comprehensive Test Complete!"
echo "=========================================="
echo
echo "Next Steps:"
echo "1. Open Rundeck UI: ${RUNDECK_URL}/project/comprehensive-test"
echo "2. Review each job to verify settings"
echo "3. Run a few jobs to test execution"
echo "4. Test import functionality if desired"
echo
echo "To test import:"
echo "  JOB_ID=\$(terraform output -json job_ids | jq -r '.import_test')"
echo "  terraform state rm rundeck_job.import_test"
echo "  terraform import rundeck_job.import_test \$JOB_ID"
echo "  terraform plan  # Should show no drift"
echo
echo "Cleanup:"
echo "  terraform destroy -auto-approve"
echo "  rm -f terraform-provider-rundeck .terraformrc"

