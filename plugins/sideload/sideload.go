package sideload

import (
	"bytes"
	"io"
	"io/fs"
	"path"

	"git.sr.ht/~foosoft/goldsmith"
)

type (
	Sideload struct {
		files       []File
		fileSystems []sfs
	}

	File struct {
		Path   string
		Reader io.Reader
	}

	sfs interface {
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
	for _, fileSystem := range self.fileSystems {
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
	for _, fileSystem := range fileSystems {
		self.fileSystems = append(self.fileSystems, fileSystem.(sfs))
	}

	return self
}

func (self *Sideload) gatherFsFiles(context *goldsmith.Context, fileSystem sfs, searchPath string) ([]*goldsmith.File, error) {
	entries, err := fileSystem.ReadDir(searchPath)
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
