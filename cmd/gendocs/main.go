// gendocs regenerates docs/ from the provider schemas and examples/.
// Run from the package root: go run ./cmd/gendocs
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/docsgen"
	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/provider"
)

func main() {
	pages, err := docsgen.Generate(context.Background(), provider.New("dev"), "examples")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := docsgen.WriteAll("docs", pages); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for rel := range pages {
		fmt.Println("wrote docs/" + rel)
	}
}
