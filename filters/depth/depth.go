// Package depth filters files based on their directory depth.
package depth

import (
	"strings"

	"go.foosoft.net/goldsmith"
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

	return strings.Count(file.Path(), "/") < self.depth
}
