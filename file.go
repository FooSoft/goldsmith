package goldsmith

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type (
	FileProp  any
	FileProps map[string]FileProp

	// File represents in-memory or on-disk files in a chain.
	File struct {
		relPath string
		props   FileProps
		modTime time.Time
		size    int64

		dataPath string
		reader   *bytes.Reader

		index int
	}
)

// Rename modifies the file path relative to the source directory.
func (self *File) Rename(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("unexpected absolute path: %s", path)
	}

	self.relPath = path
	return nil
}

// Path returns the file path relative to the source directory.
func (self *File) Path() string {
	return filepath.ToSlash(self.relPath)
}

// Dir returns the containing directory of the file.
func (self *File) Dir() string {
	return filepath.ToSlash(filepath.Dir(self.relPath))
}

// Name returns the base name of the file.
func (self *File) Name() string {
	return filepath.Base(self.relPath)
}

// Ext returns the extension of the file.
func (self *File) Ext() string {
	return filepath.Ext(self.relPath)
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
	if self.reader == nil && offset == 0 && (whence == io.SeekStart || whence == io.SeekCurrent) {
		return 0, nil
	}

	if err := self.load(); err != nil {
		return 0, err
	}

	return self.reader.Seek(offset, whence)
}

// GoString returns value for string formatting.
func (self *File) GoString() string {
	return self.relPath
}

// RemoveProp deletes the metadata property for the provided name.
func (self *File) RemoveProp(name string) {
	delete(self.props, name)
}

// SetProp updates the metadata property for the provided name.
func (self *File) SetProp(name string, value FileProp) {
	self.props[name] = value
}

// Prop returns the metadata property for the provided name.
func (self *File) Prop(name string) (FileProp, bool) {
	value, ok := self.props[name]
	return value, ok
}

// PropOrDef returns the metadata property for the provided name or the default.
func (self *File) PropOrDef(name string, valueDef FileProp) FileProp {
	if value, ok := self.Prop(name); ok {
		return value
	}

	return valueDef
}

// Props returns all of the metadata properties.
func (self *File) Props() FileProps {
	return self.props
}

// CopyProps copies all metadata properties from the provided file.
func (self *File) CopyProps(file *File) {
	for key, value := range file.props {
		self.props[key] = value
	}
}

func (self *File) load() error {
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
