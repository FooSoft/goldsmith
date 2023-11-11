package goldsmith

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Context corresponds to the current link in the chain and provides methods
// that enable plugins to inject new files into the chain.
type Context struct {
	goldsmith *Goldsmith

	plugin Plugin

	filtersExt filterStack
	filtersInt filterStack

	threads int
	index   int

	filesIn  chan *File
	filesOut chan *File
}

// CreateFileFrom data creates a new file instance from the provided data buffer.
func (self *Context) CreateFileFromReader(sourcePath string, reader io.Reader) (*File, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	file := &File{
		relPath: sourcePath,
		props:   make(map[string]Prop),
		modTime: time.Now(),
		size:    int64(len(data)),
		reader:  bytes.NewReader(data),
		index:   self.index,
	}

	return file, nil
}

// CreateFileFromAsset creates a new file instance from the provided file path.
func (self *Context) CreateFileFromAsset(sourcePath, dataPath string) (*File, error) {
	if filepath.IsAbs(sourcePath) {
		return nil, errors.New("source paths must be relative")
	}

	if filepath.IsAbs(dataPath) {
		return nil, errors.New("data paths must be relative")
	}

	info, err := os.Stat(dataPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("assets must be files")
	}

	file := &File{
		relPath:  sourcePath,
		props:    make(map[string]Prop),
		modTime:  info.ModTime(),
		size:     info.Size(),
		dataPath: dataPath,
		index:    self.index,
	}

	return file, nil
}

// DispatchFile causes the file to get passed to the next link in the chain.
func (self *Context) DispatchFile(file *File) {
	self.filesOut <- file
}

// DispatchAndCacheFile caches the file data (excluding the metadata), taking
// dependencies on any input files that are needed to generate it, and then
// passes it to the next link in the chain.
func (self *Context) DispatchAndCacheFile(outputFile *File, inputFiles ...*File) {
	if self.goldsmith.cache != nil {
		self.goldsmith.cache.storeFile(self, outputFile, inputFiles)
	}

	self.filesOut <- outputFile
}

// RetrieveCachedFile looks up file data (excluding the metadata), given an
// output path and any input files that are needed to generate it. The function
// will return nil if the desired file is not found in the cache.
func (self *Context) RetrieveCachedFile(outputPath string, inputFiles ...*File) *File {
	var outputFile *File
	if self.goldsmith.cache != nil {
		outputFile, _ = self.goldsmith.cache.retrieveFile(self, outputPath, inputFiles)
	}

	return outputFile
}

// Specify internal filter(s) that exclude files from being processed.
func (self *Context) Filter(filters ...Filter) *Context {
	for _, filter := range filters {
		self.filtersInt.push(filter, self.index)
	}

	return self
}

// Specify the maximum number of threads used for processing.
func (self *Context) Threads(threads int) *Context {
	self.threads = threads
	return self
}

func (self *Context) step() {
	defer close(self.filesOut)

	if initializer, ok := self.plugin.(Initializer); ok {
		if err := initializer.Initialize(self); err != nil {
			self.goldsmith.fault(self.plugin.Name(), nil, err)
			return
		}
	}

	if self.filesIn != nil {
		processor, _ := self.plugin.(Processor)

		threads := self.threads
		if threads < 1 {
			threads = runtime.NumCPU()
		}

		var wg sync.WaitGroup
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for inputFile := range self.filesIn {
					if processor != nil && self.filtersInt.accept(inputFile) && self.filtersExt.accept(inputFile) {
						if _, err := inputFile.Seek(0, io.SeekStart); err != nil {
							self.goldsmith.fault("core", inputFile, err)
						}
						if err := processor.Process(self, inputFile); err != nil {
							self.goldsmith.fault(self.plugin.Name(), inputFile, err)
						}
					} else {
						self.filesOut <- inputFile
					}
				}
			}()
		}

		wg.Wait()
	}

	if finalizer, ok := self.plugin.(Finalizer); ok {
		if err := finalizer.Finalize(self); err != nil {
			self.goldsmith.fault(self.plugin.Name(), nil, err)
		}
	}
}
