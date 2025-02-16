package rule

import (
	"testing"

	"git.sr.ht/~foosoft/goldsmith"
	"git.sr.ht/~foosoft/goldsmith/harness"
	"git.sr.ht/~foosoft/goldsmith/plugins/frontmatter"
	"git.sr.ht/~foosoft/goldsmith/plugins/layout"
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
