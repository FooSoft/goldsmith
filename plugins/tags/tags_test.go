package tags

import (
	"fmt"
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
	plugin := New().IndexMeta(meta)

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(markdown.New()).
				Chain(plugin).
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

func TestCollidingTagPathsReturnError(t *testing.T) {
	context := new(goldsmith.Context)
	plugin := New().IndexMeta(map[string]interface{}{})
	for index, tag := range []string{"foo bar", "foo-bar"} {
		file, err := context.CreateFileFromReader(fmt.Sprintf("page-%d.html", index), strings.NewReader("content"))
		if err != nil {
			t.Fatal(err)
		}
		file.SetProp("Tags", []string{tag})
		if err := plugin.Process(context, file); err != nil {
			t.Fatal(err)
		}
	}

	if err := plugin.Finalize(context); err == nil || !strings.Contains(err.Error(), "same page path") {
		t.Fatalf("expected a tag path collision error, got %v", err)
	}
}
