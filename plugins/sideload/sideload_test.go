package sideload

import (
	"io/fs"
	"strings"
	"testing"

	"embed"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
)

//go:embed testdata/fs/source
var embedFs embed.FS

func TestFs(self *testing.T) {
	harness.ValidateCase(
		self,
		"fs",
		func(gs *goldsmith.Goldsmith) {
			subFs, err := fs.Sub(embedFs, "testdata/fs/source")
			if err != nil {
				panic(err)
			}

			gs.Chain(New().FileSystems(subFs))
		},
	)
}

func TestFile(self *testing.T) {
	harness.ValidateCase(
		self,
		"file",
		func(gs *goldsmith.Goldsmith) {
			file := File{
				Path:   "hello.txt",
				Reader: strings.NewReader("hello world"),
			}

			gs.Chain(New().Files(file))
		},
	)
}
