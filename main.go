// terraform-provider-realuptime: manage RealUptime checks, status pages,
// status page components and notification channels from Terraform.
//
// Source: packages/terraform-provider-realuptime in the RealUptime monorepo,
// mirrored to github.com/RealUptimeHQ/terraform-provider-realuptime for the
// Terraform Registry (scripts/publish-terraform-provider.mjs).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/provider"
)

// version is set by GoReleaser via -ldflags at release time; "dev" locally.
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run with support for debuggers like delve")
	flag.Parse()

	err := providerserver.Serve(context.Background(), provider.New(version), providerserver.ServeOpts{
		Address: "registry.terraform.io/RealUptimeHQ/realuptime",
		Debug:   debug,
	})
	if err != nil {
		log.Fatal(err)
	}
}
