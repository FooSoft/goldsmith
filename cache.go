package goldsmith

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type cache struct {
	baseDir string
}

func (self *cache) retrieveFile(context *Context, outputPath string, inputFiles []*File) (*File, error) {
	cachePath, err := self.buildCachePath(outputPath, inputFiles)
	if err != nil {
		return nil, err
	}

	outputFile, err := context.CreateFileFromAsset(outputPath, cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	return outputFile, nil
}

func (self *cache) storeFile(outputFile *File, inputFiles []*File) (resultErr error) {
	cachePath, err := self.buildCachePath(outputFile.Path(), inputFiles)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(self.baseDir, 0755); err != nil {
		return err
	}

	fp, err := os.CreateTemp(self.baseDir, ".goldsmith-*")
	if err != nil {
		return err
	}
	tempPath := fp.Name()
	defer os.Remove(tempPath)

	offset, err := outputFile.Seek(0, io.SeekCurrent)
	if err != nil {
		fp.Close()
		return err
	}
	defer func() {
		if _, err := outputFile.Seek(offset, io.SeekStart); resultErr == nil && err != nil {
			resultErr = err
		}
	}()

	if _, err := outputFile.Seek(0, io.SeekStart); err != nil {
		fp.Close()
		return err
	}

	if _, err := outputFile.WriteTo(fp); err != nil {
		fp.Close()
		return err
	}
	if err := fp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, cachePath); err != nil {
		return err
	}

	return nil
}

func (self *cache) buildCachePath(outputPath string, inputFiles []*File) (string, error) {
	if err := validateRelativePath(outputPath); err != nil {
		return "", err
	}
	outputPath = filepath.ToSlash(filepath.Clean(filepath.FromSlash(outputPath)))

	hasher := sha256.New()
	writeString := func(value string) {
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(value)))
		hasher.Write(size[:])
		hasher.Write([]byte(value))
	}
	writeString(outputPath)

	inputs := append([]*File(nil), inputFiles...)
	sort.Slice(inputs, func(i, j int) bool {
		if inputs[i] == nil {
			return inputs[j] != nil
		}
		if inputs[j] == nil {
			return false
		}
		if inputs[i].Path() != inputs[j].Path() {
			return inputs[i].Path() < inputs[j].Path()
		}
		if !inputs[i].ModTime().Equal(inputs[j].ModTime()) {
			return inputs[i].ModTime().Before(inputs[j].ModTime())
		}
		return inputs[i].Size() < inputs[j].Size()
	})

	for _, inputFile := range inputs {
		if inputFile == nil {
			return "", fmt.Errorf("cache dependency cannot be nil")
		}

		var fileInfo [16]byte
		binary.LittleEndian.PutUint64(fileInfo[:8], uint64(inputFile.ModTime().UnixNano()))
		binary.LittleEndian.PutUint64(fileInfo[8:], uint64(inputFile.Size()))
		writeString(inputFile.Path())
		hasher.Write(fileInfo[:])
	}

	cachePath := filepath.Join(self.baseDir, fmt.Sprintf(
		"gs_%x%s",
		hasher.Sum(nil)[:16],
		filepath.Ext(outputPath),
	))

	return cachePath, nil
}
