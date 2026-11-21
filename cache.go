package goldsmith

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type cache struct {
	baseDir string
}

func (self *cache) retrieveFile(context *Context, outputPath string, keys []string) (*File, error) {
	cachePath, err := self.buildCachePath(outputPath, context.cacheNamespace(), keys)
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

func (self *cache) storeFile(outputFile *File, namespace string, keys []string) (resultErr error) {
	cachePath, err := self.buildCachePath(outputFile.Path(), namespace, keys)
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

func (self *cache) buildCachePath(outputPath, namespace string, keys []string) (string, error) {
	if err := validateRelativePath(outputPath); err != nil {
		return "", err
	}
	outputPath = filepath.ToSlash(filepath.Clean(filepath.FromSlash(outputPath)))

	hasher := sha256.New()
	for _, value := range append([]string{namespace, outputPath}, keys...) {
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(value)))
		hasher.Write(size[:])
		hasher.Write([]byte(value))
	}

	cachePath := filepath.Join(self.baseDir, fmt.Sprintf(
		"gs_%x%s",
		hasher.Sum(nil)[:16],
		filepath.Ext(outputPath),
	))

	return cachePath, nil
}
