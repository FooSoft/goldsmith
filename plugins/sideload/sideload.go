package sideload

import (
	"bytes"
	"io/fs"
	"path/filepath"

	"git.foosoft.net/alex/goldsmith"
)

type (
	Sideload struct {
		files       []*goldsmith.File
		fileSystems []sideloadFs
	}

	sideloadFs interface {
		fs.ReadDirFS
		fs.ReadFileFS
	}
)

func New() *Sideload {
	return &Sideload{}
}

func (*Sideload) Name() string {
	return "sideload"
}

func (self *Sideload) Finalize(context *goldsmith.Context) error {
	files := self.files

	for _, fileSystem := range self.fileSystems {
		currFiles, err := self.gatherFsFiles(context, fileSystem, ".")
		if err != nil {
			return err
		}

		files = append(files, currFiles...)
	}

	for _, file := range files {
		context.DispatchFile(file)
	}

	return nil
}

func (self *Sideload) Files(files ...*goldsmith.File) *Sideload {
	self.files = append(self.files, files...)
	return self
}

func (self *Sideload) FileSystems(fileSystems ...fs.FS) *Sideload {
	for _, fileSystem := range fileSystems {
		self.fileSystems = append(self.fileSystems, fileSystem.(sideloadFs))
	}

	return self
}

func (self *Sideload) gatherFsFiles(context *goldsmith.Context, fileSystem sideloadFs, path string) ([]*goldsmith.File, error) {
	entries, err := fileSystem.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []*goldsmith.File
	for _, entry := range entries {
		currPath := filepath.Join(path, entry.Name())
		if entry.IsDir() {
			currFiles, err := self.gatherFsFiles(context, fileSystem, currPath)
			if err != nil {
				return nil, err
			}

			files = append(files, currFiles...)
		} else {
			data, err := fileSystem.ReadFile(currPath)
			if err != nil {
				return nil, err
			}

			file, err := context.CreateFileFromReader(currPath, bytes.NewReader(data))
			if err != nil {
				return nil, err
			}

			files = append(files, file)
		}
	}

	return files, nil
}
