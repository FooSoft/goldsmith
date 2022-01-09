package goldsmith

import (
	"os"
	"path/filepath"
)

type saver struct {
	clean  bool
	tokens map[string]bool
}

func (*saver) Name() string {
	return "saver"
}

func (self *saver) Initialize(context *Context) error {
	self.tokens = make(map[string]bool)
	context.Threads(1)
	return nil
}

func (self *saver) Process(context *Context, file *File) error {
	for token := cleanPath(file.relPath); token != "."; token = filepath.Dir(token) {
		self.tokens[token] = true
	}

	return file.export(context.goldsmith.targetDir)
}

func (self *saver) Finalize(context *Context) error {
	if !self.clean {
		return nil
	}

	scannedInfo := make(chan fileInfo)
	go scanDir(context.goldsmith.targetDir, scannedInfo)

	for info := range scannedInfo {
		if info.path != context.goldsmith.targetDir {
			relPath, _ := filepath.Rel(context.goldsmith.targetDir, info.path)
			if contained, _ := self.tokens[relPath]; !contained {
				os.RemoveAll(info.path)
			}
		}
	}

	return nil
}
