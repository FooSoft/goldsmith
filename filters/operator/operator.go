// Package operator combines filters with logical operators.
package operator

import (
	"go.foosoft.net/goldsmith"
)

type Operator interface {
	goldsmith.Filter
}

func And(filters ...goldsmith.Filter) Operator {
	return &operatorAnd{filters}
}

type operatorAnd struct {
	filters []goldsmith.Filter
}

func (*operatorAnd) Name() string {
	return "operator"
}

func (self *operatorAnd) Accept(file *goldsmith.File) bool {
	for _, filter := range self.filters {
		if !filter.Accept(file) {
			return false
		}
	}

	return true
}

func Not(self goldsmith.Filter) Operator {
	return &operatorNot{self}
}

type operatorNot struct {
	filter goldsmith.Filter
}

func (*operatorNot) Name() string {
	return "operator"
}

func (self *operatorNot) Accept(file *goldsmith.File) bool {
	return !self.filter.Accept(file)
}

func Or(self ...goldsmith.Filter) Operator {
	return &operatorOr{self}
}

type operatorOr struct {
	filters []goldsmith.Filter
}

func (*operatorOr) Name() string {
	return "operator"
}

func (self *operatorOr) Accept(file *goldsmith.File) bool {
	for _, filter := range self.filters {
		if filter.Accept(file) {
			return true
		}
	}

	return false
}
