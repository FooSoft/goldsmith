// Package breadcrumbs generates metadata required to enable breadcrumb
// navigation. This is particularly helpful for sites that have deep
// hierarchies which may be otherwise confusing to visitors.
package breadcrumbs

import (
	"fmt"
	"sort"
	"sync"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// Crumb provides organizational information about this node and ones before it.
type Crumb struct {
	Ancestors []*Node
	Node      *Node
}

// Node represents information about a specific file in the site's structure.
type Node struct {
	File     *goldsmith.File
	Parent   *Node
	Children []*Node

	parentName string
}

// Breadcrumbs chainable plugin context.
type Breadcrumbs struct {
	nameKey   string
	parentKey string
	crumbsKey string

	allNodes   []*Node
	namedNodes map[string]*Node
	mutex      sync.Mutex
}

// New creates a new instance of the Breadcrumbs plugin.
func New() *Breadcrumbs {
	return &Breadcrumbs{
		nameKey:    "CrumbName",
		parentKey:  "CrumbParent",
		crumbsKey:  "Crumbs",
		namedNodes: make(map[string]*Node),
	}
}

// NameKey sets the metadata key used to access the crumb name (default: "CrumbName").
// Crumb names must be globally unique within any given website.
func (self *Breadcrumbs) NameKey(key string) *Breadcrumbs {
	self.nameKey = key
	return self
}

// ParentKey sets the metadata key used to access the parent name (default: "CrumbParent").
func (self *Breadcrumbs) ParentKey(key string) *Breadcrumbs {
	self.parentKey = key
	return self
}

// CrumbsKey sets the metadata key used to store information about crumbs (default: "Crumbs").
func (self *Breadcrumbs) CrumbsKey(key string) *Breadcrumbs {
	self.crumbsKey = key
	return self
}

func (*Breadcrumbs) Name() string {
	return "breadcrumbs"
}

func (self *Breadcrumbs) Initialize(context *goldsmith.Context) error {
	self.allNodes = nil
	self.namedNodes = make(map[string]*Node)
	context.Filter(wildcard.New("**/*.html", "**/*.htm"))
	return nil
}

func (self *Breadcrumbs) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	var parentNameStr string
	if parentName, ok := inputFile.Prop(self.parentKey); ok {
		parentNameStr, _ = parentName.(string)
	}

	var nodeNameStr string
	if nodeName, ok := inputFile.Prop(self.nameKey); ok {
		nodeNameStr, _ = nodeName.(string)
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()

	node := &Node{File: inputFile, parentName: parentNameStr}
	self.allNodes = append(self.allNodes, node)

	if len(nodeNameStr) > 0 {
		if _, ok := self.namedNodes[nodeNameStr]; ok {
			return fmt.Errorf("duplicate node: %s", nodeNameStr)
		}

		self.namedNodes[nodeNameStr] = node
	}

	return nil
}

func (self *Breadcrumbs) Finalize(context *goldsmith.Context) error {
	sort.Slice(self.allNodes, func(i, j int) bool {
		return self.allNodes[i].File.Path() < self.allNodes[j].File.Path()
	})

	for _, node := range self.allNodes {
		if len(node.parentName) == 0 {
			continue
		}

		if parentNode, ok := self.namedNodes[node.parentName]; ok {
			parentNode.Children = append(parentNode.Children, node)
			node.Parent = parentNode
		} else {
			return fmt.Errorf("undefined parent: %s", node.parentName)
		}
	}

	for _, node := range self.allNodes {
		var ancestors []*Node
		seen := map[*Node]bool{node: true}
		for currentNode := node.Parent; currentNode != nil; currentNode = currentNode.Parent {
			if seen[currentNode] {
				return fmt.Errorf("breadcrumb cycle involving %s", node.File.Path())
			}
			seen[currentNode] = true
			ancestors = append([]*Node{currentNode}, ancestors...)
		}

		node.File.SetProp(self.crumbsKey, Crumb{ancestors, node})
		context.DispatchFile(node.File)
	}

	return nil
}
