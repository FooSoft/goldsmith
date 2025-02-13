package document

import (
	"testing"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/harness"
	"github.com/PuerkitoBio/goquery"
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
