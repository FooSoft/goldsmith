package tags

import (
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
	"go.foosoft.net/goldsmith/plugins/markdown"
)

func Test(self *testing.T) {
	meta := map[string]interface{}{
		"Layout": "tag",
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(markdown.New()).
				Chain(New().IndexMeta(meta)).
				Chain(layout.New())
		},
	)
}

func TestTagsAcceptStringSlice(t *testing.T) {
	context := new(goldsmith.Context)
	file, err := context.CreateFileFromReader("page.html", strings.NewReader("content"))
	if err != nil {
		t.Fatal(err)
	}
	file.SetProp("Tags", []string{"first", "second"})

	plugin := New()
	if err := plugin.Process(context, file); err != nil {
		t.Fatal(err)
	}
	state, ok := file.Props()["TagState"].(*TagState)
	if !ok || len(state.CurrentTags) != 2 {
		t.Fatalf("tags were not populated: %#v", state)
	}
}

func TestSafeTagHasNoTrailingSeparator(t *testing.T) {
	if actual := safeTag(" tag! "); actual != "tag" {
		t.Fatalf("safeTag returned %q, want %q", actual, "tag")
	}
}
