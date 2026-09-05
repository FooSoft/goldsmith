package harness

import (
	"os"
	"path/filepath"
	"testing"
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
