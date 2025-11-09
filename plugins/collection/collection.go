// Package collection groups related pages into named collections. This can be
// useful for presenting blog posts on your front page, and displaying summary
// information about other types of content on your website. It can be used in
// conjunction with the "pager" plugin to create large collections which are
// split across several pages.
package collection

import (
	"sort"
	"strings"
	"sync"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// A Comparer callback function is used to sort files within a collection group.
type Comparer func(i, j *goldsmith.File) (less bool)

// Collection chainable plugin context.
type Collection struct {
	collectionKey string
	groupsKey     string
	comparer      Comparer

	groups map[string][]*goldsmith.File
	files  []*goldsmith.File
	mutex  sync.Mutex
}

// New creates a new instance of the Collection plugin.
func New() *Collection {
	return &Collection{
		collectionKey: "Collection",
		groupsKey:     "Groups",
		groups:        make(map[string][]*goldsmith.File),
	}
}

// CollectionKey sets the metadata key used to access the collection name (default: "Collection").
// The metadata associated with this key can be either a single string or an array of strings.
func (self *Collection) CollectionKey(collectionKey string) *Collection {
	self.collectionKey = collectionKey
	return self
}

// GroupsKey sets the metadata key used to store information about collection groups (default: "Groups").
// This information is stored as a mapping of group names to contained files.
func (self *Collection) GroupsKey(groupsKey string) *Collection {
	self.groupsKey = groupsKey
	return self
}

// Comparer sets the function used to sort files in collection groups (default: sort by filenames).
func (plugin *Collection) Comparer(comparer Comparer) *Collection {
	plugin.comparer = comparer
	return plugin
}

func (*Collection) Name() string {
	return "collection"
}

func (*Collection) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Collection) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	self.mutex.Lock()
	defer func() {
		inputFile.SetProp(self.groupsKey, self.groups)
		self.files = append(self.files, inputFile)
		self.mutex.Unlock()
	}()

	collectionRaw, ok := inputFile.Prop(self.collectionKey)
	if !ok {
		return nil
	}

	var collectionNames []string
	switch t := collectionRaw.(type) {
	case string:
		collectionNames = append(collectionNames, t)
	case []string:
		collectionNames = append(collectionNames, t...)
	}

	for _, collectionName := range collectionNames {
		files, _ := self.groups[collectionName]
		files = append(files, inputFile)
		self.groups[collectionName] = files
	}

	return nil
}

func (self *Collection) Finalize(context *goldsmith.Context) error {
	for _, files := range self.groups {
		fg := &fileSorter{files, self.comparer}
		sort.Sort(fg)
	}

	for _, file := range self.files {
		context.DispatchFile(file)
	}

	return nil
}

type fileSorter struct {
	files    []*goldsmith.File
	comparer Comparer
}

func (self fileSorter) Len() int {
	return len(self.files)
}

func (self fileSorter) Swap(i, j int) {
	self.files[i], self.files[j] = self.files[j], self.files[i]
}

func (self fileSorter) Less(i, j int) bool {
	if self.comparer == nil {
		return strings.Compare(self.files[i].Path(), self.files[j].Path()) < 0
	}

	return self.comparer(self.files[i], self.files[j])
}
