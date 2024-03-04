// Package minify removes superfluous data from a variety of web formats
// without modifying their behavior in web browsers. This plugin is useful for
// reducing bandwidth usage, but should be disabled for development. This can
// be accomplished by conditionally excluding it using the "condition" filter.
package minify

import (
	"bytes"
	"path/filepath"

	min "github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
)

// Minify chainable context.
type Minify struct{}

// New creates a new instance of the Minify plugin
func New() *Minify {
	return new(Minify)
}

func (*Minify) Name() string {
	return "minify"
}

func (*Minify) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.css", "**/*.html", "**/*.htm", "**/*.js", "**/*.svg", "**/*.json", "**/*.xml"))
	return nil
}

func (*Minify) Process(context *goldsmith.Context, inputFile goldsmith.File) error {
	if outputFile := context.RetrieveCachedFile(inputFile.Path(), inputFile); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	var (
		buff bytes.Buffer
		err  error
	)

	switch m := min.New(); filepath.Ext(inputFile.Path()) {
	case ".css":
		err = css.Minify(m, &buff, inputFile, nil)
	case ".html", ".htm":
		err = html.Minify(m, &buff, inputFile, nil)
	case ".js":
		err = js.Minify(m, &buff, inputFile, nil)
	case ".json":
		err = json.Minify(m, &buff, inputFile, nil)
	case ".svg":
		err = svg.Minify(m, &buff, inputFile, nil)
	case ".xml":
		err = xml.Minify(m, &buff, inputFile, nil)
	}

	if err != nil {
		return err
	}

	outputFile, err := context.CreateFileFromReader(inputFile.Path(), &buff)
	if err != nil {
		return err
	}

	outputFile.CopyProps(inputFile)
	context.DispatchAndCacheFile(outputFile, inputFile)
	return nil
}
