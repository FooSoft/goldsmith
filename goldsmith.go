// Package goldsmith generates static websites.
package goldsmith

import (
	"fmt"
	"hash"
	"hash/crc32"
	"sync"
)

// Goldsmith chainable context.
type Goldsmith struct {
	sourceDir string
	targetDir string

	contexts      []*Context
	contextHasher hash.Hash32

	fileCache *cache
	filters   filterStack
	clean     bool

	errors []error
	mutex  sync.Mutex
}

// Begin starts a chain, reading the files located in the source directory as input.
func Begin(sourceDir string) *Goldsmith {
	goldsmith := &Goldsmith{
		sourceDir:     sourceDir,
		contextHasher: crc32.NewIEEE(),
	}

	goldsmith.Chain(&loader{})
	return goldsmith
}

// Cache enables caching in cacheDir for the remainder of the chain.
func (goldsmith *Goldsmith) Cache(cacheDir string) *Goldsmith {
	goldsmith.fileCache = &cache{cacheDir}
	return goldsmith
}

// Clean enables or disables removal of leftover files in the target directory.
func (goldsmith *Goldsmith) Clean(clean bool) *Goldsmith {
	goldsmith.clean = clean
	return goldsmith
}

// Chain links a plugin instance into the chain.
func (goldsmith *Goldsmith) Chain(plugin Plugin) *Goldsmith {
	goldsmith.contextHasher.Write([]byte(plugin.Name()))

	context := &Context{
		goldsmith: goldsmith,
		plugin:    plugin,
		chainHash: goldsmith.contextHasher.Sum32(),
		filesOut:  make(chan *File),
	}

	context.filtersExt = append(context.filtersExt, goldsmith.filters...)

	if len(goldsmith.contexts) > 0 {
		context.filesIn = goldsmith.contexts[len(goldsmith.contexts)-1].filesOut
	}

	goldsmith.contexts = append(goldsmith.contexts, context)
	return goldsmith
}

// FilterPush pushes a filter instance on the chain's filter stack.
func (goldsmith *Goldsmith) FilterPush(filter Filter) *Goldsmith {
	goldsmith.filters.push(filter)
	return goldsmith
}

// FilterPop pops a filter instance from the chain's filter stack.
func (goldsmith *Goldsmith) FilterPop() *Goldsmith {
	goldsmith.filters.pop()
	return goldsmith
}

// End stops a chain, writing all recieved files to targetDir as output.
func (goldsmith *Goldsmith) End(targetDir string) []error {
	goldsmith.targetDir = targetDir

	goldsmith.Chain(&saver{
		clean: goldsmith.clean,
	})

	for _, context := range goldsmith.contexts {
		go context.step()
	}

	return goldsmith.errors
}

func (goldsmith *Goldsmith) fault(name string, file *File, err error) {
	goldsmith.mutex.Lock()
	defer goldsmith.mutex.Unlock()

	var faultError error
	if file == nil {
		faultError = fmt.Errorf("[%s]: %w", name, err)
	} else {
		faultError = fmt.Errorf("[%s@%v]: %w", name, file, err)
	}

	goldsmith.errors = append(goldsmith.errors, faultError)
}
