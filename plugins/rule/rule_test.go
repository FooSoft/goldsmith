package rule

import (
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	plugin := New()
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(plugin).
				Chain(frontmatter.New()).
				Chain(layout.New().DefaultLayout("page"))
		},
	)
}

func TestNestedRulesApplyInHierarchyOrder(t *testing.T) {
	context := new(goldsmith.Context)
	file, err := context.CreateFileFromReader("section/deep/page.html", strings.NewReader("content"))
	if err != nil {
		t.Fatal(err)
	}

	root := &ruleSet{Apply: []*ruleApply{{
		rule:  rule{Accept: []string{"**"}, baseDir: "."},
		Props: goldsmith.FileProps{"Title": "root"},
	}}}
	child := &ruleSet{Apply: []*ruleApply{{
		rule:  rule{Accept: []string{"**"}, baseDir: "section"},
		Props: goldsmith.FileProps{"Title": "child"},
	}}}
	drop := &ruleSet{Drop: []*ruleDrop{{
		rule: rule{Accept: []string{"**"}, baseDir: "section/deep"},
	}}}

	plugin := &Rule{
		ruleSets:   []*ruleSet{drop, child, root},
		inputFiles: []*goldsmith.File{file},
	}
	if err := plugin.Finalize(context); err != nil {
		t.Fatal(err)
	}
	if title, _ := file.Prop("Title"); title != "child" {
		t.Fatalf("got title %q, want %q", title, "child")
	}
}
