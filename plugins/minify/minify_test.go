package minify

import (
	"os"
	"path/filepath"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New())
		},
	)
}

func TestUppercaseExtension(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "style.CSS"), []byte("body { color: red; }"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gs goldsmith.Goldsmith
	if errs := gs.Begin(sourceDir).Chain(New()).End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
	output, err := os.ReadFile(filepath.Join(targetDir, "style.CSS"))
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "body{color:red}" {
		t.Fatalf("unexpected minified output: %q", output)
	}
}
