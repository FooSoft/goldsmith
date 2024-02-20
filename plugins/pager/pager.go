// Package pager splits arrays of metadata into standalone pages. The plugin is
// initialized with a lister callback which is used to segment a slice of
// metadata contained within the provided file. While any large set of metadata
// can be split into segments, this plugin is particularly useful when working
// with the "collection" for paging blog entries, photos, etc.
package pager

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
)

// Namer callback function builds paths for file pages based on the original file path and page index.
type Namer func(path string, index int) string

// Lister callback function is used to return a metadata slice which should be paged across several files.
type Lister func(file goldsmith.File) interface{}

// Page represents information about a given metadata segment.
type Page struct {
	Index int
	Items interface{}
	File  goldsmith.File

	Next *Page
	Prev *Page
}

// Index contains paging information for the current file.
type Index struct {
	AllPages []Page
	CurrPage *Page
	Paged    bool
}

// Pager chainable context.
type Pager struct {
	pagerKey  string
	enableKey string

	namer         Namer
	lister        Lister
	inheritedKeys []string
	itemsPerPage  int

	files []goldsmith.File
	mutex sync.Mutex
}

// New creates a new instance of the Pager plugin.
func New(lister Lister) *Pager {
	namer := func(path string, index int) string {
		ext := filepath.Ext(path)
		body := strings.TrimSuffix(path, ext)
		return fmt.Sprintf("%s-%d%s", body, index, ext)
	}

	return &Pager{
		pagerKey:     "Pager",
		enableKey:    "PagerEnable",
		namer:        namer,
		lister:       lister,
		itemsPerPage: 10,
	}
}

// PagerKey sets the metadata key used to store paging information for each file (default: "Pager").
func (self *Pager) PagerKey(key string) *Pager {
	self.pagerKey = key
	return self
}

// EnableKey sets the metadata key used to determine if the current file should be paged (default: false).
func (self *Pager) EnableKey(key string) *Pager {
	self.enableKey = key
	return self
}

// ItemsPerPage sets the maximum number of items which can be included on a single page (default: 10).
func (self *Pager) ItemsPerPage(limit int) *Pager {
	self.itemsPerPage = limit
	return self
}

// Namer sets the callback used to build paths for file pages.
// Default naming inserts page number between file name and extension,
// for example "file.html" becomes "file-2.html".
func (self *Pager) Namer(namer Namer) *Pager {
	self.namer = namer
	return self
}

// InheritedKeys sets which metadata keys should be copied to generated pages from the original file (default: []).
// When no keys are provided, all metadata is copied from the original file to generated pages.
func (self *Pager) InheritedKeys(keys ...string) *Pager {
	self.inheritedKeys = keys
	return self
}

func (*Pager) Name() string {
	return "pager"
}

func (*Pager) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Pager) Process(context *goldsmith.Context, inputFile goldsmith.File) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	enabled, err := self.isEnabledForFile(inputFile)
	if err != nil {
		return err
	}

	if !enabled {
		self.files = append(self.files, inputFile)
		return nil
	}

	var buff bytes.Buffer
	if _, err := buff.ReadFrom(inputFile); err != nil {
		return err
	}

	values := self.lister(inputFile)
	valueCount, err := sliceLength(values)
	if err != nil {
		return err
	}

	pageCount := valueCount / self.itemsPerPage
	if valueCount%self.itemsPerPage > 0 {
		pageCount++
	}

	pages := make([]Page, pageCount, pageCount)
	for i := 0; i < pageCount; i++ {
		page := &pages[i]
		page.Index = i + 1

		if i > 0 {
			page.Prev = &pages[i-1]
		}
		if i+1 < pageCount {
			page.Next = &pages[i+1]
		}

		var (
			indexStart = i * self.itemsPerPage
			indexEnd   = indexStart + self.itemsPerPage
		)

		if indexEnd > valueCount {
			indexEnd = valueCount
		}

		if page.Items, err = sliceCrop(values, indexStart, indexEnd); err != nil {
			return err
		}

		if i == 0 {
			page.File = inputFile
		} else {
			page.File, err = context.CreateFileFromReader(self.namer(inputFile.Path(), page.Index), &buff)
			if err != nil {
				return err
			}

			if len(self.inheritedKeys) == 0 {
				page.File.CopyProps(inputFile)
			} else {
				for _, key := range self.inheritedKeys {
					if value, ok := inputFile.Prop(key); ok {
						page.File.SetProp(key, value)
					}
				}
			}
		}

		page.File.SetProp(self.pagerKey, Index{
			AllPages: pages,
			CurrPage: page,
			Paged:    pageCount > 1,
		})

		self.files = append(self.files, page.File)
	}

	return nil
}

func (self *Pager) Finalize(ctx *goldsmith.Context) error {
	for _, f := range self.files {
		ctx.DispatchFile(f)
	}

	return nil
}

func (self *Pager) isEnabledForFile(file goldsmith.File) (bool, error) {
	enableRaw, ok := file.Prop(self.enableKey)
	if !ok {
		return false, nil
	}

	enable, ok := enableRaw.(bool)
	if !ok {
		return false, errors.New("invalid pager enable setting")
	}

	return enable, nil
}

func sliceLength(slice interface{}) (int, error) {
	sliceVal := reflect.Indirect(reflect.ValueOf(slice))
	if sliceVal.Kind() != reflect.Slice {
		return -1, errors.New("invalid slice")
	}

	return sliceVal.Len(), nil
}

func sliceCrop(slice interface{}, start, end int) (interface{}, error) {
	sliceVal := reflect.Indirect(reflect.ValueOf(slice))
	if sliceVal.Kind() != reflect.Slice {
		return nil, errors.New("invalid slice")
	}
	if start < 0 || start > end {
		return nil, errors.New("invalid slice range")
	}

	sliceValNew := reflect.Indirect(reflect.New(sliceVal.Type()))
	for i := start; i < end; i++ {
		sliceElemNew := sliceVal.Index(i)
		sliceValNew.Set(reflect.Append(sliceValNew, sliceElemNew))
	}

	return sliceValNew.Interface(), nil
}
