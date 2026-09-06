// Package document enables simple HTML modification via callback via
// "goquery", an API similar to "jquery". This plugin is particularly useful
// adding classes to elements and performing other cleanup tasks which are too
// case-specific to warrant the creation of a new plugin.
package document

import (
	"bytes"
	"errors"
	"sort"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// Processor callback function to modify documents.
type Processor func(*goldsmith.File, *goquery.Document) error

// Document plugin context.
type Document struct {
	callback Processor
	files    []*goldsmith.File
	mutex    sync.Mutex
}

// New creates a new instance of the Document plugin.
func New(callback Processor) *Document {
	return &Document{callback: callback}
}

func (*Document) Name() string {
	return "document"
}

func (self *Document) Initialize(context *goldsmith.Context) error {
	if self.callback == nil {
		return errors.New("document processor cannot be nil")
	}

	self.files = nil
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Document) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
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
	if err != nil {
		return err
	}
	outputFile.CopyProps(inputFile)

	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.files = append(self.files, outputFile)
	return nil
}

func (self *Document) Finalize(context *goldsmith.Context) error {
	sort.Slice(self.files, func(i, j int) bool {
		return self.files[i].Path() < self.files[j].Path()
	})
	for _, file := range self.files {
		context.DispatchFile(file)
	}

	return nil
}
