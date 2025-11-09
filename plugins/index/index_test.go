package index

import (
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
