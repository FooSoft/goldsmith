package forward

import (
	"testing"

	"git.sr.ht/~foosoft/goldsmith"
	"git.sr.ht/~foosoft/goldsmith/filters/operator"
	"git.sr.ht/~foosoft/goldsmith/filters/wildcard"
	"git.sr.ht/~foosoft/goldsmith/harness"
	"git.sr.ht/~foosoft/goldsmith/plugins/layout"
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
