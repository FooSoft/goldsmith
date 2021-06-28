package goldsmith

import (
	"bytes"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Property interface{}

// File represents in-memory or on-disk files in a chain.
type File struct {
	sourcePath string
	dataPath   string
	properties map[string]Property

	hashValue *uint32
	reader    *bytes.Reader
	size      int64
	modTime   time.Time
}

// Rename modifies the file path relative to the source directory.
func (file *File) Rename(path string) {
	file.sourcePath = path
}

// Path returns the file path relative to the source directory.
func (file *File) Path() string {
	return file.sourcePath
}

// Name returns the base name of the file.
func (file *File) Name() string {
	return path.Base(file.sourcePath)
}

// Dir returns the containing directory of the file.
func (file *File) Dir() string {
	return path.Dir(file.sourcePath)
}

// Ext returns the extension of the file.
func (file *File) Ext() string {
	return path.Ext(file.sourcePath)
}

// Size returns the file length in bytes.
func (file *File) Size() int64 {
	return file.size
}

// ModTime returns the time of the file's last modification.
func (file *File) ModTime() time.Time {
	return file.modTime
}

func (file *File) SetProperty(name string, value Property) {
	file.properties[name] = value
}

func (file *File) GetProperty(name string) (Property, bool) {
	value, ok := file.properties[name]
	return value, ok
}

func (file *File) GetPropertyOrDefault(name string, valueDef Property) Property {
	if value, ok := file.GetProperty(name); ok {
		return value
	}

	return valueDef
}

func (file *File) CopyPropertiesTo(other *File) {
	for key, value := range file.properties {
		other.SetProperty(key, value)
	}
}

// Read reads file data into the provided buffer.
func (file *File) Read(data []byte) (int, error) {
	if err := file.load(); err != nil {
		return 0, err
	}

	return file.reader.Read(data)
}

// Write writes file data into the provided writer.
func (file *File) WriteTo(writer io.Writer) (int64, error) {
	if err := file.load(); err != nil {
		return 0, err
	}

	return file.reader.WriteTo(writer)
}

// Seek updates the file pointer to the desired position.
func (file *File) Seek(offset int64, whence int) (int64, error) {
	if file.reader == nil && offset == 0 && (whence == os.SEEK_SET || whence == os.SEEK_CUR) {
		return 0, nil
	}

	if err := file.load(); err != nil {
		return 0, err
	}

	return file.reader.Seek(offset, whence)
}

// Returns value for string formatting.
func (file *File) GoString() string {
	return file.sourcePath
}

func (file *File) export(targetDir string) error {
	targetPath := filepath.Join(targetDir, file.sourcePath)

	if len(file.dataPath) == 0 {
		if targetInfo, err := os.Stat(targetPath); err == nil && targetInfo.ModTime().After(file.ModTime()) {
			return nil
		}
	}

	if err := os.MkdirAll(path.Dir(targetPath), 0755); err != nil {
		return err
	}

	fw, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer fw.Close()

	if file.reader == nil {
		fr, err := os.Open(file.dataPath)
		if err != nil {
			return err
		}
		defer fr.Close()

		if _, err := io.Copy(fw, fr); err != nil {
			return err
		}
	} else {
		if _, err := file.Seek(0, os.SEEK_SET); err != nil {
			return err
		}

		if _, err := file.WriteTo(fw); err != nil {
			return err
		}
	}

	return nil
}

func (file *File) load() error {
	if file.reader != nil {
		return nil
	}

	data, err := ioutil.ReadFile(file.dataPath)
	if err != nil {
		return err
	}

	file.reader = bytes.NewReader(data)
	return nil
}

func (file *File) hash() (uint32, error) {
	if file.hashValue != nil {
		return *file.hashValue, nil
	}

	if err := file.load(); err != nil {
		return 0, err
	}

	offset, err := file.Seek(0, os.SEEK_CUR)
	if err != nil {
		return 0, err
	}

	if _, err := file.Seek(0, os.SEEK_SET); err != nil {
		return 0, err
	}

	hasher := crc32.NewIEEE()
	if _, err := io.Copy(hasher, file.reader); err != nil {
		return 0, err
	}

	if _, err := file.Seek(offset, os.SEEK_SET); err != nil {
		return 0, err
	}

	hashValue := hasher.Sum32()
	file.hashValue = &hashValue

	return *file.hashValue, nil
}

type filesByPath []*File

func (file filesByPath) Len() int {
	return len(file)
}

func (file filesByPath) Swap(i, j int) {
	file[i], file[j] = file[j], file[i]
}

func (file filesByPath) Less(i, j int) bool {
	return strings.Compare(file[i].Path(), file[j].Path()) < 0
}

type fileInfo struct {
	os.FileInfo
	path string
}

func cleanPath(path string) string {
	if filepath.IsAbs(path) {
		var err error
		if path, err = filepath.Rel("/", path); err != nil {
			panic(err)
		}
	}

	return filepath.Clean(path)
}

func scanDir(rootDir string, infos chan fileInfo) {
	defer close(infos)

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			infos <- fileInfo{FileInfo: info, path: path}
		}

		return err
	})
}

type filterStack []Filter

func (filters *filterStack) accept(file *File) bool {
	for _, filter := range *filters {
		if !filter.Accept(file) {
			return false
		}
	}

	return true
}

func (filters *filterStack) push(filter Filter) {
	*filters = append(*filters, filter)
}

func (filters *filterStack) pop() {
	count := len(*filters)
	if count == 0 {
		panic("attempted to pop empty filter stack")
	}

	*filters = (*filters)[:count-1]
}
