package sideload

import (
	"io/fs"
	"testing"

	"embed"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/harness"
)

//go:embed testdata/source
var embedFs embed.FS

func Test(self *testing.T) {
	subFs, err := fs.Sub(embedFs, "testdata/source")
	if err != nil {
		panic(err)
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New().FileSystems(subFs))
		},
	)
}
