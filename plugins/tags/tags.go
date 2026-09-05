// Package tags builds tag clouds from file metadata. This makes it easy to
// create lists of all files tagged with a specific tag, as well as to see all
// tags globally used on a site.
package tags

import (
	"bytes"
	"path"
	"sort"
	"strings"
	"sync"
	"unicode"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// TagInfo contains site-wide information about a particular tag.
type TagInfo struct {
	TaggedFiles filesByPath
	IndexFile   *goldsmith.File
	SafeName    string
	RawName     string
}

// TagState contains site-wide information about tags used on a site.
type TagState struct {
	CurrentTag  *TagInfo
	CurrentTags tagInfoByName
	TagsByName  *tagInfoByName
	TagsByCount *tagInfoByCount
}

// Tags chainable context.
type Tags struct {
	tagsKey  string
	stateKey string

	baseDir   string
	indexName string
	indexMeta map[string]interface{}

	info        map[string]*TagInfo
	infoByName  tagInfoByName
	infoByCount tagInfoByCount

	files []*goldsmith.File
	mutex sync.Mutex
}

// New creates a new instance of the Tags plugin.
func New() *Tags {
	return &Tags{
		tagsKey:   "Tags",
		stateKey:  "TagState",
		baseDir:   "tags",
		indexName: "index.html",
		info:      make(map[string]*TagInfo),
	}
}

// TagsKey sets the metadata key used to get the tags for this file, stored as a slice of strings (default: "Tags").
func (self *Tags) TagsKey(key string) *Tags {
	self.tagsKey = key
	return self
}

// StateKey sets the meatadata key used to store site-wide tag information (default: "TagState").
func (self *Tags) StateKey(key string) *Tags {
	self.stateKey = key
	return self
}

// IndexName sets the filename which will be used to create tag list files (default: "index.html").
func (plugin *Tags) IndexName(name string) *Tags {
	plugin.indexName = name
	return plugin
}

// IndexMeta sets the metadata which will be assigned to generated tag list files (default: {}).
func (self *Tags) IndexMeta(meta map[string]interface{}) *Tags {
	self.indexMeta = meta
	return self
}

// BaseDir sets the base directory used to generate tag list files (default: "tags").
func (self *Tags) BaseDir(dir string) *Tags {
	self.baseDir = dir
	return self
}

func (*Tags) Name() string {
	return "tags"
}

func (*Tags) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Tags) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	tagState := &TagState{
		TagsByName:  &self.infoByName,
		TagsByCount: &self.infoByCount,
	}

	self.mutex.Lock()
	defer func() {
		inputFile.SetProp(self.stateKey, tagState)
		self.files = append(self.files, inputFile)
		self.mutex.Unlock()
	}()

	tagsRaw, ok := inputFile.Prop(self.tagsKey)
	if !ok {
		return nil
	}

	var tags []string
	switch values := tagsRaw.(type) {
	case []string:
		tags = append(tags, values...)
	case []interface{}:
		for _, value := range values {
			if tag, ok := value.(string); ok {
				tags = append(tags, tag)
			}
		}
	default:
		return nil
	}

	for _, tagRaw := range tags {
		tagSafe := safeTag(tagRaw)
		if len(tagSafe) == 0 {
			continue
		}

		var duplicate bool
		for _, tagState := range tagState.CurrentTags {
			if tagState.RawName == tagRaw {
				duplicate = true
				break
			}
		}

		if duplicate {
			continue
		}

		info, ok := self.info[tagRaw]
		if !ok {
			info = &TagInfo{
				SafeName: tagSafe,
				RawName:  tagRaw,
			}

			self.info[tagRaw] = info
		}
		info.TaggedFiles = append(info.TaggedFiles, inputFile)

		tagState.CurrentTags = append(tagState.CurrentTags, info)
	}

	sort.Sort(tagState.CurrentTags)

	return nil
}

func (self *Tags) Finalize(context *goldsmith.Context) error {
	for _, info := range self.info {
		sort.Sort(info.TaggedFiles)

		self.infoByName = append(self.infoByName, info)
		self.infoByCount = append(self.infoByCount, info)
	}

	sort.Sort(self.infoByName)
	sort.Sort(self.infoByCount)

	if self.indexMeta != nil {
		files, err := self.buildPages(context)
		if err != nil {
			return err
		}

		self.files = append(self.files, files...)
	}

	for _, file := range self.files {
		context.DispatchFile(file)
	}

	return nil
}

func (self *Tags) buildPages(context *goldsmith.Context) ([]*goldsmith.File, error) {
	var files []*goldsmith.File
	for tag, info := range self.info {
		var err error
		info.IndexFile, err = context.CreateFileFromReader(self.tagPagePath(tag), bytes.NewReader(nil))
		if err != nil {
			return nil, err
		}

		info.IndexFile.SetProp(self.stateKey, &TagState{
			CurrentTag:  info,
			TagsByName:  &self.infoByName,
			TagsByCount: &self.infoByCount,
		})

		for name, value := range self.indexMeta {
			info.IndexFile.SetProp(name, value)
		}

		files = append(files, info.IndexFile)
	}

	return files, nil
}

func (self *Tags) tagPagePath(tag string) string {
	return path.Join(self.baseDir, safeTag(tag), self.indexName)
}

func safeTag(tagRaw string) string {
	tagRaw = strings.TrimSpace(tagRaw)
	tagRaw = strings.ToLower(tagRaw)

	var valid bool
	var tagSafe string
	for _, c := range tagRaw {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			tagSafe += string(c)
			valid = true
		} else if valid {
			tagSafe += "-"
			valid = false
		}
	}

	return strings.Trim(tagSafe, "-")
}

type tagInfoByCount []*TagInfo

func (self tagInfoByCount) Len() int {
	return len(self)
}

func (self tagInfoByCount) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self tagInfoByCount) Less(i, j int) bool {
	if len(self[i].TaggedFiles) > len(self[j].TaggedFiles) {
		return true
	} else if len(self[i].TaggedFiles) == len(self[j].TaggedFiles) && strings.Compare(self[i].RawName, self[j].RawName) < 0 {
		return true
	}

	return false
}

type tagInfoByName []*TagInfo

func (self tagInfoByName) Len() int {
	return len(self)
}

func (self tagInfoByName) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self tagInfoByName) Less(i, j int) bool {
	if strings.Compare(self[i].RawName, self[j].RawName) < 0 {
		return true
	} else if self[i].RawName == self[j].RawName && len(self[i].TaggedFiles) > len(self[j].TaggedFiles) {
		return true
	}

	return false
}

type filesByPath []*goldsmith.File

func (self filesByPath) Len() int {
	return len(self)
}

func (self filesByPath) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self filesByPath) Less(i, j int) bool {
	return strings.Compare(self[i].Path(), self[j].Path()) < 0
}
