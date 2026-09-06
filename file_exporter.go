package goldsmith

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

type fileExporter struct {
	targetDir string
	clean     bool
	root      *os.Root
	files     map[string]struct{}
	tokens    map[string]struct{}
}

func (*fileExporter) Name() string {
	return "exporter"
}

func (self *fileExporter) Initialize(context *Context) error {
	self.files = make(map[string]struct{})
	self.tokens = make(map[string]struct{})
	context.Threads(1)
	return nil
}

func (self *fileExporter) Process(context *Context, file *File) error {
	if err := self.openRoot(true); err != nil {
		return err
	}

	filePath := path.Clean(file.Path())
	_, duplicate := self.files[filePath]
	self.files[filePath] = struct{}{}

	for token := filePath; token != "."; token = path.Dir(token) {
		self.tokens[token] = struct{}{}
	}

	rootPath := filepath.FromSlash(filePath)
	if targetInfo, err := self.root.Stat(rootPath); err == nil {
		if !duplicate && targetInfo.Mode().IsRegular() && targetInfo.Size() == file.Size() && !targetInfo.ModTime().Before(file.ModTime()) {
			return nil
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := self.root.MkdirAll(filepath.Dir(rootPath), 0o755); err != nil {
		return err
	}

	fw, tempPath, err := createRootTemp(self.root)
	if err != nil {
		return err
	}
	defer self.root.Remove(tempPath)

	_, writeErr := file.Seek(0, io.SeekStart)
	if writeErr == nil {
		_, writeErr = file.WriteTo(fw)
	}
	closeErr := fw.Close()

	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return self.root.Rename(tempPath, rootPath)
}

func (self *fileExporter) Finalize(context *Context) error {
	if self.root == nil {
		if !self.clean {
			return nil
		}
		if err := self.openRoot(false); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
	}

	var finalizeErr error
	if self.clean {
		finalizeErr = fs.WalkDir(self.root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == "." {
				return nil
			}

			if _, tokenized := self.tokens[path]; !tokenized {
				if err := self.root.RemoveAll(filepath.FromSlash(path)); err != nil {
					return err
				}
				if entry.IsDir() {
					return fs.SkipDir
				}
			}

			return nil
		})
	}
	closeErr := self.root.Close()
	self.root = nil
	if finalizeErr != nil {
		return finalizeErr
	}
	return closeErr
}

func (self *fileExporter) openRoot(create bool) error {
	if self.root != nil {
		return nil
	}
	if create {
		if err := os.MkdirAll(self.targetDir, 0o755); err != nil {
			return err
		}
	}

	root, err := os.OpenRoot(self.targetDir)
	if err != nil {
		return err
	}
	self.root = root
	return nil
}

func createRootTemp(root *os.Root) (*os.File, string, error) {
	for range 100 {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return nil, "", err
		}
		name := fmt.Sprintf(".goldsmith-%x", random)
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			return file, name, nil
		}
		if !os.IsExist(err) {
			return nil, "", err
		}
	}

	return nil, "", fmt.Errorf("could not create temporary output file")
}
