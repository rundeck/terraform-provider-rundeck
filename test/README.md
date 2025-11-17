# Terraform Provider for Rundeck - Test Environments

This directory contains test environments for validating the Rundeck Terraform Provider against both Open Source and Enterprise versions of Rundeck.

## Directory Structure

```
test/
├── README.md                    # This file
├── oss/                         # Open Source Rundeck tests
│   ├── README.md               # OSS setup & instructions
│   ├── docker-compose.yml      # Docker Compose configuration
│   ├── Dockerfile              # Rundeck OSS container
│   ├── remco/                  # Configuration management
│   └── tokens.properties       # API token configuration
└── enterprise/                  # Enterprise Rundeck tests
    ├── README.md               # Enterprise setup & instructions
    ├── comprehensive.tf        # Comprehensive test configuration
    └── comprehensive.sh        # Automated test runner
```

## Quick Start

### Open Source Testing

Test against Rundeck Open Source (Community Edition) using Docker:

```bash
cd test/oss
docker-compose up -d
# Wait ~30 seconds for Rundeck to start
# API Token: 1d08bf61-f962-467f-8ba3-ab8a463b3467
# URL: http://localhost:4440
```

See [test/oss/README.md](oss/README.md) for detailed instructions.

### Enterprise Testing

Test against Rundeck Enterprise Edition (commercial version):

```bash
cd test/enterprise
# Requires a running Rundeck Enterprise instance
# Update API token in comprehensive.tf
./comprehensive.sh
```

See [test/enterprise/README.md](enterprise/README.md) for detailed instructions.

## Test Environments

### Open Source (OSS)
- **Version**: Rundeck 5.17.0 Community Edition
- **API Version**: 46+
- **Deployment**: Docker Compose
- **Purpose**: Basic provider functionality, compatibility testing
- **Features**: Core job management, projects, ACL policies, credentials

### Enterprise
- **Version**: Rundeck Enterprise/Process Automation 5.17.0+
- **API Version**: 56+ (includes runner management)
- **Deployment**: User-managed instance
- **Purpose**: Comprehensive feature testing, Enterprise-specific functionality
- **Features**: Everything in OSS plus:
  - Execution lifecycle plugins
  - Project schedules
  - Runners (system and project)
  - Enterprise-specific features

## Running Tests

### Automated Test Suite

Run the full acceptance test suite (requires `RUNDECK_ENTERPRISE` flag for Enterprise features):

```bash
# From repository root
cd /Users/forrest/Documents/GitHub/terraform-provider-rundeck

# Run all tests
make testacc

# Run only job tests
make testacc TESTARGS='-run=TestAccJob'

# Run with Enterprise features
RUNDECK_ENTERPRISE=1 make testacc
```

### Manual Testing

Use the test configurations in this directory for manual validation:

1. **OSS**: Start Docker environment, run provider operations
2. **Enterprise**: Run comprehensive test script for full feature validation

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RUNDECK_URL` | Rundeck API endpoint | `http://localhost:4440` |
| `RUNDECK_AUTH_TOKEN` | API authentication token | See test configs |
| `RUNDECK_API_VERSION` | API version to use | `46` (OSS), `56` (Enterprise) |
| `RUNDECK_ENTERPRISE` | Enable Enterprise feature tests | `0` (disabled) |

## Test Scenarios Covered

### Core Functionality (OSS + Enterprise)
- ✅ Project creation and management
- ✅ Job definition with commands
- ✅ Node filters and dispatch settings
- ✅ Job options and parameters
- ✅ Notifications (webhooks, email)
- ✅ Scheduling and cron
- ✅ ACL policies
- ✅ Key storage (passwords, private keys, public keys)
- ✅ Import/export functionality

### Enterprise-Specific Features
- ✅ Execution lifecycle plugins
- ✅ Project schedules
- ✅ System runners
- ✅ Project runners
- ✅ UUID-based job references

### Recent Critical Fixes (Verified in Tests)
- ✅ **NodeFilters Structure**: Dispatch nested inside nodefilters (not at root)
- ✅ **Lifecycle Plugins**: Map structure (not array)
- ✅ **Plugin Ordering**: Consistent alphabetical sorting
- ✅ **UUID Job References**: Immutable job references

## Prerequisites

### For OSS Testing
- Docker and Docker Compose
- Go 1.24+ (for building provider)
- jq (for JSON parsing in test scripts)

### For Enterprise Testing
- Running Rundeck Enterprise instance (5.17.0+)
- Valid API token with appropriate permissions
- Go 1.24+ (for building provider)
- jq (for JSON parsing)

## Contributing

When adding new tests:

1. Add OSS tests to `test/oss/` for core functionality
2. Add Enterprise tests to `test/enterprise/` for commercial features
3. Update relevant README files
4. Ensure tests are documented and reproducible
5. Include cleanup steps

## Troubleshooting

### Common Issues

**OSS Docker won't start:**
- Check if port 4440 is already in use
- Ensure Docker has sufficient resources
- Check logs: `docker-compose logs -f`

**Enterprise tests failing:**
- Verify Rundeck Enterprise is running
- Check API token has correct permissions
- Confirm API version compatibility
- Review test logs in `/tmp/enterprise_test.log`

**Provider build failures:**
- Ensure Go 1.24+ is installed
- Run `go mod tidy`
- Clear Go cache: `go clean -cache`

## Additional Resources

- [Rundeck API Documentation](https://docs.rundeck.com/docs/api/)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [Provider Documentation](../website/docs/)
- [Main README](../README.md)

