package collection

import (
	"strings"
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

func TestCollectionAcceptsInterfaceSlice(t *testing.T) {
	context := new(goldsmith.Context)
	file, err := context.CreateFileFromReader("page.html", strings.NewReader("content"))
	if err != nil {
		t.Fatal(err)
	}
	file.SetProp("Collection", []interface{}{"first", "second"})

	plugin := New()
	if err := plugin.Process(context, file); err != nil {
		t.Fatal(err)
	}
	groups, ok := file.Props()["Groups"].(map[string][]*goldsmith.File)
	if !ok || len(groups["first"]) != 1 || len(groups["second"]) != 1 {
		t.Fatalf("collection groups were not populated: %#v", groups)
	}
}
