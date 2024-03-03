// Package document enables simple HTML modification via callback via
// "goquery", an API similar to "jquery". This plugin is particularly useful
// adding classes to elements and performing other cleanup tasks which are too
// case-specific to warrant the creation of a new plugin.
package document

import (
	"bytes"
	"sync"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
	"github.com/PuerkitoBio/goquery"
)

// Processor callback function to modify documents.
type Processor func(goldsmith.File, *goquery.Document) error

// Document plugin context.
type Document struct {
	callback Processor
	files    []goldsmith.File
	mutex    sync.Mutex
}

// New creates a new instance of the Document plugin.
func New(callback Processor) *Document {
	return &Document{callback: callback}
}

func (*Document) Name() string {
	return "document"
}

func (*Document) Initialize(context goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Document) Process(context goldsmith.Context, inputFile goldsmith.File) error {
	doc, err := goquery.NewDocumentFromReader(inputFile)
	if err != nil {
		return err
	}

	if err := self.callback(inputFile, doc); err != nil {
		return err
	}

	html, err := doc.Html()
	if err != nil {
		return err
	}

	outputFile, err := context.CreateFileFromReader(inputFile.Path(), bytes.NewReader([]byte(html)))
	outputFile.CopyProps(inputFile)

	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.files = append(self.files, outputFile)
	return nil
}

func (self *Document) Finalize(context goldsmith.Context) error {
	for _, file := range self.files {
		context.DispatchFile(file)
	}

	return nil
}
