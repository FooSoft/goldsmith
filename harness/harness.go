// Package harness provides a simple way to test goldsmith plugins and filters.
// It executes a goldsmith chain on provided "source" data and compares the
// generated "target" resuts with the known to be good "reference" data.
package harness

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"git.sr.ht/~foosoft/goldsmith"
)

// StagingCallback callback function is used to set up a goldsmith chain.
type StagingCallback func(gs *goldsmith.Goldsmith)

// Validate enables validation of a single, unnamed case (test data is stored in "testdata").
func Validate(t *testing.T, stager StagingCallback) {
	ValidateCase(t, "", stager)
}

// ValidateCase enables enables of a single, named case (test data is stored in "testdata/caseName").
func ValidateCase(t *testing.T, caseName string, stager StagingCallback) {
	var (
		caseDir      = filepath.Join("testdata", caseName)
		sourceDir    = filepath.Join(caseDir, "source")
		targetDir    = filepath.Join(caseDir, "target")
		cacheDir     = filepath.Join(caseDir, "cache")
		referenceDir = filepath.Join(caseDir, "reference")
	)

	if errs := validate(sourceDir, targetDir, cacheDir, referenceDir, stager); len(errs) > 0 {
		for _, err := range errs {
			log.Println(err)
		}

		t.Fail()
	}
}

func validate(sourceDir, targetDir, cacheDir, referenceDir string, stager StagingCallback) []error {
	if err := os.RemoveAll(targetDir); err != nil {
		return []error{err}
	}

	if err := os.RemoveAll(cacheDir); err != nil {
		return []error{err}
	}

	defer os.RemoveAll(cacheDir)

	for i := 0; i < 2; i++ {
		if errs := execute(sourceDir, targetDir, cacheDir, stager); errs != nil {
			return errs
		}

		if hashDirState(targetDir) != hashDirState(referenceDir) {
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

func hashDirState(dir string) uint32 {
	hasher := crc32.NewIEEE()

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		summary := fmt.Sprintf("%s %t", relPath, info.IsDir())
		if _, err := hasher.Write([]byte(summary)); err != nil {
			return err
		}

		if !info.IsDir() {
			fp, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fp.Close()

			if _, err := io.Copy(hasher, fp); err != nil {
				return err
			}
		}

		return nil
	})

	return hasher.Sum32()
}
