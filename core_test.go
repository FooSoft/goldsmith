package goldsmith

import (
	"bytes"
	"errors"
	"io"
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

func TestCreateFileNormalizesPaths(t *testing.T) {
	context := &Context{}
	file, err := context.CreateFileFromReader("safe/../file.txt", strings.NewReader("data"))
	if err != nil {
		t.Fatal(err)
	}
	if file.Path() != "file.txt" {
		t.Fatalf("created file path was not normalized: %q", file.Path())
	}

	if err := file.Rename("dir/./renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if file.Path() != "dir/renamed.txt" {
		t.Fatalf("renamed file path was not normalized: %q", file.Path())
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

func TestCacheKeyIncludesFileSize(t *testing.T) {
	modTime := time.Unix(1, 0)
	cache := &cache{baseDir: t.TempDir()}
	first, err := cache.buildCachePath("out.txt", []*File{{relPath: "input.txt", modTime: modTime, size: 1}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := cache.buildCachePath("out.txt", []*File{{relPath: "input.txt", modTime: modTime, size: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("cache key did not change with the input size")
	}

	if _, err := cache.buildCachePath("out.txt", []*File{nil}); err == nil {
		t.Fatal("nil cache dependency was accepted")
	}
}

func TestCacheKeyIgnoresDependencyOrder(t *testing.T) {
	first := &File{relPath: "input.txt", modTime: time.Unix(1, 0), size: 1}
	second := &File{relPath: "input.txt", modTime: time.Unix(2, 0), size: 2}
	cache := &cache{baseDir: t.TempDir()}
	forward, err := cache.buildCachePath("out.txt", []*File{first, second})
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := cache.buildCachePath("out.txt", []*File{second, first})
	if err != nil {
		t.Fatal(err)
	}
	if forward != reverse {
		t.Fatalf("dependency order changed the cache key: %q != %q", forward, reverse)
	}
}

func TestCacheStorePreservesFileOffset(t *testing.T) {
	context := new(Context)
	file, err := context.CreateFileFromReader("out.txt", strings.NewReader("output"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(3, io.SeekStart); err != nil {
		t.Fatal(err)
	}

	if err := (&cache{baseDir: t.TempDir()}).storeFile(file, nil); err != nil {
		t.Fatal(err)
	}
	if offset, err := file.Seek(0, io.SeekCurrent); err != nil || offset != 3 {
		t.Fatalf("cache store left offset at %d with error %v", offset, err)
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

type completionPlugin struct {
	complete chan struct{}
}

func (*completionPlugin) Name() string { return "completion" }

func (plugin *completionPlugin) Finalize(context *Context) error {
	close(plugin.complete)
	return nil
}

type failingInitializerPlugin struct{}

func (*failingInitializerPlugin) Name() string { return "failing-initializer" }

func (*failingInitializerPlugin) Initialize(context *Context) error {
	return errors.New("initialization failed")
}

func TestInitializerErrorDrainsUpstream(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "input.txt"), []byte("input"), 0o644); err != nil {
		t.Fatal(err)
	}

	completion := &completionPlugin{complete: make(chan struct{})}
	var gs Goldsmith
	errs := gs.Begin(sourceDir).
		Chain(completion).
		Chain(new(failingInitializerPlugin)).
		End(filepath.Join(root, "target"))
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "initialization failed") {
		t.Fatalf("expected an initialization error, got %v", errs)
	}
	select {
	case <-completion.complete:
	default:
		t.Fatal("End returned before upstream finalization completed")
	}
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

func TestExporterReportsDirectoryAtFilePath(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(targetDir, "input.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "input.txt"), []byte("input"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(targetDir, "input.txt"), future, future); err != nil {
		t.Fatal(err)
	}

	var gs Goldsmith
	errs := gs.Begin(sourceDir).End(targetDir)
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "input.txt") {
		t.Fatalf("expected an exporter error, got %v", errs)
	}
}

func TestExporterReplacesNewerFileWithWrongSize(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "input.txt"), []byte("correct"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(targetDir, "input.txt")
	if err := os.WriteFile(targetPath, []byte("wrong size"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(targetPath, future, future); err != nil {
		t.Fatal(err)
	}

	var gs Goldsmith
	if errs := gs.Begin(sourceDir).End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "correct" {
		t.Fatalf("got output %q, want %q", data, "correct")
	}
}

func TestExporterUsesLastDuplicatePath(t *testing.T) {
	context := new(Context)
	first, err := context.CreateFileFromReader("file.txt", strings.NewReader("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := context.CreateFileFromReader("file.txt", strings.NewReader("other"))
	if err != nil {
		t.Fatal(err)
	}

	exporter := &fileExporter{targetDir: t.TempDir()}
	if err := exporter.Initialize(context); err != nil {
		t.Fatal(err)
	}
	defer exporter.Finalize(context)
	if err := exporter.Process(context, first); err != nil {
		t.Fatal(err)
	}
	if err := exporter.Process(context, second); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(exporter.targetDir, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "other" {
		t.Fatalf("got output %q, want the final duplicate", data)
	}
}

func TestExporterCleansStaleDirectory(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "current.txt"), []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(targetDir, "stale", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "stale", "nested", "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	gs := Goldsmith{Clean: true}
	if errs := gs.Begin(sourceDir).End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "stale")); !os.IsNotExist(err) {
		t.Fatalf("stale directory still exists or could not be checked: %v", err)
	}
}

func TestExporterPreservesExistingFileAfterReadError(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "target")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(targetDir, "file.txt")
	if err := os.WriteFile(targetPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(targetPath, past, past); err != nil {
		t.Fatal(err)
	}

	dataPath := filepath.Join(root, "data.txt")
	if err := os.WriteFile(dataPath, []byte("replacement"), 0o644); err != nil {
		t.Fatal(err)
	}
	context := new(Context)
	file, err := context.CreateFileFromAsset("file.txt", dataPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dataPath); err != nil {
		t.Fatal(err)
	}

	exporter := &fileExporter{targetDir: targetDir}
	if err := exporter.Initialize(context); err != nil {
		t.Fatal(err)
	}
	defer exporter.Finalize(context)
	if err := exporter.Process(context, file); err == nil {
		t.Fatal("export unexpectedly succeeded after its input was removed")
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("existing output was changed to %q", data)
	}
}

func TestExporterDoesNotFollowEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "target")
	outsideDir := filepath.Join(root, "outside")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(targetDir, "escape")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	context := new(Context)
	file, err := context.CreateFileFromReader("escape/file.txt", strings.NewReader("content"))
	if err != nil {
		t.Fatal(err)
	}
	exporter := &fileExporter{targetDir: targetDir}
	if err := exporter.Initialize(context); err != nil {
		t.Fatal(err)
	}
	defer exporter.Finalize(context)
	if err := exporter.Process(context, file); err == nil {
		t.Fatal("export unexpectedly followed a symlink outside its target root")
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "file.txt")); !os.IsNotExist(err) {
		t.Fatalf("file was written outside the target root: %v", err)
	}
}

func TestEndRejectsOverlappingDirectories(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "input.txt"), []byte("input"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gs Goldsmith
	errs := gs.Begin(sourceDir).End(filepath.Join(sourceDir, "output"))
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "directories overlap") {
		t.Fatalf("expected an overlapping-directory error, got %v", errs)
	}
	if _, err := os.Stat(filepath.Join(sourceDir, "output")); !os.IsNotExist(err) {
		t.Fatalf("overlapping output directory was created: %v", err)
	}
}

func TestEndBeforeBeginReturnsError(t *testing.T) {
	var gs Goldsmith
	if errs := gs.End(t.TempDir()); len(errs) != 1 || !strings.Contains(errs[0].Error(), "Begin") {
		t.Fatalf("expected a missing-Begin error, got %v", errs)
	}
}
