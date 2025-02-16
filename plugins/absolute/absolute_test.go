package absolute

import (
	"testing"

	"git.sr.ht/~foosoft/goldsmith"
	"git.sr.ht/~foosoft/goldsmith/harness"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New().BaseUrl("https://foosoft.net"))
		},
	)
}
