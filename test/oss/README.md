# Open Source Rundeck Test Environment

This directory contains a Docker-based test environment for Rundeck Open Source (Community Edition).

## Overview

- **Rundeck Version**: 5.17.0
- **Edition**: Open Source / Community
- **API Version**: 46
- **Deployment**: Docker Compose
- **Default Port**: 4440
- **Default Token**: `1d08bf61-f962-467f-8ba3-ab8a463b3467`

## Quick Start

```bash
# Start Rundeck OSS
cd test/oss
docker-compose up -d

# Wait for Rundeck to start (~30 seconds)
sleep 30

# Verify Rundeck is running
curl -H "X-Rundeck-Auth-Token: 1d08bf61-f962-467f-8ba3-ab8a463b3467" \
     http://localhost:4440/api/46/system/info | jq '.system.rundeck'

# Access UI
open http://localhost:4440
# Login: admin / admin
```

## Files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Docker Compose configuration for Rundeck container |
| `Dockerfile` | Custom Rundeck image with token configuration |
| `tokens.properties` | API token configuration (admin token) |
| `remco/` | Configuration management for tokens |

## Usage

### Starting the Environment

```bash
docker-compose up -d
```

### Checking Logs

```bash
docker-compose logs -f
```

### Stopping the Environment

```bash
docker-compose down
```

### Cleaning Up (Remove Data)

```bash
docker-compose down -v
```

## Testing with the Provider

### Build the Provider

```bash
# Navigate to the provider repository root
cd terraform-provider-rundeck  # or your clone location
go build -o terraform-provider-rundeck
```

### Setup Dev Overrides

Create `.terraformrc` in your test directory:

```hcl
provider_installation {
  dev_overrides {
    "terraform-providers/rundeck" = "/Users/forrest/Documents/GitHub/terraform-provider-rundeck"
  }
  direct {}
}
```

Export the config:

```bash
export TF_CLI_CONFIG_FILE="$PWD/.terraformrc"
```

### Example Terraform Configuration

```hcl
terraform {
  required_providers {
    rundeck = {
      source = "rundeck/rundeck"
    }
  }
}

provider "rundeck" {
  url         = "http://localhost:4440"
  auth_token  = "1d08bf61-f962-467f-8ba3-ab8a463b3467"
  api_version = "46"
}

resource "rundeck_project" "test" {
  name        = "test-project"
  description = "Test project for provider development"
  
  resource_model_source {
    type = "file"
    config = {
      format = "resourceyaml"
      file   = "/tmp/test-nodes.yaml"
    }
  }
}

resource "rundeck_job" "test" {
  project_name = rundeck_project.test.name
  name         = "test-job"
  description  = "Test job"
  
  node_filter_query = ".*"
  max_thread_count  = 5
  
  command {
    shell_command = "echo 'Hello from OSS Rundeck'"
  }
}
```

### Apply Configuration

```bash
terraform init  # Skip if using dev overrides
terraform plan
terraform apply
```

## API Access

### Base URL
```
http://localhost:4440
```

### API Token
```
1d08bf61-f962-467f-8ba3-ab8a463b3467
```

### Example API Calls

**Get System Info:**
```bash
curl -H "X-Rundeck-Auth-Token: 1d08bf61-f962-467f-8ba3-ab8a463b3467" \
     http://localhost:4440/api/46/system/info
```

**List Projects:**
```bash
curl -H "X-Rundeck-Auth-Token: 1d08bf61-f962-467f-8ba3-ab8a463b3467" \
     http://localhost:4440/api/46/projects
```

**Export Job:**
```bash
curl -H "X-Rundeck-Auth-Token: 1d08bf61-f962-467f-8ba3-ab8a463b3467" \
     "http://localhost:4440/api/46/project/PROJECT/jobs/export?idlist=JOB_UUID&format=json"
```

## Features Available in OSS

✅ **Available:**
- Projects
- Jobs with commands
- Node filters and dispatch
- Options and parameters
- Notifications (webhook, email)
- Scheduling
- ACL policies
- Key storage (passwords, SSH keys)
- Orchestrators
- Log filters

❌ **Enterprise Only:**
- Execution lifecycle plugins
- Project schedules  
- Runners (system/project)
- Advanced metrics
- Some enterprise plugins

## Troubleshooting

### Port Already in Use

If port 4440 is in use:

```bash
# Find the process
lsof -i :4440

# Stop other Rundeck instances
docker ps | grep rundeck
docker stop <container-id>
```

### Container Won't Start

```bash
# Check logs
docker-compose logs

# Rebuild image
docker-compose build --no-cache
docker-compose up -d
```

### Rundeck UI Not Accessible

```bash
# Check if container is running
docker-compose ps

# Check container logs
docker-compose logs rundeck

# Restart container
docker-compose restart
```

### API Calls Fail

```bash
# Verify Rundeck is ready
docker-compose logs | grep "Grails application running"

# Test connectivity
curl -v http://localhost:4440/api/46/system/info

# Check token
grep admin tokens.properties
```

## Development Workflow

1. **Start OSS Rundeck**: `docker-compose up -d`
2. **Make provider changes**: Edit provider code
3. **Rebuild provider**: `go build`
4. **Test changes**: Run terraform apply with test configuration
5. **Verify in UI**: Check http://localhost:4440
6. **Check API**: Verify resources via API calls
7. **Clean up**: `terraform destroy` or `docker-compose down`

## Notes

- **Data Persistence**: By default, data is stored in Docker volumes
- **Fresh Start**: Use `docker-compose down -v` to remove all data
- **Token**: The admin token is pre-configured in `tokens.properties`
- **No Enterprise Features**: This is OSS, so Enterprise-specific features won't work
- **Resource Files**: Jobs require node resource files (can be empty for local execution)

## Next Steps

After validating with OSS, test Enterprise-specific features using the Enterprise test environment in `test/enterprise/`.

