#!/bin/bash
set -e

echo "=========================================="
echo "Comprehensive Enterprise Test"
echo "=========================================="
echo

# Clean up any previous test state
echo "1. Cleaning up previous test state..."
rm -rf .terraform .terraform.lock.hcl terraform.tfstate terraform.tfstate.backup
echo "   ✓ Clean"
echo

# Build the provider
echo "2. Building provider..."
go clean -cache
go build -o terraform-provider-rundeck
echo "   ✓ Provider built"
echo

# Create dev overrides config
echo "3. Setting up dev overrides..."
cat > .terraformrc <<EOF
provider_installation {
  dev_overrides {
    "terraform-providers/rundeck" = "$PWD"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"
echo "   ✓ Dev overrides configured"
echo

# Show the plan
echo "4. Showing Terraform plan..."
echo "=========================================="
terraform plan
echo "=========================================="
echo

# Ask for confirmation
read -p "Do you want to apply this configuration? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "   ⚠ Skipping apply"
    exit 0
fi
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
curl -s -H "X-Rundeck-Auth-Token: ztW3s5kZtInFzaUlg1M3oLn81t8sAJtI" \
     -H "Accept: application/json" \
     "http://localhost:4440/api/56/job/$JOB_ID?format=json" | \
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
curl -s -H "X-Rundeck-Auth-Token: ztW3s5kZtInFzaUlg1M3oLn81t8sAJtI" \
     -H "Accept: application/json" \
     "http://localhost:4440/api/56/job/$PLUGIN_JOB_ID?format=json" | \
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
echo "1. Open Rundeck UI: http://localhost:4440/project/comprehensive-test"
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

