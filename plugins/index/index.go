// Package index creates metadata for directory listings and generates index
// pages for every directory which contains other files. This is useful for
// creating static directory views for downloads, image galleries, etc.
package index

import (
	"bytes"
	"path/filepath"
	"sort"
	"sync"

	"go.foosoft.net/goldsmith"
)

// Entry contains information about a directory item.
type Entry struct {
	Name  string
	Path  string
	IsDir bool
	File  *goldsmith.File
}

// Index chainable plugin context.
type Index struct {
	indexName  string
	filesKey   string
	indexProps map[string]interface{}

	dirLists    map[string]*directory
	dirsHandled map[string]bool
	mutex       sync.Mutex
}

// New creates a new instance of the Index plugin.
// The meta parameter allows additional metadata to be provided for generated indices.
func New(indexProps map[string]interface{}) *Index {
	return &Index{
		indexName:   "index.html",
		indexProps:  indexProps,
		filesKey:    "Files",
		dirsHandled: make(map[string]bool),
		dirLists:    make(map[string]*directory),
	}
}

// IndexFilename sets the name of the file to be created as the directory index (default: "index.html").
func (self *Index) IndexFilename(filename string) *Index {
	self.indexName = filename
	return self
}

// FilesKey sets the metadata key used to access the files in the current directory (default: "Files").
func (self *Index) FilesKey(key string) *Index {
	self.filesKey = key
	return self
}

func (*Index) Name() string {
	return "index"
}

func (self *Index) Initialize(context *goldsmith.Context) error {
	self.dirsHandled = make(map[string]bool)
	self.dirLists = make(map[string]*directory)
	return nil
}

func (self *Index) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	currentPath := inputFile.Path()
	currentIsDir := false

	for {
		if handled, _ := self.dirsHandled[currentPath]; handled {
			break
		}

		self.dirsHandled[currentPath] = true

		currentDir := filepath.Dir(currentPath)
		currentBase := filepath.Base(currentPath)

		list, ok := self.dirLists[currentDir]
		if !ok {
			list = new(directory)
			self.dirLists[currentDir] = list
		}

		if !currentIsDir {
			if currentBase == self.indexName {
				list.indexFile = inputFile
			} else {
				context.DispatchFile(inputFile)
			}
		}

		if currentIsDir || currentBase != self.indexName {
			entry := Entry{Name: currentBase, Path: currentPath, IsDir: currentIsDir, File: inputFile}
			list.entries = append(list.entries, entry)
		}

		if currentDir == "." {
			break
		}

		currentPath = currentDir
		currentIsDir = true
	}

	return nil
}

func (self *Index) Finalize(context *goldsmith.Context) error {
	dirNames := make([]string, 0, len(self.dirLists))
	for name := range self.dirLists {
		dirNames = append(dirNames, name)
	}
	sort.Strings(dirNames)

	for _, name := range dirNames {
		list := self.dirLists[name]
		sort.Sort(list.entries)

		indexFile := list.indexFile
		if indexFile == nil {
			var err error
			indexFile, err = context.CreateFileFromReader(filepath.Join(name, self.indexName), bytes.NewReader(nil))
			if err != nil {
				return err
			}

			for name, value := range self.indexProps {
				indexFile.SetProp(name, value)
			}
		}

		indexFile.SetProp(self.filesKey, list.entries)
		context.DispatchFile(indexFile)
	}

	return nil
}

type directory struct {
	entries   entriesByName
	indexFile *goldsmith.File
}

type entriesByName []Entry

func (self entriesByName) Len() int {
	return len(self)
}

func (self entriesByName) Less(i, j int) bool {
	e1, e2 := self[i], self[j]

	if e1.IsDir && !e2.IsDir {
		return true
	}
	if !e1.IsDir && e2.IsDir {
		return false
	}

	return e1.Name < e2.Name
}

func (self entriesByName) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}
