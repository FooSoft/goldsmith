package absolute

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New().BaseUrl("https://foosoft.net"))
		},
	)
}

func TestProtocolRelativeURLIsPreserved(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	input := `<html><body><script src="//cdn.example.com/file.js"></script></body></html>`
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	if errs := gs.Begin(sourceDir).Chain(New().BaseUrl("https://example.com/base")).End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
	output, err := os.ReadFile(filepath.Join(targetDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `src="//cdn.example.com/file.js"`) {
		t.Fatalf("protocol-relative URL was changed: %s", output)
	}
}

func TestInvalidBaseURLReturnsError(t *testing.T) {
	err := New().BaseUrl("https://example.com/%zz").Initialize(new(goldsmith.Context))
	if err == nil || !strings.Contains(err.Error(), "invalid base URL") {
		t.Fatalf("expected an invalid base URL error, got %v", err)
	}
}
