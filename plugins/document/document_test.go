package document

import (
	"testing"

	"github.com/PuerkitoBio/goquery"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

func process(file *goldsmith.File, doc *goquery.Document) error {
	doc.Find("h1").SetAttr("style", "color: red;")
	return nil
}

func Test(self *testing.T) {
	type Processor func(*goquery.Document) error

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New(process))
		},
	)
}

func TestNilProcessorReturnsError(t *testing.T) {
	if err := New(nil).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("nil document processor was accepted")
	}
}
