package goldsmith

import (
	"os"
	"path/filepath"
	"sync"
)

type saver struct {
	clean bool

	tokens map[string]bool
	mutex  sync.Mutex
}

func (*saver) Name() string {
	return "saver"
}

func (saver *saver) Process(context *Context, file *File) error {
	saver.mutex.Lock()
	defer saver.mutex.Unlock()

	for token := cleanPath(file.sourcePath); token != "."; token = filepath.Dir(token) {
		saver.tokens[token] = true
	}

	return file.export(context.goldsmith.targetDir)
}

func (saver *saver) Finalize(context *Context) error {
	if !saver.clean {
		return nil
	}

	infos := make(chan fileInfo)
	go scanDir(context.goldsmith.targetDir, infos)

	for info := range infos {
		if info.path != context.goldsmith.targetDir {
			relPath, _ := filepath.Rel(context.goldsmith.targetDir, info.path)
			if contained, _ := saver.tokens[relPath]; !contained {
				os.RemoveAll(info.path)
			}
		}
	}

	return nil
}
