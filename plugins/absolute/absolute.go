// Package absolute converts relative file references in HTML documents to
// absolute paths. This is useful when working with plugins like "layout" and
// "collection", which can render a page’s content from the context of a
// different directory (imagine an index page showing inline previews of blog
// posts). This plugin makes it easy to fix incorrect relative file references
// by making sure all paths are absolute before content is featured on other
// sections of your site.
package absolute

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// Absolute chainable plugin context.
type Absolute struct {
	attributes []string
	baseUrl    *url.URL
	baseUrlErr error
}

// New creates absolute new instance of the Absolute plugin.
func New() *Absolute {
	return &Absolute{attributes: []string{"href", "src"}}
}

// Attributes sets the attributes which are scanned for relative URLs (default: "href", "src").
func (self *Absolute) Attributes(attributes ...string) *Absolute {
	self.attributes = attributes
	return self
}

// BaseUrl sets the base URL which is prepended to absolute-converted relative paths.
func (self *Absolute) BaseUrl(baseUrl string) *Absolute {
	self.baseUrl, self.baseUrlErr = url.Parse(baseUrl)
	return self
}

func (*Absolute) Name() string {
	return "absolute"
}

func (self *Absolute) Initialize(context *goldsmith.Context) error {
	if self.baseUrlErr != nil {
		return fmt.Errorf("invalid base URL: %w", self.baseUrlErr)
	}

	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	context.CacheKey(fmt.Sprintf("%q/%v", self.attributes, self.baseUrl))
	return nil
}

func (self *Absolute) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	if outputFile := context.RetrieveCachedFile(inputFile.Path(), goldsmith.FileCacheKeys(inputFile)...); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	fileUrl := &url.URL{Path: "/" + inputFile.Path()}

	doc, err := goquery.NewDocumentFromReader(inputFile)
	if err != nil {
		return err
	}

	doc.Find("*").Each(func(index int, selection *goquery.Selection) {
		for _, attribute := range self.attributes {
			value, exists := selection.Attr(attribute)
			if !exists {
				continue
			}

			currUrl, err := url.Parse(value)
			if err != nil {
				continue
			}

			if currUrl.IsAbs() || currUrl.Host != "" {
				continue
			}

			currUrl = fileUrl.ResolveReference(currUrl)
			if self.baseUrl != nil {
				rebasedUrl := *self.baseUrl
				rebasedUrl.Path = strings.TrimRight(self.baseUrl.Path, "/") + currUrl.Path
				rebasedUrl.RawPath = strings.TrimRight(self.baseUrl.EscapedPath(), "/") + currUrl.EscapedPath()
				rebasedUrl.Fragment = currUrl.Fragment
				rebasedUrl.RawFragment = currUrl.RawFragment
				rebasedUrl.RawQuery = currUrl.RawQuery
				currUrl = &rebasedUrl
			}

			selection.SetAttr(attribute, currUrl.String())
		}
	})

	html, err := doc.Html()
	if err != nil {
		return err
	}

	outputFile, err := context.CreateFileFromReader(inputFile.Path(), bytes.NewReader([]byte(html)))
	if err != nil {
		return err
	}

	outputFile.CopyProps(inputFile)
	context.DispatchAndCacheFile(outputFile, goldsmith.FileCacheKeys(inputFile)...)
	return nil
}
