package goldsmith

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateFileRejectsUnsafePaths(t *testing.T) {
	context := &Context{}
	if _, err := context.CreateFileFromReader("file.txt", nil); err == nil {
		t.Fatal("CreateFileFromReader unexpectedly accepted a nil reader")
	}
	for _, path := range []string{"", ".", "..", "../outside.txt", "safe/../../outside.txt"} {
		if _, err := context.CreateFileFromReader(path, strings.NewReader("data")); err == nil {
			t.Errorf("CreateFileFromReader(%q) unexpectedly succeeded", path)
		}
	}

	file, err := context.CreateFileFromReader("safe/file.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("CreateFileFromReader returned an error for a safe path: %v", err)
	}
	if err := file.Rename("../outside.txt"); err == nil {
		t.Fatal("Rename unexpectedly accepted a path outside the output root")
	}
	if file.Path() != "safe/file.txt" {
		t.Fatalf("Rename changed the path after failing: %q", file.Path())
	}
}

func TestCacheKeyDoesNotReorderInputs(t *testing.T) {
	inputs := []*File{
		{relPath: "z.txt", modTime: time.Unix(1, 0)},
		{relPath: "a.txt", modTime: time.Unix(2, 0)},
	}

	if _, err := (&cache{baseDir: t.TempDir()}).buildCachePath("out.txt", inputs); err != nil {
		t.Fatal(err)
	}
	if inputs[0].Path() != "z.txt" || inputs[1].Path() != "a.txt" {
		t.Fatalf("cache key generation reordered caller inputs: %q, %q", inputs[0].Path(), inputs[1].Path())
	}
}

type cacheWriterPlugin struct{}

func (*cacheWriterPlugin) Name() string { return "cache-writer" }

func (*cacheWriterPlugin) Process(context *Context, input *File) error {
	output, err := context.CreateFileFromReader(input.Path(), bytes.NewBufferString("output"))
	if err != nil {
		return err
	}
	context.DispatchAndCacheFile(output, input)
	return nil
}

func TestCacheWriteErrorsAreReported(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "input.txt"), []byte("input"), 0o644); err != nil {
		t.Fatal(err)
	}

	cachePath := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(cachePath, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	gs := Goldsmith{CacheDir: cachePath}
	errs := gs.Begin(sourceDir).Chain(&cacheWriterPlugin{}).End(targetDir)
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "[cache@input.txt]") {
		t.Fatalf("expected a cache write error, got %v", errs)
	}
}
