package condition

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/harness"
)

func TestEnabled(self *testing.T) {
	harness.ValidateCase(
		self,
		"true",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New(true))
		},
	)
}

func TestDisabled(self *testing.T) {
	harness.ValidateCase(
		self,
		"false",
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New(false))
		},
	)
}
