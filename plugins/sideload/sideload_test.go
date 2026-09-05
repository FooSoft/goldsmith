package sideload

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"embed"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

//go:embed testdata/fs/source
var embedFs embed.FS

func TestFs(self *testing.T) {
	harness.ValidateCase(
		self,
		"fs",
		func(gs *goldsmith.Goldsmith) {
			subFs, err := fs.Sub(embedFs, "testdata/fs/source")
			if err != nil {
				panic(err)
			}

			gs.Chain(New().FileSystems(subFs))
		},
	)
}

type openOnlyFS struct{ fs.FS }

func TestGenericFileSystem(t *testing.T) {
	fileSystem := openOnlyFS{fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("hello world")},
	}}

	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	if errs := gs.Begin(sourceDir).Chain(New().FileSystems(fileSystem)).End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
	data, err := os.ReadFile(filepath.Join(targetDir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Fatalf("got %q, want %q", data, "hello world")
	}
}

func TestNilFileSystemReturnsError(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gs := new(goldsmith.Goldsmith)
	errs := gs.Begin(sourceDir).Chain(New().FileSystems(nil)).End(filepath.Join(root, "target"))
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "file system cannot be nil") {
		t.Fatalf("expected a nil file system error, got %v", errs)
	}
}

func TestFile(self *testing.T) {
	harness.ValidateCase(
		self,
		"file",
		func(gs *goldsmith.Goldsmith) {
			file := File{
				Path:   "hello.txt",
				Reader: strings.NewReader("hello world"),
			}

			gs.Chain(New().Files(file))
		},
	)
}
