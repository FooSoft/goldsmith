package pager

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
	"codeberg.org/foosoft/goldsmith/plugins/collection"
	"codeberg.org/foosoft/goldsmith/plugins/frontmatter"
	"codeberg.org/foosoft/goldsmith/plugins/layout"
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
