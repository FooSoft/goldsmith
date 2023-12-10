package goldsmith

import (
	"os"
	"path/filepath"
)

type fileInfo struct {
	os.FileInfo
	path string
}

func scanDir(dir string, infoChan chan fileInfo) {
	defer close(infoChan)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			infoChan <- fileInfo{FileInfo: info, path: path}
		}

		return err
	})
}
