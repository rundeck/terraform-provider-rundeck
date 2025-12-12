# Rundeck Terraform Provider

## Overview

The Rundeck Terraform Provider enables infrastructure automation teams to manage Rundeck resources using HashiCorp Terraform. This provider is maintained by the community in the spirit of open source collaboration, with oversight from Rundeck/PagerDuty staff who review and approve contributions.

## Community Support

This provider is **community-supported**. While Rundeck/PagerDuty staff review and approve pull requests, new feature development is driven by community contributions. We welcome and encourage community involvement through:

- Bug reports and feature requests via GitHub Issues
- Code contributions via Pull Requests
- Documentation improvements
- Usage questions and discussions

## Documentation

- Provider Usage Documentation: [Terraform Registry](https://registry.terraform.io/providers/rundeck/rundeck/latest/docs)
- Community Discussion: [Google Groups](http://groups.google.com/group/terraform-tool)
- Chat: [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
- [Go](https://golang.org/doc/install) >= 1.24
- [Rundeck](https://www.rundeck.com/) >= 5.0.0 (API v46+)

## Building The Provider

1. Clone the repository
2. Enter the repository directory
3. Build the provider using the Go `install` command:

```sh
$ go install
```

## Development
### Contributing

If you wish to work on the provider, you'll first need [Go](https://www.golang.org) installed on your machine (see Requirements above).

To compile the provider:

Run `go install` - This will build the provider and put the provider binary in the `$GOPATH/bin` directory
To generate or update documentation, run `go generate`

### Testing

#### Running Acceptance Tests

To run the full suite of acceptance tests:

```sh
$ make testacc
```

**Note:** Acceptance tests create real resources and require a running Rundeck instance.

#### Local Testing with Docker

For local development, you can use the provided Docker setup:

```sh
$ cd test/oss
$ docker-compose up -d
$ cd ../..
$ TF_ACC=1 go test -v ./rundeck -timeout 120m
```

This will start a Rundeck instance at `http://localhost:4440` with default credentials (`admin`/`admin`).

#### Enterprise Feature Tests

Some tests require **Rundeck Enterprise** features and will fail on Rundeck Community Edition. These tests are automatically skipped by default to prevent CI/CD failures.

**Enterprise-only features tested:**
- **Runner resources** - System runners and project runners (5 tests)
- **Project schedules** - Job scheduling via project-level schedules (3 tests)
- **Execution lifecycle plugins**

**To run Enterprise tests locally:**

1. Ensure you have Rundeck Enterprise running (locally or remote)
2. Set the environment variable to enable Enterprise tests:

```sh
$ export RUNDECK_ENTERPRISE_TESTS=1
$ make testacc
```

**Project Schedule Tests (Additional Setup Required):**

The project schedule tests require manual setup because Rundeck requires schedules to exist in the project configuration before jobs can reference them. This cannot be automated through the Terraform provider.

To run project schedule tests:

1. Create a project named `terraform-schedules-test` in your Rundeck Enterprise instance
2. Add the following schedules to the project configuration:
   - `my-schedule` - Used by basic project schedule test
   - `schedule-1` and `schedule-2` - Used by multiple schedule test
   - `simple-schedule` - Used by schedule without options test
3. Set both environment variables:

```sh
$ export RUNDECK_ENTERPRISE_TESTS=1
$ export RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1
$ make testacc
```

Without `RUNDECK_PROJECT_SCHEDULES_CONFIGURED=1`, the project schedule tests will be skipped even when `RUNDECK_ENTERPRISE_TESTS=1` is set.

**In CI/CD pipelines:**

By default, Enterprise tests are skipped unless `RUNDECK_ENTERPRISE_TESTS=1` is set. To enable them in GitHub Actions or other CI:

```yaml
env:
  RUNDECK_ENTERPRISE_TESTS: 1
  RUNDECK_PROJECT_SCHEDULES_CONFIGURED: 1  # Only if project schedule tests should run
  RUNDECK_URL: https://your-enterprise-rundeck.example.com
  RUNDECK_AUTH_TOKEN: ${{ secrets.RUNDECK_TOKEN }}
```

#### Test Requirements

- **Go 1.24+** - Required for modern Terraform Plugin Framework
- **Rundeck 5.0.0+** - Minimum supported version (API v46)
- **Docker** - For local testing environment (optional but recommended)
- **Rundeck Enterprise** - Only if running Enterprise feature tests