package goldsmith

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type rawFile struct {
	relPath string
	props   map[string]FileProp
	modTime time.Time
	size    int64

	dataPath string
	reader   *bytes.Reader
}

func (self *rawFile) Path() string {
	return filepath.ToSlash(self.relPath)
}

func (self *rawFile) Dir() string {
	return filepath.ToSlash(filepath.Dir(self.relPath))
}

func (self *rawFile) Name() string {
	return filepath.Base(self.relPath)
}

func (self *rawFile) Ext() string {
	return filepath.Ext(self.relPath)
}

func (self *rawFile) Rename(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("unexpected absolute path: %s", path)
	}

	self.relPath = path
	return nil
}

func (self *rawFile) Size() int64 {
	return self.size
}

func (self *rawFile) ModTime() time.Time {
	return self.modTime
}

func (self *rawFile) Read(data []byte) (int, error) {
	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.Read(data)
}

func (self *rawFile) WriteTo(writer io.Writer) (int64, error) {
	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.WriteTo(writer)
}

func (self *rawFile) Seek(offset int64, whence int) (int64, error) {
	if self.reader == nil && offset == 0 && (whence == io.SeekStart || whence == io.SeekCurrent) {
		return 0, nil
	}

	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.Seek(offset, whence)
}

func (self *rawFile) GoString() string {
	return self.relPath
}

func (self *rawFile) SetProp(name string, value FileProp) {
	self.props[name] = value
}

func (self *rawFile) Prop(name string) (FileProp, bool) {
	value, ok := self.props[name]
	return value, ok
}

func (self *rawFile) PropOrDef(name string, valueDef FileProp) FileProp {
	if value, ok := self.Prop(name); ok {
		return value
	}

	return valueDef
}

func (self *rawFile) Props() FileProps {
	return self.props
}

func (self *rawFile) CopyProps(file File) {
	for key, value := range file.Props() {
		self.props[key] = value
	}
}

func (self *rawFile) RemoveProp(name string) {
	delete(self.props, name)
}

func (self *rawFile) load() error {
	if self.reader != nil {
		return nil
	}

	data, err := os.ReadFile(self.dataPath)
	if err != nil {
		return err
	}

	self.reader = bytes.NewReader(data)
	return nil
}
