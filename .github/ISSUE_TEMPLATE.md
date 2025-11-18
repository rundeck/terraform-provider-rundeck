Hi there,

Thank you for opening an issue! Please use this template to help us resolve your issue quickly.

**For general usage questions**, please see the [Terraform Community](https://www.terraform.io/community.html) or [Rundeck Community Forums](https://community.pagerduty.com/forum/c/rundeck).

---

### Issue Type
<!-- Check one -->
- [ ] Bug Report
- [ ] Feature Request
- [ ] Question/Support

### Rundeck Version
**Required**: Provider versions 1.0.0+ require **Rundeck 5.0.0 or higher** (API v46+).

Run: `curl -H "Accept: application/json" http://your-rundeck/api/1/system/info | jq .system.rundeck.version`
- Rundeck Version: 
- API Version: 

**Note**: For Enterprise Runner resources, Rundeck 5.17.0+ (API v56) is required.

### Provider Version
Run: `terraform version`
- Terraform Version: 
- Provider Version: 

**Note**: If you're not on the latest version, please check the [CHANGELOG](https://github.com/rundeck/terraform-provider-rundeck/blob/main/CHANGELOG.md) to see if your issue is already fixed.

### Affected Resource(s)
<!-- List the resources, for example: -->
- `rundeck_job`
- `rundeck_project`

### Terraform Configuration
```hcl
# Please include the relevant parts of your Terraform configuration
# Remove sensitive data (tokens, passwords, etc.)
# For large configs, use a Gist: https://gist.github.com
```

### Terraform Plan/Apply Output
```
# If applicable, include the output from:
# terraform plan
# terraform apply
# Redact sensitive information
```

### Expected Behavior
What should have happened?

### Actual Behavior
What actually happened?

### Steps to Reproduce
1. `terraform init`
2. `terraform apply`
3. ...

### Known Limitations
Have you checked if this is a known limitation? See [TODO.md](https://github.com/rundeck/terraform-provider-rundeck/blob/main/TODO.md) for:
- Features not yet implemented
- Known issues and workarounds
- Planned enhancements

### Debug Output
**For bugs**, please provide debug output:
```bash
TF_LOG=DEBUG terraform apply 2>&1 | tee terraform-debug.log
```
Upload to a Gist: https://gist.github.com (do NOT paste directly)

Debug Output URL: 

### Additional Context
- Are you using Runbook Automation (commercial version) or Rundeck Open Source?
- Any non-standard Rundeck configuration?
- Did this work in a previous provider version?
- Related GitHub issues or PRs?

---

**Tip**: The more details you provide, the faster we can help! Include your `.tf` files and plan output if possible.
