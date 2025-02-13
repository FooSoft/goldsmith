package thumbnail

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New())
		},
	)
}
