package devserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestBuildRejectsNilCallback(t *testing.T) {
	if err := build(nil, nil); err == nil {
		t.Fatal("nil build callback was accepted")
	}
}

func TestWatchAddsDirectoriesOnly(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	if err := watch(root, watcher); err != nil {
		t.Fatal(err)
	}

	watched := make(map[string]bool)
	for _, path := range watcher.WatchList() {
		watched[filepath.Clean(path)] = true
	}
	if len(watched) != 2 || !watched[filepath.Clean(root)] || !watched[filepath.Clean(nested)] {
		t.Fatalf("unexpected watch list: %v", watcher.WatchList())
	}
}
