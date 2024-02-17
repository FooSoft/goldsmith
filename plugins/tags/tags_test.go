package tags

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/harness"
	"git.foosoft.net/alex/goldsmith/plugins/frontmatter"
	"git.foosoft.net/alex/goldsmith/plugins/layout"
	"git.foosoft.net/alex/goldsmith/plugins/markdown"
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
