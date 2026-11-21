// Package syntax generates syntax highlighting for preformatted code blocks
// using the "chroma" processor. All of the themes and styles from the
// processor are directly exposed through the plugin interface.
package syntax

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

type Placement int

const (
	PlaceInside Placement = iota
	PlaceInline
)

// Syntax chainable context.
type Syntax struct {
	style     string
	numbers   bool
	prefix    string
	placement Placement
}

// New creates a new instance of the Syntax plugin.
func New() *Syntax {
	return &Syntax{
		style:     "github",
		numbers:   false,
		prefix:    "language-",
		placement: PlaceInside,
	}
}

// Style sets the color scheme used for syntax highlighting (default: "github").
// Additional styles can be found at: https://github.com/alecthomas/chroma/tree/master/styles.
func (self *Syntax) Style(style string) *Syntax {
	self.style = style
	return self
}

// LineNumbers sets the visibility of a line number gutter next to the code (default: false).
func (self *Syntax) LineNumbers(numbers bool) *Syntax {
	self.numbers = numbers
	return self
}

// Prefix sets the CSS class name prefix for code language identification (default: "language-").
func (self *Syntax) Prefix(prefix string) *Syntax {
	self.prefix = prefix
	return self
}

// Placement determines if code should replace the containing block or be placed inside of it (default: "PlaceInside").
func (self *Syntax) Placement(placement Placement) *Syntax {
	self.placement = placement
	return self
}

func (*Syntax) Name() string {
	return "syntax"
}

func (self *Syntax) Initialize(context *goldsmith.Context) error {
	if self.prefix == "" {
		return errors.New("syntax class prefix cannot be empty")
	}
	if self.placement != PlaceInside && self.placement != PlaceInline {
		return errors.New("unsupported syntax placement")
	}

	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	context.CacheKey(fmt.Sprintf("%q/%t/%q/%d", self.style, self.numbers, self.prefix, self.placement))
	return nil
}

func (self *Syntax) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	if outputFile := context.RetrieveCachedFile(inputFile.Path(), goldsmith.FileCacheKeys(inputFile)...); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(inputFile)
	if err != nil {
		return err
	}

	var errs []error
	doc.Find("[class]").Each(func(i int, sel *goquery.Selection) {
		class := sel.AttrOr("class", "")
		var language string
		for _, name := range strings.Fields(class) {
			if strings.HasPrefix(name, self.prefix) {
				language = strings.TrimPrefix(name, self.prefix)
				break
			}
		}
		if language == "" {
			return
		}
		lexer := lexers.Get(language)
		if lexer == nil {
			lexer = lexers.Fallback
		}

		iterator, err := lexer.Tokenise(nil, sel.Text())
		if err != nil {
			errs = append(errs, err)
			return
		}

		style := styles.Get(self.style)
		if style == nil {
			style = styles.Fallback
		}

		var options []html.Option
		if self.numbers {
			options = append(options, html.WithLineNumbers(true))
		}

		formatter := html.New(options...)
		var buff bytes.Buffer
		if err := formatter.Format(&buff, style, iterator); err != nil {
			errs = append(errs, err)
			return
		}

		switch self.placement {
		case PlaceInside:
			sel.SetHtml(buff.String())
		case PlaceInline:
			docCode, err := goquery.NewDocumentFromReader(&buff)
			if err != nil {
				errs = append(errs, err)
				return
			}

			selPre := docCode.Find("pre")
			if style, exists := selPre.Attr("style"); exists {
				sel.SetAttr("style", style)
			}

			htmlPre, err := selPre.Html()
			if err != nil {
				errs = append(errs, err)
				return
			}
			sel.SetHtml(htmlPre)
		}
	})

	if len(errs) > 0 {
		return errs[0]
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
	context.DispatchAndCacheFile(outputFile, goldsmith.FileCacheKeys(inputFile)...)
	return nil
}
