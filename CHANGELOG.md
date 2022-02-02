## 0.5.0 (Unreleased)

## O.4.3
* Updated Documentation

## 0.4.1/0.4.2

* Community improvements and updates to modern Terraform model.

## 0.4.0 (August 02, 2019)

### Added:
* Job reference node filter override [#27](https://github.com/terraform-providers/terraform-provider-rundeck/pull/27)

### Fixed:
* Handle empty value options gracefully [#28](https://github.com/terraform-providers/terraform-provider-rundeck/pull/28)

## 0.3.0 (June 13, 2019)

### Added:

* **Terraform 0.12** update Terraform SDK to 0.12.1 ([#25](https://github.com/terraform-providers/terraform-provider-rundeck/pull/25))
* resource/job: Add attribute `notification` ([#24](https://github.com/terraform-providers/terraform-provider-rundeck/pull/24))

## 0.2.1 (June 12, 2019)

### Added:
* Job Schedule Enabled argument
* Job Execution Enabled argument

### FIXED:
* Executions and schedules getting disabled due to missing defaults

## 0.2.0 (May 13, 2019)

### Added:
* ACL Policy resource
* API Version provider option

### FIXED:
* Idempotency issue when node filter not set on job

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
