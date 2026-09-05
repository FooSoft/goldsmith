package sideload

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"path"

	"go.foosoft.net/goldsmith"
)

type (
	Sideload struct {
		files       []File
		fileSystems []fs.FS
	}

	File struct {
		Path   string
		Reader io.Reader
	}
)

func New() *Sideload {
	return &Sideload{}
}

func (*Sideload) Name() string {
	return "sideload"
}

func (self *Sideload) Finalize(context *goldsmith.Context) error {
	for _, fileSystem := range self.fileSystems {
		if fileSystem == nil {
			return errors.New("sideload file system cannot be nil")
		}
		files, err := self.gatherFsFiles(context, fileSystem, ".")
		if err != nil {
			return err
		}

		for _, file := range files {
			context.DispatchFile(file)
		}
	}

	for _, file := range self.files {
		gsFile, err := context.CreateFileFromReader(file.Path, file.Reader)
		if err != nil {
			return err
		}

		context.DispatchFile(gsFile)
	}

	return nil
}

func (self *Sideload) Files(files ...File) *Sideload {
	self.files = append(self.files, files...)
	return self
}

func (self *Sideload) FileSystems(fileSystems ...fs.FS) *Sideload {
	self.fileSystems = append(self.fileSystems, fileSystems...)
	return self
}

func (self *Sideload) gatherFsFiles(context *goldsmith.Context, fileSystem fs.FS, searchPath string) ([]*goldsmith.File, error) {
	entries, err := fs.ReadDir(fileSystem, searchPath)
	if err != nil {
		return nil, err
	}

	var files []*goldsmith.File
	for _, entry := range entries {
		currPath := path.Join(searchPath, entry.Name())
		if entry.IsDir() {
			currFiles, err := self.gatherFsFiles(context, fileSystem, currPath)
			if err != nil {
				return nil, err
			}

			files = append(files, currFiles...)
		} else {
			data, err := fs.ReadFile(fileSystem, currPath)
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
