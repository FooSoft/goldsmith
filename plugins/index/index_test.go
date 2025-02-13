package index

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/filters/operator"
	"codeberg.org/foosoft/goldsmith/filters/wildcard"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/layout"
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
