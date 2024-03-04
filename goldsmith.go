// Package goldsmith generates static websites.
package goldsmith

import (
	"fmt"
	"sync"
)

type chainState struct {
	contexts []*contextImpl

	cache   *cache
	filters filterStack
	clean   bool
	index   int

	errors []error
	mutex  sync.Mutex
}

// Goldsmith chainable context.
type Goldsmith struct {
	state *chainState
}

// Begin starts a chain, reading the files located in the source directory as input.
func (self *Goldsmith) Begin(sourceDir string) *Goldsmith {
	self.state = new(chainState)
	self.Chain(&fileImporter{sourceDir: sourceDir})
	return self
}

// Cache enables caching in cacheDir for the remainder of the chain.
func (self *Goldsmith) Cache(cacheDir string) *Goldsmith {
	self.state.cache = &cache{cacheDir}
	return self
}

// Clean enables or disables removal of leftover files in the target directory.
func (self *Goldsmith) Clean(clean bool) *Goldsmith {
	self.state.clean = clean
	return self
}

// Chain links a plugin instance into the chain.
func (self *Goldsmith) Chain(plugin Plugin) *Goldsmith {
	context := &contextImpl{
		goldsmith:  self,
		plugin:     plugin,
		filtersExt: append(filterStack(nil), self.state.filters...),
		index:      self.state.index,
		filesOut:   make(chan File),
	}

	if len(self.state.contexts) > 0 {
		context.filesIn = self.state.contexts[len(self.state.contexts)-1].filesOut
	}

	self.state.contexts = append(self.state.contexts, context)
	self.state.index++

	return self
}

// FilterPush pushes a filter instance on the chain's filter stack.
func (self *Goldsmith) FilterPush(filter Filter) *Goldsmith {
	self.state.filters.push(filter, self.state.index)
	self.state.index++
	return self
}

// FilterPop pops a filter instance from the chain's filter stack.
func (self *Goldsmith) FilterPop() *Goldsmith {
	self.state.filters.pop()
	self.state.index++
	return self
}

// End stops a chain, writing all recieved files to targetDir as output.
func (self *Goldsmith) End(targetDir string) []error {
	self.Chain(&fileExporter{targetDir: targetDir, clean: self.state.clean})
	for _, context := range self.state.contexts {
		go context.step()
	}

	context := self.state.contexts[len(self.state.contexts)-1]
	for range context.filesOut {

	}

	errors := self.state.errors
	self.state = nil

	return errors
}

func (self *Goldsmith) fault(name string, file File, err error) {
	self.state.mutex.Lock()
	defer self.state.mutex.Unlock()

	var faultError error
	if file == nil {
		faultError = fmt.Errorf("[%s]: %w", name, err)
	} else {
		faultError = fmt.Errorf("[%s@%v]: %w", name, file, err)
	}

	self.state.errors = append(self.state.errors, faultError)
}
