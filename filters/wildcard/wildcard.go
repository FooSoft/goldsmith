package wildcard

import (
	"strings"

	"git.foosoft.net/alex/goldsmith"
	"github.com/bmatcuk/doublestar/v4"
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

func (self *Wildcard) Accept(file goldsmith.File) bool {
	filePath := self.adjustCase(file.Path())

	for _, wildcard := range self.wildcards {
		wildcard = self.adjustCase(wildcard)
		if matched, _ := doublestar.PathMatch(wildcard, filePath); matched {
			return true
		}
	}

	return false
}

func (self *Wildcard) adjustCase(str string) string {
	if self.caseSensitive {
		return str
	}

	return strings.ToLower(str)
}
