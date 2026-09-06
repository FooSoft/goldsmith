package thumbnail

import (
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New())
		},
	)
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	if err := New().Size(0).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("zero thumbnail size was accepted")
	}
	if err := New().Namer(nil).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("nil thumbnail namer was accepted")
	}
	if err := New().Style(Style(99)).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("invalid thumbnail style was accepted")
	}
	if err := New().Style(Pad).Color(nil).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("nil thumbnail padding color was accepted")
	}
}
