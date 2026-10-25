package absolute

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
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

func TestURLResolutionAndCacheConfiguration(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err := os.MkdirAll(filepath.Join(source, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	input := `<a id="dir" href="../archive/">directory</a>
<a id="escaped" href="../a%2Fb/?q=a%20b#part">escaped</a>
<a id="fragment" href="#part">fragment</a>
<a id="query" href="?q=1">query</a>
<a id="root" href="/">root</a>
<a id="external" href="mailto:me@example.com">external</a>`
	if err := os.WriteFile(filepath.Join(source, "dir", "page #1.html"), []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"", "https://example.com/base%20dir/", "https://other.example/"} {
		gs := goldsmith.Goldsmith{CacheDir: filepath.Join(root, "cache")}
		target := filepath.Join(root, "target")
		plugin := New()
		if base != "" {
			plugin.BaseUrl(base)
		}
		if errs := gs.Begin(source).Chain(plugin).End(target); len(errs) > 0 {
			t.Fatal(errs)
		}
		data, err := os.ReadFile(filepath.Join(target, "dir", "page #1.html"))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
		if err != nil {
			t.Fatal(err)
		}
		prefix := strings.TrimRight(base, "/")
		for id, want := range map[string]string{
			"dir":      prefix + "/archive/",
			"escaped":  prefix + "/a%2Fb/?q=a%20b#part",
			"fragment": prefix + "/dir/page%20%231.html#part",
			"query":    prefix + "/dir/page%20%231.html?q=1",
			"root":     prefix + "/",
			"external": "mailto:me@example.com",
		} {
			if got := doc.Find("#"+id).AttrOr("href", ""); got != want {
				t.Errorf("base %q, %s: got %q, want %q", base, id, got, want)
			}
		}
	}
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
