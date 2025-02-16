// Package layout transforms content by applying Go templates to the content
// and metadata of HTML files. This plugin can be easily used with the
// "frontmatter" and "markdown" plugins to generate easy to maintain
// content-driven websites that are completely decoupled from layout details.
package layout

import (
	"bytes"
	"html/template"
	"sync"

	"git.sr.ht/~foosoft/goldsmith"
	"git.sr.ht/~foosoft/goldsmith/filters/wildcard"
)

// Layout chainable context.
type Layout struct {
	layoutKey     string
	contentKey    string
	defaultLayout *string
	helpers       template.FuncMap

	inputFiles    []*goldsmith.File
	templateFiles []*goldsmith.File
	mutex         sync.Mutex

	template *template.Template
}

// New creates a new instance of the Layout plugin.
func New() *Layout {
	return &Layout{
		layoutKey:  "Layout",
		contentKey: "Content",
		helpers:    nil,
	}
}

// LayoutKey sets the metadata key used to access the layout identifier (default: "Layout").
func (self *Layout) LayoutKey(key string) *Layout {
	self.layoutKey = key
	return self
}

// DefaultLayout sets the name of the layout to use if none is specified.
func (self *Layout) DefaultLayout(name string) *Layout {
	self.defaultLayout = &name
	return self
}

// ContentKey sets the metadata key used to access the source content (default: "Content").
func (self *Layout) ContentKey(key string) *Layout {
	self.contentKey = key
	return self
}

// Helpers sets the function map used to lookup template helper functions.
func (self *Layout) Helpers(helpers template.FuncMap) *Layout {
	self.helpers = helpers
	return self
}

func (*Layout) Name() string {
	return "layout"
}

func (self *Layout) Initialize(context *goldsmith.Context) error {
	self.template = template.New("").Funcs(self.helpers)
	context.Filter(wildcard.New("**/*.html", "**/*.htm", "**/*.tmpl", "**/*.gohtml"))
	return nil
}

func (self *Layout) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	switch inputFile.Ext() {
	case ".html", ".htm":
		if _, ok := self.getFileLayout(inputFile); ok {
			var buff bytes.Buffer
			if _, err := inputFile.WriteTo(&buff); err != nil {
				return err
			}

			inputFile.SetProp(self.contentKey, template.HTML(buff.Bytes()))
			self.inputFiles = append(self.inputFiles, inputFile)
		} else {
			context.DispatchFile(inputFile)
		}
	case ".tmpl", ".gohtml":
		self.templateFiles = append(self.templateFiles, inputFile)
	}

	return nil
}

func (self *Layout) Finalize(context *goldsmith.Context) error {
	for _, templateFile := range self.templateFiles {
		var buff bytes.Buffer
		if _, err := templateFile.WriteTo(&buff); err != nil {
			return err
		}

		if _, err := self.template.Parse(string(buff.Bytes())); err != nil {
			return err
		}
	}

	for _, inputFile := range self.inputFiles {
		if name, ok := self.getFileLayout(inputFile); ok {
			var buff bytes.Buffer
			if err := self.template.ExecuteTemplate(&buff, name, inputFile); err != nil {
				return err
			}

			outputFile, err := context.CreateFileFromReader(inputFile.Path(), &buff)
			if err != nil {
				return err
			}

			outputFile.CopyProps(inputFile)
			context.DispatchFile(outputFile)
		} else {
			context.DispatchFile(inputFile)
		}
	}

	return nil
}

func (self *Layout) getFileLayout(file *goldsmith.File) (string, bool) {
	if name, ok := file.Props()[self.layoutKey].(string); ok {
		return name, true
	}

	if self.defaultLayout != nil {
		return *self.defaultLayout, true
	}

	return "", false
}
