package wildcard

import (
	"testing"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/harness"
)

func Test(t *testing.T) {
	harness.Validate(
		t,
		func(gs *goldsmith.Goldsmith) {
			gs.FilterPush(New("**/*.txt", "*.md"))
		},
	)
}
