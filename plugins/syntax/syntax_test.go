package syntax

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
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

func TestPlacementOptionsAndCacheConfiguration(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	input := `<html><body><code class="code-go">package main</code><code class="code-unknown-language">plain &lt;text&gt;</code></body></html>`
	if err := os.WriteFile(filepath.Join(source, "index.html"), []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, placement := range []Placement{PlaceInside, PlaceInline} {
		for range 2 { // Exercise fresh generation and cache retrieval.
			gs := goldsmith.Goldsmith{CacheDir: filepath.Join(root, "cache")}
			target := filepath.Join(root, "target")
			if errs := gs.Begin(source).Chain(New().Prefix("code-").Style("unknown-style").LineNumbers(true).Placement(placement)).End(target); len(errs) > 0 {
				t.Fatal(errs)
			}
			data, err := os.ReadFile(filepath.Join(target, "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			code := doc.Find(".code-go")
			if code.Find("span").Length() == 0 {
				t.Fatal("code was not highlighted")
			}
			if (code.Find("pre").Length() > 0) != (placement == PlaceInside) {
				t.Fatalf("unexpected preformatted block for placement %d: %s", placement, data)
			}
			if !strings.Contains(doc.Find(".code-unknown-language").Text(), "plain <text>") {
				t.Fatal("fallback lexer lost source text")
			}
			if !strings.Contains(code.Text(), "1") {
				t.Fatal("line numbers missing")
			}
		}
	}
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
