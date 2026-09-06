package breadcrumbs

import (
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	plugin := New()
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(plugin).
				Chain(layout.New())
		},
	)
}

func TestCycleReturnsError(t *testing.T) {
	context := new(goldsmith.Context)
	first, err := context.CreateFileFromReader("first.html", strings.NewReader("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := context.CreateFileFromReader("second.html", strings.NewReader("second"))
	if err != nil {
		t.Fatal(err)
	}
	first.SetProp("CrumbName", "first")
	first.SetProp("CrumbParent", "second")
	second.SetProp("CrumbName", "second")
	second.SetProp("CrumbParent", "first")

	plugin := New()
	if err := plugin.Process(context, first); err != nil {
		t.Fatal(err)
	}
	if err := plugin.Process(context, second); err != nil {
		t.Fatal(err)
	}
	if err := plugin.Finalize(context); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected a breadcrumb cycle error, got %v", err)
	}
}
