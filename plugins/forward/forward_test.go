package forward

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/operator"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
	"git.foosoft.net/alex/goldsmith/harness"
	"git.foosoft.net/alex/goldsmith/plugins/layout"
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
