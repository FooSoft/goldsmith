package depth

import (
	"path/filepath"
	"strings"

	"git.foosoft.net/alex/goldsmith"
)

type Depth struct {
	depth int
}

func New(depth int) *Depth {
	return &Depth{depth: depth}
}

func (*Depth) Name() string {
	return "depth"
}

func (self *Depth) Accept(file *goldsmith.File) bool {
	if self.depth == 0 {
		return true
	}

	if parts := strings.Split(file.Path(), string(filepath.Separator)); len(parts) <= self.depth {
		return true
	}

	return false
}
