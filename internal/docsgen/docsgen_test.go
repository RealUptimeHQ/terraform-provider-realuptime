package docsgen

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RealUptimeHQ/terraform-provider-realuptime/internal/provider"
)

// TestDocsAreCurrent fails when docs/ does not match what the schemas
// render: run `go run ./cmd/gendocs` from the package root and commit.
func TestDocsAreCurrent(t *testing.T) {
	root := filepath.Join("..", "..")
	pages, err := Generate(context.Background(), provider.New("dev"), filepath.Join(root, "examples"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"index.md", "resources/check.md", "resources/status_page.md", "resources/status_page_component.md", "resources/notification_channel.md", "data-sources/status_page.md"} {
		if _, ok := pages[want]; !ok {
			t.Fatalf("generator produced no %s", want)
		}
	}
	for rel, want := range pages {
		got, err := os.ReadFile(filepath.Join(root, "docs", rel))
		if err != nil {
			t.Fatalf("docs/%s: %v (run go run ./cmd/gendocs)", rel, err)
		}
		if string(got) != want {
			t.Fatalf("docs/%s is stale: run go run ./cmd/gendocs and commit", rel)
		}
	}
}

func TestEveryAttributeIsDocumentedWithAnExample(t *testing.T) {
	pages, err := Generate(context.Background(), provider.New("dev"), filepath.Join("..", "..", "examples"))
	if err != nil {
		t.Fatal(err)
	}
	for rel, page := range pages {
		if !strings.Contains(page, "## Example Usage") {
			t.Fatalf("%s has no example", rel)
		}
		if strings.Contains(page, "- `") && strings.Contains(page, "` () ") {
			t.Fatalf("%s has an attribute with an empty description", rel)
		}
	}
	if !strings.Contains(pages["resources/notification_channel.md"], "(Sensitive.)") {
		t.Fatal("config must be marked sensitive in the docs")
	}
	if !strings.Contains(pages["resources/status_page_component.md"], "## Import") {
		t.Fatal("component import syntax must be documented")
	}
}
