package index

import (
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/operator"
	"go.foosoft.net/goldsmith/filters/wildcard"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	props := map[string]interface{}{
		"Layout": "index",
	}
	plugin := New(props)

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				FilterPush(operator.Not(wildcard.New("*.gohtml"))).
				Chain(plugin).
				FilterPop().
				Chain(layout.New())
		},
	)
}

func TestExistingIndexDoesNotListItself(t *testing.T) {
	context := new(goldsmith.Context)
	file, err := context.CreateFileFromReader("index.html", strings.NewReader("content"))
	if err != nil {
		t.Fatal(err)
	}

	plugin := New(nil)
	if err := plugin.Process(context, file); err != nil {
		t.Fatal(err)
	}
	if entries := plugin.dirLists["."].entries; len(entries) != 0 {
		t.Fatalf("index listed itself: %#v", entries)
	}
}
