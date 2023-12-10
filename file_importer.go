package goldsmith

import (
	"path/filepath"
)

type fileImporter struct {
	sourceDir string
}

func (*fileImporter) Name() string {
	return "importer"
}

func (self *fileImporter) Initialize(context *Context) error {
	infoChan := make(chan fileInfo)
	go scanDir(self.sourceDir, infoChan)

	for info := range infoChan {
		if info.IsDir() {
			continue
		}

		relPath, err := filepath.Rel(self.sourceDir, info.path)
		if err != nil {
			panic(err)
		}

		file, err := context.CreateFileFromAsset(relPath, info.path)
		if err != nil {
			return err
		}

		context.DispatchFile(file)
	}

	return nil
}
