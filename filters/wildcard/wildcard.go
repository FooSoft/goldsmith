// Package wildcard filters files using path patterns.
package wildcard

import (
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"go.foosoft.net/goldsmith"
)

type Wildcard struct {
	wildcards     []string
	caseSensitive bool
}

func New(wildcards ...string) *Wildcard {
	return &Wildcard{wildcards: wildcards}
}

func (self *Wildcard) CaseSensitive(caseSensitive bool) *Wildcard {
	self.caseSensitive = caseSensitive
	return self
}

func (*Wildcard) Name() string {
	return "wildcard"
}

func (self *Wildcard) Accept(file *goldsmith.File) bool {
	filePath := self.adjustPath(file.Path())

	for _, wildcard := range self.wildcards {
		wildcard = self.adjustPath(wildcard)
		if matched, _ := doublestar.PathMatch(wildcard, filePath); matched {
			return true
		}
	}

	return false
}

func (self *Wildcard) adjustPath(str string) string {
	str = filepath.FromSlash(str)

	if self.caseSensitive {
		return str
	}

	return strings.ToLower(str)
}
