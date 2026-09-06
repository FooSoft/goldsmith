package syntax

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
	"go.foosoft.net/goldsmith/plugins/markdown"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(markdown.New()).
				Chain(New()).
				Chain(layout.New())
		},
	)
}

func TestLanguageClassAfterOtherClasses(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	input := `<html><body><code class="extra language-go">package main</code></body></html>`
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	if errs := gs.Begin(sourceDir).Chain(New()).End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
	output, err := os.ReadFile(filepath.Join(targetDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "font-weight:bold\">package") {
		t.Fatalf("Go syntax was not highlighted: %s", output)
	}
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	if err := New().Prefix("").Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("empty syntax prefix was accepted")
	}
	if err := New().Placement(Placement(99)).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("invalid syntax placement was accepted")
	}
}
