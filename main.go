package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-rundeck/rundeck"
)

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary
	version string = "dev"
)

func main() {
	ctx := context.Background()

	// Convert SDK provider to protocol v5 server
	sdkProviderFunc := func() tfprotov5.ProviderServer {
		return schema.NewGRPCProviderServer(rundeck.Provider())
	}

	// Create Framework provider server
	frameworkProviderFunc := providerserver.NewProtocol5(rundeck.NewFrameworkProvider(version)())

	// Create a muxed server that serves both SDK and Framework providers
	muxServer, err := tf5muxserver.NewMuxServer(ctx, sdkProviderFunc, frameworkProviderFunc)

	if err != nil {
		log.Fatal(err)
	}

	// Serve the muxed provider
	var serveOpts []tf5server.ServeOpt

	err = tf5server.Serve(
		"registry.terraform.io/terraform-providers/rundeck",
		muxServer.ProviderServer,
		serveOpts...,
	)

	if err != nil {
		log.Fatal(err)
	}
}
