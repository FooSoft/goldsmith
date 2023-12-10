package goldsmith

import (
	"os"
	"path/filepath"
)

type fileExporter struct {
	targetDir string
	clean     bool
	tokens    map[string]bool
}

func (*fileExporter) Name() string {
	return "exporter"
}

func (self *fileExporter) Initialize(context *Context) error {
	self.tokens = make(map[string]bool)
	context.Threads(1)
	return nil
}

func (self *fileExporter) Process(context *Context, file *File) error {
	slicePath := func(path string) string {
		if filepath.IsAbs(path) {
			var err error
			if path, err = filepath.Rel("/", path); err != nil {
				panic(err)
			}
		}

		return filepath.Clean(path)
	}

	for token := slicePath(file.relPath); token != "."; token = filepath.Dir(token) {
		self.tokens[token] = true
	}

	return file.export(self.targetDir)
}

func (self *fileExporter) Finalize(context *Context) error {
	if !self.clean {
		return nil
	}

	infoChan := make(chan fileInfo)
	go scanDir(self.targetDir, infoChan)

	for info := range infoChan {
		if info.path == self.targetDir {
			continue
		}

		relPath, err := filepath.Rel(self.targetDir, info.path)
		if err != nil {
			panic(err)
		}

		if tokenized, _ := self.tokens[relPath]; !tokenized {
			if err := os.RemoveAll(info.path); err != nil {
				return err
			}
		}
	}

	return nil
}
