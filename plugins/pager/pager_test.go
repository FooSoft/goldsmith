package pager

import (
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/collection"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
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
