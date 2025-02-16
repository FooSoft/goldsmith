package tags

import (
	"testing"

	"git.sr.ht/~foosoft/goldsmith"
	"git.sr.ht/~foosoft/goldsmith/harness"
	"git.sr.ht/~foosoft/goldsmith/plugins/frontmatter"
	"git.sr.ht/~foosoft/goldsmith/plugins/layout"
	"git.sr.ht/~foosoft/goldsmith/plugins/markdown"
)

func Test(self *testing.T) {
	meta := map[string]interface{}{
		"Layout": "tag",
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(markdown.New()).
				Chain(New().IndexMeta(meta)).
				Chain(layout.New())
		},
	)
}
