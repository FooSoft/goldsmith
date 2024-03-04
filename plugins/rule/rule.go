package rule

import (
	"io"
	"path/filepath"
	"sync"

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/operator"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
	"github.com/BurntSushi/toml"
)

type rule struct {
	Accept  []string
	Reject  []string
	baseDir string
}

type ruleApply struct {
	rule
	Props goldsmith.FileProps
}

type ruleDrop struct {
	rule
}

func (self *rule) accept(inputFile *goldsmith.File) bool {
	if !wildcard.New(filepath.Join(self.baseDir, "**")).Accept(inputFile) {
		return false
	}

	var acceptPaths []string
	for _, path := range self.Accept {
		acceptPaths = append(acceptPaths, filepath.Join(self.baseDir, path))
	}

	if wildcard.New(acceptPaths...).Accept(inputFile) {
		return true
	}

	var rejectPaths []string
	for _, path := range self.Reject {
		rejectPaths = append(rejectPaths, filepath.Join(self.baseDir, path))
	}

	if len(rejectPaths) == 0 {
		return false
	}

	return operator.Not(wildcard.New(rejectPaths...)).Accept(inputFile)
}

func (self *ruleApply) apply(inputFile *goldsmith.File) {
	if self.accept(inputFile) {
		for name, value := range self.Props {
			inputFile.SetProp(name, value)
		}
	}
}

func (self *ruleDrop) drop(inputFile *goldsmith.File) bool {
	return self.accept(inputFile)
}

type ruleSet struct {
	Apply []*ruleApply
	Drop  []*ruleDrop
}

func newRuleSet(inputFile *goldsmith.File) (*ruleSet, error) {
	data, err := io.ReadAll(inputFile)
	if err != nil {
		return nil, err
	}

	var ruleSet ruleSet
	if err := toml.Unmarshal(data, &ruleSet); err != nil {
		return nil, err
	}

	for _, rule := range ruleSet.Apply {
		rule.baseDir = inputFile.Dir()
	}

	for _, rule := range ruleSet.Drop {
		rule.baseDir = inputFile.Dir()
	}

	return &ruleSet, nil
}

func (self *ruleSet) process(inputFile *goldsmith.File) bool {
	for _, rule := range self.Apply {
		rule.apply(inputFile)
	}

	for _, rule := range self.Drop {
		if rule.drop(inputFile) {
			return false
		}
	}

	return true
}

// Rule chainable context.
type Rule struct {
	filename string

	ruleSets   []*ruleSet
	inputFiles []*goldsmith.File
	mutex      sync.Mutex
}

// New creates a new instance of the Rule plugin.
func New() *Rule {
	return &Rule{filename: "rules.toml"}
}

func (self *Rule) Filename(filename string) *Rule {
	self.filename = filename
	return self
}

func (*Rule) Name() string {
	return "rule"
}

func (self *Rule) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	if inputFile.Name() == self.filename {
		ruleSet, err := newRuleSet(inputFile)
		if err != nil {
			return err
		}

		self.mutex.Lock()
		self.ruleSets = append(self.ruleSets, ruleSet)
		self.mutex.Unlock()
	} else {
		self.mutex.Lock()
		self.inputFiles = append(self.inputFiles, inputFile)
		self.mutex.Unlock()
	}

	return nil
}

func (self *Rule) Finalize(context *goldsmith.Context) error {
	for _, inputFile := range self.inputFiles {
		var block bool
		for _, ruleSet := range self.ruleSets {
			if !ruleSet.process(inputFile) {
				block = true
				break
			}
		}

		if !block {
			context.DispatchFile(inputFile)
		}
	}

	return nil
}
