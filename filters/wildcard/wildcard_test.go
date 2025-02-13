package wildcard

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
)

func Test(t *testing.T) {
	harness.Validate(
		t,
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New("**/*.txt", "*.md"))
		},
	)
}
