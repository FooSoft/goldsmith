package forward

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/filters/operator"
	"codeberg.org/foosoft/goldsmith/filters/wildcard"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/layout"
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
