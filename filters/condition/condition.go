// Package condition filters files based on a boolean condition.
package condition

import (
	"go.foosoft.net/goldsmith"
)

type Condition struct {
	accept bool
}

func New(accept bool) *Condition {
	return &Condition{accept: accept}
}

func (*Condition) Name() string {
	return "condition"
}

func (self *Condition) Accept(file *goldsmith.File) bool {
	return self.accept
}
