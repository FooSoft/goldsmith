// Package goldsmith generates static websites.
package goldsmith

import (
	"fmt"
	"sync"
)

// Goldsmith chainable context.
type Goldsmith struct {
	chain *chainState
}

// Begin starts a chain, reading the files located in the source directory as input.
func (self *Goldsmith) Begin(sourceDir string) *Goldsmith {
	self.chain = new(chainState)
	self.Chain(&fileImporter{sourceDir: sourceDir})
	return self
}

// Cache enables caching in cacheDir for the remainder of the chain.
func (self *Goldsmith) Cache(cacheDir string) *Goldsmith {
	self.chain.cache = &cache{cacheDir}
	return self
}

// Clean enables or disables removal of leftover files in the target directory.
func (self *Goldsmith) Clean(clean bool) *Goldsmith {
	self.chain.clean = clean
	return self
}

// Chain links a plugin instance into the chain.
func (self *Goldsmith) Chain(plugin Plugin) *Goldsmith {
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
	self.chain.filters.push(filter, self.chain.index)
	self.chain.index++
	return self
}

// FilterPop pops a filter instance from the chain's filter stack.
func (self *Goldsmith) FilterPop() *Goldsmith {
	self.chain.filters.pop()
	self.chain.index++
	return self
}

// End stops a chain, writing all recieved files to targetDir as output.
func (self *Goldsmith) End(targetDir string) []error {
	self.Chain(&fileExporter{targetDir: targetDir, clean: self.chain.clean})
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

type chainState struct {
	contexts []*Context

	cache   *cache
	filters filterStack
	clean   bool
	index   int

	errors []error
	mutex  sync.Mutex
}

func (self *chainState) fault(name string, file *File, err error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	var faultError error
	if file == nil {
		faultError = fmt.Errorf("[%s]: %w", name, err)
	} else {
		faultError = fmt.Errorf("[%s@%v]: %w", name, file, err)
	}

	self.errors = append(self.errors, faultError)
}
