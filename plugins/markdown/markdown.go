// Package markdown renders Markdown documents to HTML with the "goldmark"
// processor. Note that unlike other static site generators, Markdown processing
// does not automatically parse frontmatter; you will need to use the "frontmatter"
// plugin to extract any metadata which may be present in your source content.
package markdown

import (
	"bytes"
	"errors"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// Markdown chainable context.
type Markdown struct {
	md goldmark.Markdown
}

// New creates a new instance of the Markdown plugin.
func New() *Markdown {
	return NewWithGoldmark(
		goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.Typographer),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	)
}

// NewWithGoldmark creates a Markdown plugin with a user-provided Goldmark instance.
func NewWithGoldmark(md goldmark.Markdown) *Markdown {
	return &Markdown{md}
}

func (*Markdown) Name() string {
	return "markdown"
}

func (self *Markdown) Initialize(context *goldsmith.Context) error {
	if self.md == nil {
		return errors.New("markdown processor cannot be nil")
	}

	context.Filter(wildcard.New("**/*.md", "**/*.markdown"))
	return nil
}

func (self *Markdown) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	outputPath := strings.TrimSuffix(inputFile.Path(), path.Ext(inputFile.Path())) + ".html"
	if outputFile := context.RetrieveCachedFile(outputPath, goldsmith.FileCacheKeys(inputFile)...); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	var dataIn bytes.Buffer
	if _, err := dataIn.ReadFrom(inputFile); err != nil {
		return err
	}

	var dataOut bytes.Buffer
	if err := self.md.Convert(dataIn.Bytes(), &dataOut); err != nil {
		return err
	}

	outputFile, err := context.CreateFileFromReader(outputPath, &dataOut)
	if err != nil {
		return err
	}

	outputFile.CopyProps(inputFile)
	context.DispatchAndCacheFile(outputFile, goldsmith.FileCacheKeys(inputFile)...)
	return nil
}
