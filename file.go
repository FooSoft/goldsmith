package goldsmith

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

type Prop interface{}
type PropMap map[string]Prop

// File represents in-memory or on-disk files in a chain.
type File struct {
	relPath string
	props   map[string]Prop
	modTime time.Time
	size    int64

	dataPath string
	reader   *bytes.Reader

	index int
}

// Rename modifies the file path relative to the source directory.
func (self *File) Rename(path string) {
	self.relPath = path
}

func (self *File) Rewrite(reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	self.reader = bytes.NewReader(data)
	self.modTime = time.Now()
	self.size = int64(len(data))
	return nil
}

// Path returns the file path relative to the source directory.
func (self *File) Path() string {
	return self.relPath
}

// Name returns the base name of the file.
func (self *File) Name() string {
	return path.Base(self.relPath)
}

// Dir returns the containing directory of the file.
func (self *File) Dir() string {
	return path.Dir(self.relPath)
}

// Ext returns the extension of the file.
func (self *File) Ext() string {
	return path.Ext(self.relPath)
}

// Size returns the file length in bytes.
func (self *File) Size() int64 {
	return self.size
}

// ModTime returns the time of the file's last modification.
func (self *File) ModTime() time.Time {
	return self.modTime
}

// Read reads file data into the provided buffer.
func (self *File) Read(data []byte) (int, error) {
	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.Read(data)
}

// Write writes file data into the provided writer.
func (self *File) WriteTo(writer io.Writer) (int64, error) {
	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.WriteTo(writer)
}

// Seek updates the file pointer to the desired position.
func (self *File) Seek(offset int64, whence int) (int64, error) {
	if self.reader == nil && offset == 0 && (whence == os.SEEK_SET || whence == os.SEEK_CUR) {
		return 0, nil
	}

	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.Seek(offset, whence)
}

// Returns value for string formatting.
func (self *File) GoString() string {
	return self.relPath
}

func (self *File) SetProp(name string, value Prop) {
	self.props[name] = value
}

func (self *File) CopyProps(file *File) {
	for key, value := range file.props {
		self.props[key] = value
	}
}

func (self *File) Prop(name string) (Prop, bool) {
	value, ok := self.props[name]
	return value, ok
}

func (self *File) Props() PropMap {
	return self.props
}

func (self *File) PropOrDefault(name string, valueDef Prop) Prop {
	if value, ok := self.Prop(name); ok {
		return value
	}

	return valueDef
}

func (self *File) export(targetDir string) error {
	targetPath := filepath.Join(targetDir, self.relPath)

	if targetInfo, err := os.Stat(targetPath); err == nil && !targetInfo.ModTime().Before(self.ModTime()) {
		return nil
	}

	if err := os.MkdirAll(path.Dir(targetPath), 0755); err != nil {
		return err
	}

	fw, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer fw.Close()

	if self.reader == nil {
		fr, err := os.Open(self.dataPath)
		if err != nil {
			return err
		}
		defer fr.Close()

		if _, err := io.Copy(fw, fr); err != nil {
			return err
		}
	} else {
		if _, err := self.Seek(0, os.SEEK_SET); err != nil {
			return err
		}

		if _, err := self.WriteTo(fw); err != nil {
			return err
		}
	}

	return nil
}

func (self *File) load() error {
	if self.reader != nil {
		return nil
	}

	data, err := ioutil.ReadFile(self.dataPath)
	if err != nil {
		return err
	}

	self.reader = bytes.NewReader(data)
	return nil
}
