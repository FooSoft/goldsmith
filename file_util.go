package goldsmith

import (
	"os"
	"path/filepath"
	"strings"
)

type filesByPath []*File

func (self filesByPath) Len() int {
	return len(self)
}

func (self filesByPath) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self filesByPath) Less(i, j int) bool {
	return strings.Compare(self[i].Path(), self[j].Path()) < 0
}

type fileInfo struct {
	os.FileInfo
	path string
}

func cleanPath(path string) string {
	if filepath.IsAbs(path) {
		var err error
		if path, err = filepath.Rel("/", path); err != nil {
			panic(err)
		}
	}

	return filepath.Clean(path)
}

func scanDir(rootDir string, infos chan fileInfo) {
	defer close(infos)

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			infos <- fileInfo{FileInfo: info, path: path}
		}

		return err
	})
}
