// Package goldsmith generates static websites.
package goldsmith

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Goldsmith chainable context.
type Goldsmith struct {
	CacheDir string
	Clean    bool

	chain *chainState
}

// Begin starts a chain, reading the files located in the source directory as input.
func (self *Goldsmith) Begin(sourceDir string) *Goldsmith {
	self.chain = &chainState{sourceDir: sourceDir}
	if len(self.CacheDir) > 0 {
		self.chain.cache = &cache{self.CacheDir}
	}

	self.Chain(&fileImporter{sourceDir: sourceDir})
	return self
}

// Chain links a plugin instance into the chain.
func (self *Goldsmith) Chain(plugin Plugin) *Goldsmith {
	if self.chain == nil {
		panic("goldsmith: Begin must be called before Chain")
	}
	if plugin == nil {
		panic("goldsmith: plugin cannot be nil")
	}

	context := &Context{
		chain:      self.chain,
		plugin:     plugin,
		filtersExt: append(filterStack(nil), self.chain.filters...),
		index:      self.chain.index,
		filesOut:   make(chan *File),
	}

	if len(self.chain.contexts) > 0 {
		context.filesIn = self.chain.contexts[len(self.chain.contexts)-1].filesOut
	}

	self.chain.contexts = append(self.chain.contexts, context)
	self.chain.index++

	return self
}

// FilterPush pushes a filter instance on the chain's filter stack.
func (self *Goldsmith) FilterPush(filter Filter) *Goldsmith {
	if self.chain == nil {
		panic("goldsmith: Begin must be called before FilterPush")
	}
	if filter == nil {
		panic("goldsmith: filter cannot be nil")
	}

	self.chain.filters.push(filter, self.chain.index)
	self.chain.index++
	return self
}

// FilterPop pops a filter instance from the chain's filter stack.
func (self *Goldsmith) FilterPop() *Goldsmith {
	if self.chain == nil {
		panic("goldsmith: Begin must be called before FilterPop")
	}

	self.chain.filters.pop()
	self.chain.index++
	return self
}

// End stops a chain, writing all received files to targetDir as output.
func (self *Goldsmith) End(targetDir string) []error {
	if self.chain == nil {
		return []error{fmt.Errorf("goldsmith: Begin must be called before End")}
	}
	if err := validateBuildDirectories(self.chain.sourceDir, targetDir, self.CacheDir); err != nil {
		self.chain = nil
		return []error{err}
	}

	self.Chain(&fileExporter{targetDir: targetDir, clean: self.Clean})

	for _, context := range self.chain.contexts {
		go context.step()
	}

	context := self.chain.contexts[len(self.chain.contexts)-1]
	for range context.filesOut {

	}

	errors := self.chain.errors
	self.chain = nil

	return errors
}

func validateBuildDirectories(sourceDir, targetDir, cacheDir string) error {
	type namedDir struct {
		name string
		path string
	}

	dirs := []namedDir{{"source", sourceDir}, {"target", targetDir}}
	if cacheDir != "" {
		dirs = append(dirs, namedDir{"cache", cacheDir})
	}
	for i := range dirs {
		for j := i + 1; j < len(dirs); j++ {
			overlap, err := directoriesOverlap(dirs[i].path, dirs[j].path)
			if err != nil {
				return err
			}
			if overlap {
				return fmt.Errorf("goldsmith: %s and %s directories overlap", dirs[i].name, dirs[j].name)
			}
		}
	}

	return nil
}

func directoriesOverlap(first, second string) (bool, error) {
	first, err := canonicalPath(first)
	if err != nil {
		return false, err
	}
	second, err = canonicalPath(second)
	if err != nil {
		return false, err
	}

	contains := func(parent, child string) (bool, error) {
		if !strings.EqualFold(filepath.VolumeName(parent), filepath.VolumeName(child)) {
			return false, nil
		}
		relPath, err := filepath.Rel(parent, child)
		if err != nil {
			return false, err
		}
		return relPath == "." || (relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator))), nil
	}
	if containsSecond, err := contains(first, second); err != nil || containsSecond {
		return containsSecond, err
	}
	return contains(second, first)
}

func canonicalPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	var suffix []string
	for current := filepath.Clean(absPath); ; current = filepath.Dir(current) {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return absPath, nil
		}
		suffix = append(suffix, filepath.Base(current))
	}
}
