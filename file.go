package goldsmith

import (
	"io"
	"time"
)

type (
	FileProp  any
	FileProps map[string]FileProp
)

// File represents in-memory or on-disk files in a chain.
type File interface {
	Path() string
	Dir() string
	Name() string
	Ext() string
	Rename(path string) error

	Size() int64
	ModTime() time.Time

	Read(data []byte) (int, error)
	WriteTo(writer io.Writer) (int64, error)
	Seek(offset int64, whence int) (int64, error)

	SetProp(name string, value FileProp)
	Prop(name string) (FileProp, bool)
	PropOrDef(name string, valueDef FileProp) FileProp
	Props() FileProps
	CopyProps(file File)
	RemoveProp(name string)

	GoString() string
}
