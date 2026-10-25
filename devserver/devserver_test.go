package devserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestBuildRejectsNilCallback(t *testing.T) {
	if _, err := build(nil, nil); err == nil {
		t.Fatal("nil build callback was accepted")
	}
}

func TestBuildWatchesChangesAndNewDirectories(t *testing.T) {
	root := t.TempDir()
	builds := make(chan struct{}, 16)
	watcher, err := build([]string{root}, func() { builds <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	waitBuild := func() {
		t.Helper()
		select {
		case <-builds:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for build")
		}
	}
	waitBuild() // Initial build needs no file system event.
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(filepath.Join(root, "page.txt"), []byte{byte(i)}, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	waitBuild()
	select {
	case <-builds:
		t.Fatal("a burst of writes triggered multiple builds")
	case <-time.After(250 * time.Millisecond):
	}
	nested := filepath.Join(root, "new")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	waitBuild()
	if err := os.WriteFile(filepath.Join(nested, "page.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatal(err)
	}
	waitBuild()
}

func TestBuildRejectsMissingDirectory(t *testing.T) {
	watcher, err := build([]string{filepath.Join(t.TempDir(), "missing")}, func() {})
	if err == nil || watcher != nil {
		t.Fatalf("got watcher %v and error %v for a missing directory", watcher, err)
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
