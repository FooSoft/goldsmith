package tags

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/frontmatter"
	"codeberg.org/foosoft/goldsmith/plugins/layout"
	"codeberg.org/foosoft/goldsmith/plugins/markdown"
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
