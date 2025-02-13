package layout

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/frontmatter"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(New().DefaultLayout("page"))
		},
	)
}
