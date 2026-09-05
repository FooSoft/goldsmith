package pager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/collection"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	lister := func(file *goldsmith.File) interface{} {
		if groupsRaw, ok := file.Prop("Groups"); ok {
			if groups, ok := groupsRaw.(map[string][]*goldsmith.File); ok {
				if group, ok := groups["group"]; ok {
					return group
				}
			}
		}

		return nil
	}

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(collection.New()).
				Chain(New(lister).ItemsPerPage(4)).
				Chain(layout.New())
		},
	)
}

func TestRejectsInvalidItemsPerPage(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	errs := gs.Begin(sourceDir).Chain(New(func(*goldsmith.File) interface{} {
		return []string{}
	}).ItemsPerPage(0)).End(filepath.Join(root, "target"))
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "items per page") {
		t.Fatalf("expected an items-per-page error, got %v", errs)
	}
}

func TestSliceCropRejectsOutOfBoundsRange(t *testing.T) {
	if _, err := sliceCrop([]string{"item"}, 0, 2); err == nil {
		t.Fatal("sliceCrop unexpectedly accepted an out-of-bounds range")
	}
}

func TestEmptyListKeepsOriginalPage(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nPagerEnable: true\n---\nempty"
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	errs := gs.Begin(sourceDir).
		Chain(frontmatter.New()).
		Chain(New(func(*goldsmith.File) interface{} { return []string{} })).
		End(targetDir)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	data, err := os.ReadFile(filepath.Join(targetDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "empty" {
		t.Fatalf("got %q, want %q", data, "empty")
	}
}

func TestGeneratedPagesRetainContent(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nPagerEnable: true\n---\nsame content"
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	errs := gs.Begin(sourceDir).
		Chain(frontmatter.New()).
		Chain(New(func(*goldsmith.File) interface{} { return []int{1, 2, 3} }).ItemsPerPage(1)).
		End(targetDir)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for _, name := range []string{"index.html", "index-2.html", "index-3.html"} {
		data, err := os.ReadFile(filepath.Join(targetDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "same content" {
			t.Fatalf("%s contains %q, want %q", name, data, "same content")
		}
	}
}
