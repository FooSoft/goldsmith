package goldsmith

import (
	"hash"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"
)

type Goldsmith struct {
	sourceDir string
	targetDir string

	contexts      []*Context
	contextHasher hash.Hash32

	fileRefs    map[string]bool
	fileFilters []Filter
	fileCache   *fileCache

	errors   []error
	errorMtx sync.Mutex
}

func Begin(sourceDir string) *Goldsmith {
	gs := &Goldsmith{
		sourceDir:     sourceDir,
		contextHasher: crc32.NewIEEE(),
		fileRefs:      make(map[string]bool),
	}

	gs.Chain(new(loader))
	return gs
}

func (gs *Goldsmith) Cache(cacheDir string) *Goldsmith {
	gs.fileCache = &fileCache{cacheDir}
	return gs
}

func (gs *Goldsmith) Chain(plugin Plugin) *Goldsmith {
	gs.contextHasher.Write([]byte(plugin.Name()))

	context := &Context{
		goldsmith:   gs,
		plugin:      plugin,
		hash:        gs.contextHasher.Sum32(),
		outputFiles: make(chan *File),
	}

	context.fileFilters = append(context.fileFilters, gs.fileFilters...)

	if len(gs.contexts) > 0 {
		context.inputFiles = gs.contexts[len(gs.contexts)-1].outputFiles
	}

	gs.contexts = append(gs.contexts, context)
	return gs
}

func (gs *Goldsmith) FilterPush(filter Filter) *Goldsmith {
	gs.fileFilters = append(gs.fileFilters, filter)
	return gs
}

func (gs *Goldsmith) FilterPop() *Goldsmith {
	count := len(gs.fileFilters)
	if count == 0 {
		panic("attempted to pop empty filter stack")
	}

	gs.fileFilters = gs.fileFilters[:count-1]
	return gs
}

func (gs *Goldsmith) End(targetDir string) []error {
	gs.targetDir = targetDir

	for _, context := range gs.contexts {
		go context.step()
	}

	context := gs.contexts[len(gs.contexts)-1]
	for file := range context.outputFiles {
		gs.exportFile(file)
	}

	gs.removeUnreferencedFiles()
	return gs.errors
}

func (gs *Goldsmith) retrieveFile(context *Context, outputPath string, inputFiles []*File) *File {
	if gs.fileCache != nil {
		outputFile, _ := gs.fileCache.retrieveFile(context, outputPath, inputFiles)
		return outputFile
	}

	return nil
}

func (gs *Goldsmith) storeFile(context *Context, outputFile *File, inputFiles []*File) {
	if gs.fileCache != nil {
		gs.fileCache.storeFile(context, outputFile, inputFiles)
	}
}

func (gs *Goldsmith) removeUnreferencedFiles() {
	infos := make(chan fileInfo)
	go scanDir(gs.targetDir, infos)

	for info := range infos {
		if info.path != gs.targetDir {
			relPath, _ := filepath.Rel(gs.targetDir, info.path)
			if contained, _ := gs.fileRefs[relPath]; !contained {
				os.RemoveAll(info.path)
			}
		}
	}
}

func (gs *Goldsmith) exportFile(file *File) error {
	if err := file.export(gs.targetDir); err != nil {
		return err
	}

	for pathSeg := cleanPath(file.sourcePath); pathSeg != "."; pathSeg = filepath.Dir(pathSeg) {
		gs.fileRefs[pathSeg] = true
	}

	return nil
}

func (gs *Goldsmith) fault(pluginName string, file *File, err error) {
	gs.errorMtx.Lock()
	defer gs.errorMtx.Unlock()

	faultError := &Error{Name: pluginName, Err: err}
	if file != nil {
		faultError.Path = file.sourcePath
	}

	gs.errors = append(gs.errors, faultError)
}
