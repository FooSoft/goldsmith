package harness

import (
	"os"
	"path/filepath"
	"testing"

	"go.foosoft.net/goldsmith"
)

func TestHashDirStateMatchesEquivalentDirectories(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	for _, dir := range []string{first, second} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	firstHash, err := hashDirState(first)
	if err != nil {
		t.Fatal(err)
	}
	secondHash, err := hashDirState(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstHash != secondHash {
		t.Fatalf("equivalent directories hashed differently: %d != %d", firstHash, secondHash)
	}
}

func TestHashDirStateDetectsChanges(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	hash := func() [32]byte {
		t.Helper()
		value, err := hashDirState(root)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	if err := os.WriteFile(file, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := hash()
	if err := os.WriteFile(file, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	second := hash()
	if first == second {
		t.Fatal("same-size content change was not detected")
	}
	if err := os.Rename(file, filepath.Join(root, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	if second == hash() {
		t.Fatal("rename was not detected")
	}
}

func TestHashDirStateTreatsMissingAndEmptyRootsEqually(t *testing.T) {
	root := t.TempDir()
	empty, err := hashDirState(root)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := hashDirState(filepath.Join(root, "missing"))
	if err != nil {
		t.Fatal(err)
	}
	if empty != missing {
		t.Fatal("empty output differs from missing empty reference directory")
	}
}

func TestValidateBuildsTwiceAndDetectsMismatch(t *testing.T) {
	root := t.TempDir()
	source, target, cache := filepath.Join(root, "source"), filepath.Join(root, "target"), filepath.Join(root, "cache")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "file.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	calls := 0
	stage := func(*goldsmith.Goldsmith) { calls++ }
	if errs := validate(source, target, cache, source, stage); len(errs) > 0 {
		t.Fatal(errs)
	}
	if calls != 2 {
		t.Fatalf("staged %d builds, want 2", calls)
	}
	if errs := validate(source, target, cache, filepath.Join(root, "missing"), stage); len(errs) != 1 {
		t.Fatalf("expected mismatch, got %v", errs)
	}
	if errs := validate(filepath.Join(root, "missing"), target, cache, source, stage); len(errs) != 1 {
		t.Fatalf("expected import error, got %v", errs)
	}
}
