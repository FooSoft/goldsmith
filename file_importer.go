package goldsmith

import (
	"os"
	"path/filepath"
)

type fileImporter struct {
	sourceDir string
}

func (*fileImporter) Name() string {
	return "importer"
}

func (self *fileImporter) Initialize(context *Context) error {
	return filepath.Walk(self.sourceDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(self.sourceDir, path)
		if err != nil {
			panic(err)
		}

		file, err := context.CreateFileFromAsset(relPath, path)
		if err != nil {
			return err
		}

		context.DispatchFile(file)
		return nil
	})
}
