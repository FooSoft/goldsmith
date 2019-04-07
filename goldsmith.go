// Package goldsmith generates static websites.
package goldsmith

import (
	"hash"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"
)

// Goldsmith chainable context.
type Goldsmith struct {
	sourceDir string
	targetDir string

	contexts      []*Context
	contextHasher hash.Hash32

	fileRefs    map[string]bool
	fileFilters []Filter
	fileCache   *cache

	errors []error
	mutex  sync.Mutex
}

// Begin starts a chain, reading the files located in sourceDir as input.
func Begin(sourceDir string) *Goldsmith {
	goldsmith := &Goldsmith{
		sourceDir:     sourceDir,
		contextHasher: crc32.NewIEEE(),
		fileRefs:      make(map[string]bool),
	}

	goldsmith.Chain(new(loader))
	return goldsmith
}

// Cache enables caching in cacheDir for the remainder of the chain.
func (goldsmith *Goldsmith) Cache(cacheDir string) *Goldsmith {
	goldsmith.fileCache = &cache{cacheDir}
	return goldsmith
}

// Chain links a plugin instance into the chain.
func (goldsmith *Goldsmith) Chain(plugin Plugin) *Goldsmith {
	goldsmith.contextHasher.Write([]byte(plugin.Name()))

	context := &Context{
		goldsmith:   goldsmith,
		plugin:      plugin,
		hash:        goldsmith.contextHasher.Sum32(),
		outputFiles: make(chan *File),
	}

	context.fileFilters = append(context.fileFilters, goldsmith.fileFilters...)

	if len(goldsmith.contexts) > 0 {
		context.inputFiles = goldsmith.contexts[len(goldsmith.contexts)-1].outputFiles
	}

	goldsmith.contexts = append(goldsmith.contexts, context)
	return goldsmith
}

// FilterPush pushes a filter instance on the chain's filter stack.
func (goldsmith *Goldsmith) FilterPush(filter Filter) *Goldsmith {
	goldsmith.fileFilters = append(goldsmith.fileFilters, filter)
	return goldsmith
}

// FilterPop pops a filter instance from the chain's filter stack.
func (goldsmith *Goldsmith) FilterPop() *Goldsmith {
	count := len(goldsmith.fileFilters)
	if count == 0 {
		panic("attempted to pop empty filter stack")
	}

	goldsmith.fileFilters = goldsmith.fileFilters[:count-1]
	return goldsmith
}

// End stops a chain, writing all recieved files to targetDir as output.
func (goldsmith *Goldsmith) End(targetDir string) []error {
	goldsmith.targetDir = targetDir

	for _, context := range goldsmith.contexts {
		go context.step()
	}

	context := goldsmith.contexts[len(goldsmith.contexts)-1]

export:
	for file := range context.outputFiles {
		for _, fileFilter := range goldsmith.fileFilters {
			accept, err := fileFilter.Accept(file)
			if err != nil {
				goldsmith.fault(fileFilter.Name(), file, err)
				continue export
			}
			if !accept {
				continue export
			}
		}

		goldsmith.exportFile(file)
	}

	goldsmith.removeUnreferencedFiles()
	return goldsmith.errors
}

func (goldsmith *Goldsmith) retrieveFile(context *Context, outputPath string, inputFiles []*File) *File {
	if goldsmith.fileCache != nil {
		outputFile, _ := goldsmith.fileCache.retrieveFile(context, outputPath, inputFiles)
		return outputFile
	}

	return nil
}

func (goldsmith *Goldsmith) storeFile(context *Context, outputFile *File, inputFiles []*File) {
	if goldsmith.fileCache != nil {
		goldsmith.fileCache.storeFile(context, outputFile, inputFiles)
	}

}

func (goldsmith *Goldsmith) removeUnreferencedFiles() {
	infos := make(chan fileInfo)
	go scanDir(goldsmith.targetDir, infos)

	for info := range infos {
		if info.path != goldsmith.targetDir {
			relPath, _ := filepath.Rel(goldsmith.targetDir, info.path)
			if contained, _ := goldsmith.fileRefs[relPath]; !contained {
				os.RemoveAll(info.path)
			}
		}
	}
}

func (goldsmith *Goldsmith) exportFile(file *File) error {
	if err := file.export(goldsmith.targetDir); err != nil {
		return err
	}

	for pathSeg := cleanPath(file.sourcePath); pathSeg != "."; pathSeg = filepath.Dir(pathSeg) {
		goldsmith.fileRefs[pathSeg] = true
	}

	return nil
}

func (goldsmith *Goldsmith) fault(pluginName string, file *File, err error) {
	goldsmith.mutex.Lock()
	defer goldsmith.mutex.Unlock()

	faultError := &Error{Name: pluginName, Err: err}
	if file != nil {
		faultError.Path = file.sourcePath
	}

	goldsmith.errors = append(goldsmith.errors, faultError)
}
