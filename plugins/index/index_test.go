package index

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/operator"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
	"git.foosoft.net/alex/goldsmith/harness"
	"git.foosoft.net/alex/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	props := map[string]interface{}{
		"Layout": "index",
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				FilterPush(operator.Not(wildcard.New("*.gohtml"))).
				Chain(New(props)).
				FilterPop().
				Chain(layout.New())
		},
	)
}
