module github.com/terraform-providers/terraform-provider-rundeck

go 1.14

require (
	github.com/Azure/go-autorest/autorest v0.9.2 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.1-0.20191028180845-3492b2aff503 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/hashicorp/go-getter v1.4.2-0.20200106182914-9813cbd4eb02 // indirect
	github.com/hashicorp/hcl/v2 v2.3.0 // indirect
	github.com/hashicorp/terraform-config-inspect v0.0.0-20191212124732-c6ae6269b9d7 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.9.0
	github.com/rundeck/go-rundeck/rundeck v0.0.0-20190510195016-2cf9670bbcc4
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible
