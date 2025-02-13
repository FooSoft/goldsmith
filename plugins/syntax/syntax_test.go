package syntax

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/frontmatter"
	"codeberg.org/foosoft/goldsmith/plugins/layout"
	"codeberg.org/foosoft/goldsmith/plugins/markdown"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(markdown.New()).
				Chain(New()).
				Chain(layout.New())
		},
	)
}
