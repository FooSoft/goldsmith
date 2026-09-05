package goldsmith

import (
	"io"
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

	for token := slicePath(file.Path()); token != "."; token = filepath.Dir(token) {
		self.tokens[token] = true
	}

	targetPath := filepath.Join(self.targetDir, file.Path())
	if targetInfo, err := os.Stat(targetPath); err == nil && !targetInfo.ModTime().Before(file.ModTime()) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	fw, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		fw.Close()
		return err
	}

	if _, err := file.WriteTo(fw); err != nil {
		fw.Close()
		return err
	}

	return fw.Close()
}

func (self *fileExporter) Finalize(context *Context) error {
	if !self.clean {
		return nil
	}

	return filepath.Walk(self.targetDir, func(path string, info os.FileInfo, err error) error {
		if path == self.targetDir {
			return nil
		}

		relPath, err := filepath.Rel(self.targetDir, path)
		if err != nil {
			panic(err)
		}

		if tokenized, _ := self.tokens[relPath]; !tokenized {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}

		return nil
	})
}
