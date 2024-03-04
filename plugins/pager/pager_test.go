package pager

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/harness"
	"git.foosoft.net/alex/goldsmith/plugins/collection"
	"git.foosoft.net/alex/goldsmith/plugins/frontmatter"
	"git.foosoft.net/alex/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	lister := func(file *goldsmith.File) interface{} {
		if groupsRaw, ok := file.Prop("Groups"); ok {
			if groups, ok := groupsRaw.(map[string][]*goldsmith.File); ok {
				if group, ok := groups["group"]; ok {
					return group
				}
			}
		}

		return nil
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(collection.New()).
				Chain(New(lister).ItemsPerPage(4)).
				Chain(layout.New())
		},
	)
}
