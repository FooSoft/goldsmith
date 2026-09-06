package livejs

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.Chain(New())
		},
	)
}

func TestCacheUsesInputDependency(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	targetDir := filepath.Join(root, "target")
	cacheDir := filepath.Join(root, "cache")
	if err := os.Mkdir(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("<html><body></body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	build := func() {
		gs := goldsmith.Goldsmith{CacheDir: cacheDir}
		if errs := gs.Begin(sourceDir).Chain(New()).End(targetDir); len(errs) > 0 {
			t.Fatal(errs)
		}
	}
	build()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d cache files, want 1", len(entries))
	}

	cachePath := filepath.Join(cacheDir, entries[0].Name())
	oldTime := time.Unix(1, 0)
	if err := os.Chtimes(cachePath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	build()
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatal("cache entry was needlessly rewritten")
	}
}
