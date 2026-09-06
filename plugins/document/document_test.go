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
	plugin := New(process)
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(plugin)
		},
	)
}

func TestNilProcessorReturnsError(t *testing.T) {
	if err := New(nil).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("nil document processor was accepted")
	}
}
