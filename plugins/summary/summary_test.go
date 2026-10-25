package summary

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(New()).
				Chain(layout.New())
		},
	)
}

func TestCustomSelectorsAndMissingMatches(t *testing.T) {
	for _, tc := range []struct{ name, content, title, summary string }{
		{"first match", `<h2><em>First</em></h2><h2>Second</h2><article><b>Body</b></article>`, "<em>First</em>", "<b>Body</b>"},
		{"no matches", `<p>Other content</p>`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			source := filepath.Join(root, "source")
			if err := os.Mkdir(source, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "page.html"), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			checked := false
			verify := summaryVerifier(func(file *goldsmith.File) {
				checked = true
				for key, want := range map[string]string{"Heading": tc.title, "Preview": tc.summary} {
					value, exists := file.Prop(key)
					if want == "" {
						if exists {
							t.Errorf("missing selector set %s to %v", key, value)
						}
					} else if html, ok := value.(template.HTML); !ok || string(html) != want {
						t.Errorf("%s: got %#v, want trusted HTML %q", key, value, want)
					}
				}
			})
			var gs goldsmith.Goldsmith
			if errs := gs.Begin(source).Chain(New().TitleKey("Heading").SummaryKey("Preview").TitlePath("h2").SummaryPath("article")).Chain(verify).End(filepath.Join(root, "target")); len(errs) > 0 {
				t.Fatal(errs)
			}
			if !checked {
				t.Fatal("no page was verified")
			}
		})
	}
}

type summaryVerifier func(*goldsmith.File)

func (summaryVerifier) Name() string { return "summary-verifier" }

func (verify summaryVerifier) Process(_ *goldsmith.Context, file *goldsmith.File) error {
	verify(file)
	return nil
}
