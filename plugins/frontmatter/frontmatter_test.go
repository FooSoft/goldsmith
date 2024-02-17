package frontmatter

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/harness"
	"git.foosoft.net/alex/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(New()).
				Chain(layout.New())
		},
	)
}
