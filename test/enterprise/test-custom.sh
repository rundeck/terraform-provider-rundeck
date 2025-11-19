#!/bin/bash
set -e

# ==============================================================================
# Custom Plan Test Script
# ==============================================================================
# This script helps you test your own Terraform configurations against the
# provider, including testing dev branches before official releases.
#
# Usage:
#   export RUNDECK_URL="http://localhost:4440"
#   export RUNDECK_TOKEN="your-api-token"
#   ./test-custom.sh /path/to/your/plan/directory
#
# Or for quick local testing:
#   ./test-custom.sh .  # Use current directory
# ==============================================================================

# Check for required environment variables
if [ -z "$RUNDECK_TOKEN" ]; then
    echo "ERROR: RUNDECK_TOKEN environment variable must be set"
    echo "Usage: RUNDECK_TOKEN=your-token ./test-custom.sh <plan-directory>"
    exit 1
fi

RUNDECK_URL="${RUNDECK_URL:-http://localhost:4440}"

# Get the plan directory (default to current directory)
PLAN_DIR="${1:-.}"

if [ ! -d "$PLAN_DIR" ]; then
    echo "ERROR: Directory not found: $PLAN_DIR"
    echo "Usage: ./test-custom.sh <plan-directory>"
    exit 1
fi

# Convert to absolute path
PLAN_DIR="$(cd "$PLAN_DIR" && pwd)"

echo "=========================================="
echo "Custom Plan Test"
echo "=========================================="
echo "Rundeck URL: $RUNDECK_URL"
echo "Plan Directory: $PLAN_DIR"
echo ""

# Save current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Step 1: Clean up any previous test artifacts in script directory
echo "1. Cleaning up previous test state..."
rm -f terraform-provider-rundeck .terraformrc
echo "   ✓ Clean"
echo ""

# Step 2: Build the provider from repo root
echo "2. Building provider..."
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
echo "   Building from: $REPO_ROOT"
cd "$REPO_ROOT"
go clean -cache
go build -o "$SCRIPT_DIR/terraform-provider-rundeck"
cd "$SCRIPT_DIR"
echo "   ✓ Provider built: ./terraform-provider-rundeck"
echo ""

# Step 3: Create dev overrides config in script directory
echo "3. Setting up dev overrides..."
cat > .terraformrc <<EOF
provider_installation {
  dev_overrides {
    "rundeck/rundeck" = "$SCRIPT_DIR"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="$SCRIPT_DIR/.terraformrc"
echo "   ✓ Dev overrides configured"
echo ""

# Step 4: Export Terraform variables for provider configuration
echo "4. Setting up environment..."
export TF_VAR_rundeck_url="$RUNDECK_URL"
export TF_VAR_rundeck_token="$RUNDECK_TOKEN"
echo "   ✓ Environment configured"
echo ""

# Step 5: Change to plan directory
cd "$PLAN_DIR"
echo "5. Working in: $PLAN_DIR"
echo ""

# Step 6: Check for Terraform files
TF_FILES=$(find . -maxdepth 1 -name "*.tf" -type f | wc -l | tr -d ' ')
if [ "$TF_FILES" -eq 0 ]; then
    echo "⚠️  WARNING: No .tf files found in $PLAN_DIR"
    echo "   Make sure your Terraform configuration is in this directory"
    exit 1
fi
echo "   Found $TF_FILES Terraform file(s)"
echo ""

# Step 7: Initialize if needed (for lock file)
# Check if .terraform.lock.hcl exists, if not we need to init once
if [ ! -f ".terraform.lock.hcl" ]; then
    echo "6. Initializing Terraform (creating lock file)..."
    echo "   Note: Temporarily disabling dev overrides for init"
    
    # Save dev overrides config
    DEV_OVERRIDES_CONFIG="$TF_CLI_CONFIG_FILE"
    unset TF_CLI_CONFIG_FILE
    
    # Run init to create lock file
    terraform init -upgrade > /dev/null 2>&1
    INIT_RESULT=$?
    
    # Restore dev overrides
    export TF_CLI_CONFIG_FILE="$DEV_OVERRIDES_CONFIG"
    
    if [ $INIT_RESULT -eq 0 ]; then
        echo "   ✓ Lock file created"
    else
        echo "   ⚠️  Init completed (may have warnings)"
    fi
    echo ""
else
    echo "6. Lock file exists, skipping init"
    echo ""
fi

# Step 8: Check for tfvars files
TFVARS_ARGS=""
if [ -f "terraform.tfvars" ]; then
    TFVARS_ARGS="-var-file=terraform.tfvars"
    echo "7. Found terraform.tfvars"
elif [ -f "app.tfvars" ]; then
    TFVARS_ARGS="-var-file=app.tfvars"
    echo "7. Found app.tfvars"
else
    echo "7. No .tfvars file found (will use variables from .tf files or prompt)"
fi
echo ""

# Step 9: Show the plan
echo "8. Running Terraform plan..."
echo "=========================================="
terraform plan $TFVARS_ARGS
echo "=========================================="
echo ""

# Step 8: Ask for confirmation
read -p "Do you want to apply this configuration? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo ""
    echo "⚠️  Skipping apply. You can run manually:"
    echo "   cd $PLAN_DIR"
    echo "   export TF_CLI_CONFIG_FILE=$SCRIPT_DIR/.terraformrc"
    echo "   export TF_VAR_rundeck_url=$RUNDECK_URL"
    echo "   export TF_VAR_rundeck_token=\$RUNDECK_TOKEN"
    echo "   terraform apply"
    exit 0
fi
echo ""

# Step 10: Apply the configuration
echo "9. Applying Terraform configuration..."
echo "=========================================="
terraform apply $TFVARS_ARGS -auto-approve
echo "=========================================="
echo ""

# Step 11: Check for drift
echo "10. Checking for plan drift..."
echo "=========================================="
if terraform plan $TFVARS_ARGS -detailed-exitcode > /dev/null 2>&1; then
    echo "✅ SUCCESS: No drift detected!"
    DRIFT_STATUS="✅ No drift"
else
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 2 ]; then
        echo "⚠️  WARNING: Drift detected - resources need changes"
        terraform plan $TFVARS_ARGS
        DRIFT_STATUS="⚠️  Drift detected"
    else
        echo "❌ ERROR: terraform plan failed"
        DRIFT_STATUS="❌ Plan failed"
    fi
fi
echo "=========================================="
echo ""

# Step 12: Summary
echo "=========================================="
echo "Test Complete!"
echo "=========================================="
echo ""
echo "Plan Directory: $PLAN_DIR"
echo "Drift Status: $DRIFT_STATUS"
echo ""
echo "Next Steps:"
echo "1. Review resources in Rundeck UI: $RUNDECK_URL"
echo "2. Test functionality (run jobs, check configs, etc.)"
echo "3. Report any issues: https://github.com/rundeck/terraform-provider-rundeck/issues"
echo ""
echo "Cleanup:"
echo "  cd $PLAN_DIR"
echo "  terraform destroy -auto-approve"
echo ""
echo "Provider artifacts (safe to remove):"
echo "  rm -f $SCRIPT_DIR/terraform-provider-rundeck"
echo "  rm -f $SCRIPT_DIR/.terraformrc"
echo ""

# Return to script directory
cd "$SCRIPT_DIR"

echo "=========================================="
echo "Thank you for testing! 🎉"
echo "=========================================="

