package goldsmith

import (
	"io"
	"time"
)

type contextFile struct {
	file  File
	index int
}

func (self *contextFile) Path() string {
	return self.file.Path()
}

func (self *contextFile) Dir() string {
	return self.file.Dir()
}

func (self *contextFile) Name() string {
	return self.file.Name()
}

func (self *contextFile) Ext() string {
	return self.file.Ext()
}

func (self *contextFile) Rename(path string) error {
	return self.file.Rename(path)
}

func (self *contextFile) Size() int64 {
	return self.file.Size()
}

func (self *contextFile) ModTime() time.Time {
	return self.file.ModTime()
}

func (self *contextFile) Read(data []byte) (int, error) {
	return self.file.Read(data)
}

func (self *contextFile) WriteTo(writer io.Writer) (int64, error) {
	return self.file.WriteTo(writer)
}

func (self *contextFile) Seek(offset int64, whence int) (int64, error) {
	return self.file.Seek(offset, whence)
}

func (self *contextFile) GoString() string {
	return self.file.GoString()
}

func (self *contextFile) SetProp(name string, value FileProp) {
	self.file.SetProp(name, value)
}

func (self *contextFile) Prop(name string) (FileProp, bool) {
	return self.file.Prop(name)
}

func (self *contextFile) PropOrDef(name string, valueDef FileProp) FileProp {
	return self.file.PropOrDef(name, valueDef)
}

func (self *contextFile) Props() FileProps {
	return self.file.Props()
}

func (self *contextFile) CopyProps(file File) {
	self.file.CopyProps(file)
}

func (self *contextFile) RemoveProp(name string) {
	self.file.RemoveProp(name)
}
