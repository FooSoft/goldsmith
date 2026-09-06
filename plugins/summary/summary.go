// Package summary generates a summary and title for HTML files using CSS
// selectors. This plugin is useful when combined with other plugins such as
// "collection" to create blog post previews.
package summary

import (
	"html/template"

	"github.com/PuerkitoBio/goquery"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// Summary chainable context.
type Summary struct {
	titleKey    string
	summaryKey  string
	titlePath   string
	summaryPath string
}

// New creates a new instance of the Summary plugin.
func New() *Summary {
	return &Summary{
		titleKey:    "Title",
		summaryKey:  "Summary",
		titlePath:   "h1",
		summaryPath: "p",
	}
}

// TitleKey sets the metadata key used to store the file title (default: "Title").
func (self *Summary) TitleKey(key string) *Summary {
	self.titleKey = key
	return self
}

// SummaryKey sets the metadata key used to store the file summary (default: "Summary").
func (self *Summary) SummaryKey(key string) *Summary {
	self.summaryKey = key
	return self
}

// TitlePath sets CSS path used to retrieve the file title (default: "h1").
func (self *Summary) TitlePath(path string) *Summary {
	self.titlePath = path
	return self
}

// SummaryPath sets the CSS path used to retrieve the file summary (default: "p").
func (self *Summary) SummaryPath(path string) *Summary {
	self.summaryPath = path
	return self
}

func (*Summary) Name() string {
	return "summary"
}

func (*Summary) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Summary) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	doc, err := goquery.NewDocumentFromReader(inputFile)
	if err != nil {
		return err
	}

	if match := doc.Find(self.titlePath); match.Length() > 0 {
		if html, err := match.Html(); err == nil {
			inputFile.SetProp(self.titleKey, template.HTML(html))
		}
	}

	if match := doc.Find(self.summaryPath); match.Length() > 0 {
		if html, err := match.Html(); err == nil {
			inputFile.SetProp(self.summaryKey, template.HTML(html))
		}
	}

	context.DispatchFile(inputFile)
	return nil
}
