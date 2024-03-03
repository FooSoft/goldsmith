// Package forward allows to create simple redirections for pages that have moved to a new URL.
package forward

import (
	"bytes"

	"git.foosoft.net/alex/goldsmith"
)

// Forward plugin context.
type Forward struct {
	sourceProps map[string]interface{}
	pathMap     map[string]string
	sourceKey   string
	targetKey   string
}

// New creates a new instance of the Forward plugin.
func New(sourceProps map[string]interface{}) *Forward {
	return &Forward{
		sourceProps: sourceProps,
		pathMap:     make(map[string]string),
		sourceKey:   "PathOld",
		targetKey:   "PathNew",
	}
}

// AddPathMapping adds a single path mapping between an old path and a new path.
func (self *Forward) AddPathMapping(sourcePath, targetPath string) *Forward {
	self.pathMap[sourcePath] = targetPath
	return self
}

// PathMap sets multiple path mappings between old paths and new paths.
func (self *Forward) PathMap(pathMap map[string]string) *Forward {
	self.pathMap = pathMap
	return self
}

// SourceKey sets the metadata key used to access the old path (default: "PathOld").
func (self *Forward) SourceKey(key string) *Forward {
	self.sourceKey = key
	return self
}

// SourceKey sets the metadata key used to access the new path (default: "PathNew").
func (self *Forward) TargetKey(key string) *Forward {
	self.targetKey = key
	return self
}

func (*Forward) Name() string {
	return "forward"
}

func (self *Forward) Initialize(context goldsmith.Context) error {
	for sourcePath, targetPath := range self.pathMap {
		sourceFile, err := context.CreateFileFromReader(sourcePath, bytes.NewReader(nil))
		if err != nil {
			return err
		}

		for name, value := range self.sourceProps {
			sourceFile.SetProp(name, value)
		}

		sourceFile.SetProp(self.targetKey, targetPath)
		context.DispatchFile(sourceFile)
	}

	return nil
}
