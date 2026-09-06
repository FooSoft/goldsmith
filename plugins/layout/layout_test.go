package layout

import (
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
)

func Test(self *testing.T) {
	plugin := New().DefaultLayout("page")
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(plugin)
		},
	)
}

func TestUppercaseExtensions(t *testing.T) {
	context := new(goldsmith.Context)
	plugin := New().DefaultLayout("page")
	if err := plugin.Initialize(context); err != nil {
		t.Fatal(err)
	}

	page, err := context.CreateFileFromReader("PAGE.HTML", strings.NewReader("content"))
	if err != nil {
		t.Fatal(err)
	}
	templateFile, err := context.CreateFileFromReader("PAGE.GOHTML", strings.NewReader(`{{define "page"}}{{.}}{{end}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := plugin.Process(context, page); err != nil {
		t.Fatal(err)
	}
	if err := plugin.Process(context, templateFile); err != nil {
		t.Fatal(err)
	}
	if len(plugin.inputFiles) != 1 || len(plugin.templateFiles) != 1 {
		t.Fatalf("uppercase extensions were not classified: %d inputs, %d templates", len(plugin.inputFiles), len(plugin.templateFiles))
	}
}
