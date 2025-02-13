// Package livejs injects code to reload the current page when it (or its
// dependencies) are modified. This plugin is helpful for authoring web content
// locally, but should be disabled for site deployment. This can be achieved by
// conditionally including it using the "condition" filter.
package livejs

import (
	"bytes"
	_ "embed"
	"fmt"

	"codeberg.org/foosoft/goldsmith"
	"codeberg.org/foosoft/goldsmith/filters/wildcard"
	"github.com/PuerkitoBio/goquery"
)

//go:embed js/live.js
var livejs string

// LiveJs chainable context.
type LiveJs struct {
	html string
}

// New creates a new instance of the LiveJs plugin.
func New() *LiveJs {
	return new(LiveJs)
}

func (*LiveJs) Name() string {
	return "livejs"
}

func (self *LiveJs) Initialize(context *goldsmith.Context) error {
	self.html = fmt.Sprintf("\n<!-- begin livejs code -->\n<script>\n%s\n</script>\n<!-- end livejs code -->\n", livejs)
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *LiveJs) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	if outputFile := context.RetrieveCachedFile(inputFile.Path(), inputFile); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(inputFile)
	if err != nil {
		return err
	}

	doc.Find("body").AppendHtml(self.html)

	html, err := doc.Html()
	if err != nil {
		return err
	}

	outputFile, err := context.CreateFileFromReader(inputFile.Path(), bytes.NewReader([]byte(html)))
	if err != nil {
		return err
	}

	outputFile.CopyProps(inputFile)
	context.DispatchAndCacheFile(outputFile)
	return nil
}
