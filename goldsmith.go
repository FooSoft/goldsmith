// Package goldsmith generates static websites.
package goldsmith

import (
	"fmt"
	"sync"
)

// Goldsmith chainable context.
type Goldsmith struct {
	sourceDir string
	targetDir string

	contexts []*Context

	cache   *cache
	filters filterStack
	clean   bool
	index   int

	errors []error
	mutex  sync.Mutex
}

// Begin starts a chain, reading the files located in the source directory as input.
func Begin(sourceDir string) *Goldsmith {
	goldsmith := &Goldsmith{sourceDir: sourceDir}
	goldsmith.Chain(&loader{})
	return goldsmith
}

// Cache enables caching in cacheDir for the remainder of the chain.
func (self *Goldsmith) Cache(cacheDir string) *Goldsmith {
	self.cache = &cache{cacheDir}
	return self
}

// Clean enables or disables removal of leftover files in the target directory.
func (self *Goldsmith) Clean(clean bool) *Goldsmith {
	self.clean = clean
	return self
}

// Chain links a plugin instance into the chain.
func (self *Goldsmith) Chain(plugin Plugin) *Goldsmith {
	context := &Context{
		goldsmith:  self,
		plugin:     plugin,
		filtersExt: append(filterStack(nil), self.filters...),
		index:      self.index,
		filesOut:   make(chan *File),
	}

	if len(self.contexts) > 0 {
		context.filesIn = self.contexts[len(self.contexts)-1].filesOut
	}

	self.contexts = append(self.contexts, context)
	self.index++

	return self
}

// FilterPush pushes a filter instance on the chain's filter stack.
func (self *Goldsmith) FilterPush(filter Filter) *Goldsmith {
	self.filters.push(filter, self.index)
	self.index++
	return self
}

// FilterPop pops a filter instance from the chain's filter stack.
func (self *Goldsmith) FilterPop() *Goldsmith {
	self.filters.pop()
	self.index++
	return self
}

// End stops a chain, writing all recieved files to targetDir as output.
func (self *Goldsmith) End(targetDir string) []error {
	self.targetDir = targetDir

	self.Chain(&saver{clean: self.clean})
	for _, context := range self.contexts {
		go context.step()
	}

	context := self.contexts[len(self.contexts)-1]
	for range context.filesOut {

	}

	return self.errors
}

func (self *Goldsmith) fault(name string, file *File, err error) {
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
