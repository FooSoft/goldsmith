package goldsmith

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
)

type cache struct {
	baseDir string
}

func (self *cache) retrieveFile(context *Context, outputPath string, inputFiles []*File) (*File, error) {
	cachePath, err := self.buildCachePath(context, outputPath, inputFiles)
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

func (self *cache) storeFile(context *Context, outputFile *File, inputFiles []*File) error {
	cachePath, err := self.buildCachePath(context, outputFile.Path(), inputFiles)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(self.baseDir, 0755); err != nil {
		return err
	}

	fp, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer fp.Close()

	offset, err := outputFile.Seek(0, os.SEEK_CUR)
	if err != nil {
		return err
	}

	if _, err := outputFile.Seek(0, os.SEEK_SET); err != nil {
		return err
	}

	if _, err := outputFile.WriteTo(fp); err != nil {
		return err
	}

	if _, err := outputFile.Seek(offset, os.SEEK_SET); err != nil {
		return err
	}

	return nil
}

func (self *cache) buildCachePath(context *Context, outputPath string, inputFiles []*File) (string, error) {
	hasher := crc32.NewIEEE()
	hasher.Write([]byte(outputPath))

	sort.Sort(filesByPath(inputFiles))

	for _, inputFile := range inputFiles {
		modTimeBuff := make([]byte, 8)
		binary.LittleEndian.PutUint64(modTimeBuff, uint64(inputFile.ModTime().UnixNano()))
		hasher.Write([]byte(inputFile.Path()))
		hasher.Write(modTimeBuff)
	}

	cachePath := filepath.Join(self.baseDir, fmt.Sprintf(
		"gs_%.8x%s",
		hasher.Sum32(),
		filepath.Ext(outputPath),
	))

	return cachePath, nil
}
