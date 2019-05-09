module github.com/terraform-providers/terraform-provider-rundeck

require (
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/hashicorp/terraform v0.11.14-0.20190429193930-20e17ec86f19
	github.com/rundeck/go-rundeck/rundeck v0.0.0-20190509013643-e000811d1074
	golang.org/x/text v0.3.2 // indirect
)

replace github.com/rundeck/go-rundeck/rundeck v0.0.0-20190508052939-539d17f6912f => ../go-rundeck/rundeck
