package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/terraform-providers/terraform-provider-rundeck/rundeck"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"
)

func main() {
	err := providerserver.Serve(
		context.Background(),
		rundeck.NewFrameworkProvider(version),
		providerserver.ServeOpts{
			Address: "registry.terraform.io/terraform-providers/rundeck",
		},
	)

	if err != nil {
		log.Fatal(err)
	}
}
