package goldsmith

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

// Context corresponds to the current link in the chain and provides methods
// that enable plugins to inject new files into the chain.
type Context struct {
	chain  *chainState
	plugin Plugin

	filtersExt filterStack
	filtersInt filterStack

	threads  int
	index    int
	cacheKey string

	filesIn  chan *File
	filesOut chan *File
}

// CreateFileFromReader creates a new file instance from the provided reader.
func (self *Context) CreateFileFromReader(relPath string, reader io.Reader) (*File, error) {
	if err := validateRelativePath(relPath); err != nil {
		return nil, err
	}
	if reader == nil {
		return nil, errors.New("file reader cannot be nil")
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	file := &File{
		relPath: filepath.Clean(filepath.FromSlash(relPath)),
		props:   make(FileProps),
		modTime: time.Now(),
		size:    int64(len(data)),
		reader:  bytes.NewReader(data),
		index:   self.index,
	}

	return file, nil
}

// CreateFileFromAsset creates a new file instance from the provided file path.
func (self *Context) CreateFileFromAsset(relPath, dataPath string) (*File, error) {
	if err := validateRelativePath(relPath); err != nil {
		return nil, err
	}

	info, err := os.Stat(dataPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("file paths cannot be directories")
	}

	file := &File{
		relPath:  filepath.Clean(filepath.FromSlash(relPath)),
		props:    make(FileProps),
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

// DispatchAndCacheFile caches the file data (excluding the metadata) under the
// given keys, and then passes it to the next link in the chain. Keys must change
// whenever the output would; use FileCacheKeys to derive them from input files,
// or supply content hashes such as git commit IDs. Key order is significant.
func (self *Context) DispatchAndCacheFile(outputFile *File, keys ...string) {
	if self.chain.cache != nil {
		if err := self.chain.cache.storeFile(outputFile, self.cacheNamespace(), keys); err != nil {
			self.chain.fault("cache", outputFile, err)
		}
	}

	self.DispatchFile(outputFile)
}

// RetrieveCachedFile looks up file data (excluding the metadata), given an
// output path and the keys it was stored with by DispatchAndCacheFile. The
// function will return nil if the desired file is not found in the cache.
func (self *Context) RetrieveCachedFile(outputPath string, keys ...string) *File {
	if self.chain.cache == nil {
		return nil
	}

	outputFile, err := self.chain.cache.retrieveFile(self, outputPath, keys)
	if err != nil {
		self.chain.fault("cache", nil, err)
	}

	return outputFile
}

// FileCacheKeys identifies input files by path, modification time, and size,
// for use as cache keys. The result does not depend on the order of the files.
func FileCacheKeys(inputFiles ...*File) []string {
	var keys []string
	for _, inputFile := range inputFiles {
		if inputFile == nil {
			panic("goldsmith: cache dependency cannot be nil")
		}

		keys = append(keys, fmt.Sprintf("%q:%d:%d", inputFile.Path(), inputFile.ModTime().UnixNano(), inputFile.Size()))
	}

	sort.Strings(keys)
	return keys
}

// CacheKey identifies the plugin configuration used to generate cached files.
// Plugins with configurable output should set this during Initialize. Cached
// files are also isolated by plugin type, name, and position in the chain.
func (self *Context) CacheKey(key string) *Context {
	self.cacheKey = key
	return self
}

func (self *Context) cacheNamespace() string {
	return fmt.Sprintf("%T/%q/%d/%q", self.plugin, self.plugin.Name(), self.index, self.cacheKey)
}

// Filter specifies internal filters that exclude files from being processed.
func (self *Context) Filter(filters ...Filter) *Context {
	for _, filter := range filters {
		self.filtersInt.push(filter, self.index)
	}

	return self
}

// Threads specifies the maximum number of threads used for processing.
func (self *Context) Threads(threads int) *Context {
	self.threads = threads
	return self
}

func (self *Context) step() {
	defer close(self.filesOut)

	if initializer, ok := self.plugin.(Initializer); ok {
		if err := initializer.Initialize(self); err != nil {
			self.chain.fault(self.plugin.Name(), nil, err)
			if self.filesIn != nil {
				for range self.filesIn {
				}
			}
			return
		}
	}

	if self.filesIn != nil {
		processor, _ := self.plugin.(Processor)

		threads := self.threads
		if threads < 1 {
			if processor == nil {
				threads = 1
			} else {
				threads = runtime.NumCPU()
			}
		}

		var wg sync.WaitGroup
		for i := 0; i < threads; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for inputFile := range self.filesIn {
					if processor != nil && self.filtersInt.accept(inputFile) && self.filtersExt.accept(inputFile) {
						if _, err := inputFile.Seek(0, io.SeekStart); err != nil {
							self.chain.fault("core", inputFile, err)
							continue
						}
						if err := processor.Process(self, inputFile); err != nil {
							self.chain.fault(self.plugin.Name(), inputFile, err)
						}
					} else {
						self.DispatchFile(inputFile)
					}
				}
			}()
		}

		wg.Wait()
	}

	if finalizer, ok := self.plugin.(Finalizer); ok {
		if err := finalizer.Finalize(self); err != nil {
			self.chain.fault(self.plugin.Name(), nil, err)
		}
	}
}
