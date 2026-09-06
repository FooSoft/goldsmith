// Package rule applies path-based metadata and filtering rules.
package rule

import (
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
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

	return !wildcard.New(rejectPaths...).Accept(inputFile)
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

func (self *ruleSet) baseDir() string {
	if len(self.Apply) > 0 {
		return self.Apply[0].baseDir
	}
	if len(self.Drop) > 0 {
		return self.Drop[0].baseDir
	}
	return "."
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

func (self *Rule) Initialize(context *goldsmith.Context) error {
	self.ruleSets = nil
	self.inputFiles = nil
	return nil
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
	sort.Slice(self.ruleSets, func(i, j int) bool {
		left := filepath.ToSlash(self.ruleSets[i].baseDir())
		right := filepath.ToSlash(self.ruleSets[j].baseDir())
		leftDepth := strings.Count(left, "/")
		rightDepth := strings.Count(right, "/")
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return left < right
	})
	sort.Slice(self.inputFiles, func(i, j int) bool {
		return self.inputFiles[i].Path() < self.inputFiles[j].Path()
	})

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
