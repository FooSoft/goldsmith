// Package harness provides a simple way to test goldsmith plugins and filters.
// It executes a goldsmith chain on provided "source" data and compares the
// generated "target" results with the known-good "reference" data.
package harness

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"go.foosoft.net/goldsmith"
)

// StagingCallback callback function is used to set up a goldsmith chain.
type StagingCallback func(gs *goldsmith.Goldsmith)

// Validate enables validation of a single, unnamed case (test data is stored in "testdata").
func Validate(t *testing.T, stager StagingCallback) {
	t.Helper()
	ValidateCase(t, "", stager)
}

// ValidateCase enables validation of a single, named case (test data is stored in "testdata/caseName").
func ValidateCase(t *testing.T, caseName string, stager StagingCallback) {
	t.Helper()
	tempDir := t.TempDir()
	var (
		caseDir      = filepath.Join("testdata", caseName)
		sourceDir    = filepath.Join(caseDir, "source")
		targetDir    = filepath.Join(tempDir, "target")
		cacheDir     = filepath.Join(tempDir, "cache")
		referenceDir = filepath.Join(caseDir, "reference")
	)

	if errs := validate(sourceDir, targetDir, cacheDir, referenceDir, stager); len(errs) > 0 {
		for _, err := range errs {
			t.Error(err)
		}
	}
}

func validate(sourceDir, targetDir, cacheDir, referenceDir string, stager StagingCallback) []error {
	for i := 0; i < 2; i++ {
		if errs := execute(sourceDir, targetDir, cacheDir, stager); errs != nil {
			return errs
		}

		targetHash, err := hashDirState(targetDir)
		if err != nil {
			return []error{err}
		}
		referenceHash, err := hashDirState(referenceDir)
		if err != nil {
			return []error{err}
		}
		if targetHash != referenceHash {
			return []error{errors.New("directory contents do not match")}
		}
	}

	return nil
}

func execute(sourceDir, targetDir, cacheDir string, stager StagingCallback) []error {
	gs := goldsmith.Goldsmith{CacheDir: cacheDir, Clean: true}
	gs.Begin(sourceDir)
	stager(&gs)
	return gs.End(targetDir)
}

func hashDirState(dir string) ([sha256.Size]byte, error) {
	hasher := sha256.New()
	writeUint64 := func(value uint64) {
		var data [8]byte
		binary.LittleEndian.PutUint64(data[:], value)
		hasher.Write(data[:])
	}
	writeString := func(value string) {
		writeUint64(uint64(len(value)))
		hasher.Write([]byte(value))
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if path == dir && os.IsNotExist(err) {
				return nil
			}
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if relPath == "." && info.IsDir() {
			return nil
		}

		writeString(relPath)
		if info.IsDir() {
			hasher.Write([]byte{1})
		} else {
			hasher.Write([]byte{0})
			writeUint64(uint64(info.Size()))
		}

		if !info.IsDir() {
			fp, err := os.Open(path)
			if err != nil {
				return err
			}

			if _, err := io.Copy(hasher, fp); err != nil {
				fp.Close()
				return err
			}
			if err := fp.Close(); err != nil {
				return err
			}
		}

		return nil
	})

	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result, err
}
