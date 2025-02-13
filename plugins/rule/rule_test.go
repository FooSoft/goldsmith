package rule

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/frontmatter"
	"codeberg.org/foosoft/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(New()).
				Chain(frontmatter.New()).
				Chain(layout.New().DefaultLayout("page"))
		},
	)
}
