package forward

import (
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/operator"
	"go.foosoft.net/goldsmith/filters/wildcard"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	meta := map[string]interface{}{
		"Layout": "forward",
	}

	pathMap := map[string]string{
		"file_old.html": "file_new.html",
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				FilterPush(operator.Not(wildcard.New("*.gohtml"))).
				Chain(New(meta).PathMap(pathMap)).
				FilterPop().
				Chain(layout.New())
		},
	)
}

func TestAddPathMappingAfterNilMap(t *testing.T) {
	plugin := New(nil).PathMap(nil).AddPathMapping("old.html", "new.html")
	if plugin.pathMap["old.html"] != "new.html" {
		t.Fatalf("path mapping was not added: %v", plugin.pathMap)
	}
}
