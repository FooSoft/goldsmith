package goldsmith

import (
	"bytes"
	"errors"
	"os"
	"runtime"
	"sync"
	"time"
)

// Context corresponds to the current link in the chain and provides methods
// that enable plugins to inject new files into the chain.
type Context struct {
	goldsmith *Goldsmith

	plugin    Plugin
	chainHash uint32

	filtersExt filterStack
	filtersInt filterStack

	threads int

	filesIn  chan *File
	filesOut chan *File
}

// CreateFileFrom data creates a new file instance from the provided data buffer.
func (*Context) CreateFileFromData(sourcePath string, data []byte) *File {
	return &File{
		sourcePath: sourcePath,
		Meta:       make(map[string]interface{}),
		reader:     bytes.NewReader(data),
		size:       int64(len(data)),
		modTime:    time.Now(),
	}
}

// CreateFileFromAsset creates a new file instance from the provided file path.
func (*Context) CreateFileFromAsset(sourcePath, dataPath string) (*File, error) {
	info, err := os.Stat(dataPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("assets must be files")
	}

	file := &File{
		sourcePath: sourcePath,
		dataPath:   dataPath,
		Meta:       make(map[string]interface{}),
		size:       info.Size(),
		modTime:    info.ModTime(),
	}

	return file, nil
}

// DispatchFile causes the file to get passed to the next link in the chain.
func (context *Context) DispatchFile(file *File) {
	context.filesOut <- file
}

// DispatchAndCacheFile caches the file data (excluding the metadata), taking
// dependencies on any input files that are needed to generate it, and then
// passes it to the next link in the chain.
func (context *Context) DispatchAndCacheFile(outputFile *File, inputFiles ...*File) {
	if context.goldsmith.fileCache != nil {
		context.goldsmith.fileCache.storeFile(context, outputFile, inputFiles)
	}

	context.filesOut <- outputFile
}

// RetrieveCachedFile looks up file data (excluding the metadata), given an
// output path and any input files that are needed to generate it. The function
// will return nil if the desired file is not found in the cache.
func (context *Context) RetrieveCachedFile(outputPath string, inputFiles ...*File) *File {
	var outputFile *File
	if context.goldsmith.fileCache != nil {
		outputFile, _ = context.goldsmith.fileCache.retrieveFile(context, outputPath, inputFiles)
	}

	return outputFile
}

// Specify internal filter(s) that exclude files from being processed.
func (context *Context) Filter(filters ...Filter) *Context {
	context.filtersInt = filters
	return context
}

// Specify the maximum number of threads used for processing.
func (context *Context) Threads(threads int) *Context {
	context.threads = threads
	return context
}

func (context *Context) step() {
	defer close(context.filesOut)

	if initializer, ok := context.plugin.(Initializer); ok {
		if err := initializer.Initialize(context); err != nil {
			context.goldsmith.fault(context.plugin.Name(), nil, err)
			return
		}
	}

	if context.filesIn != nil {
		processor, _ := context.plugin.(Processor)

		threads := context.threads
		if threads < 1 {
			threads = runtime.NumCPU()
		}

		var wg sync.WaitGroup
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for inputFile := range context.filesIn {
					if processor != nil && context.filtersInt.accept(inputFile) && context.filtersExt.accept(inputFile) {
						if _, err := inputFile.Seek(0, os.SEEK_SET); err != nil {
							context.goldsmith.fault("core", inputFile, err)
						}
						if err := processor.Process(context, inputFile); err != nil {
							context.goldsmith.fault(context.plugin.Name(), inputFile, err)
						}
					} else {
						context.filesOut <- inputFile
					}
				}
			}()
		}
		wg.Wait()
	}

	if finalizer, ok := context.plugin.(Finalizer); ok {
		if err := finalizer.Finalize(context); err != nil {
			context.goldsmith.fault(context.plugin.Name(), nil, err)
		}
	}
}
