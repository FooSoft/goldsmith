package condition

import (
	"codeberg.org/foosoft/goldsmith"
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
