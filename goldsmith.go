// Package goldsmith generates static websites.
package goldsmith

// Goldsmith chainable context.
type Goldsmith struct {
	CacheDir string
	Clean    bool

	chain *chainState
}

// Begin starts a chain, reading the files located in the source directory as input.
func (self *Goldsmith) Begin(sourceDir string) *Goldsmith {
	self.chain = &chainState{}
	if len(self.CacheDir) > 0 {
		self.chain.cache = &cache{self.CacheDir}
	}

	self.Chain(&fileImporter{sourceDir: sourceDir})
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
