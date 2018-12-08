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

type File struct {
	sourcePath string
	dataPath   string

	Meta map[string]interface{}

	hashValue uint32
	hashValid bool

	reader  *bytes.Reader
	size    int64
	modTime time.Time
}

func (f *File) Path() string {
	return f.sourcePath
}

func (f *File) Name() string {
	return path.Base(f.sourcePath)
}

func (f *File) Dir() string {
	return path.Dir(f.sourcePath)
}

func (f *File) Ext() string {
	return path.Ext(f.sourcePath)
}

func (f *File) Size() int64 {
	return f.size
}

func (f *File) ModTime() time.Time {
	return f.modTime
}

func (f *File) Read(data []byte) (int, error) {
	if err := f.load(); err != nil {
		return 0, err
	}

	return f.reader.Read(data)
}

func (f *File) WriteTo(writer io.Writer) (int64, error) {
	if err := f.load(); err != nil {
		return 0, err
	}

	return f.reader.WriteTo(writer)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if f.reader == nil && offset == 0 && (whence == os.SEEK_SET || whence == os.SEEK_CUR) {
		return 0, nil
	}

	if err := f.load(); err != nil {
		return 0, err
	}

	return f.reader.Seek(offset, whence)
}

type FilesByPath []*File

func (f FilesByPath) Len() int {
	return len(f)
}

func (f FilesByPath) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f FilesByPath) Less(i, j int) bool {
	return strings.Compare(f[i].Path(), f[j].Path()) < 0
}

func (f *File) export(targetDir string) error {
	targetPath := filepath.Join(targetDir, f.sourcePath)
	if targetInfo, err := os.Stat(targetPath); err == nil && targetInfo.ModTime().After(f.ModTime()) {
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

	if f.reader == nil {
		fr, err := os.Open(f.dataPath)
		if err != nil {
			return err
		}
		defer fr.Close()

		if _, err := io.Copy(fw, fr); err != nil {
			return err
		}
	} else {
		if _, err := f.Seek(0, os.SEEK_SET); err != nil {
			return err
		}

		if _, err := f.WriteTo(fw); err != nil {
			return err
		}
	}

	return nil
}

func (f *File) load() error {
	if f.reader != nil {
		return nil
	}

	data, err := ioutil.ReadFile(f.dataPath)
	if err != nil {
		return err
	}

	f.reader = bytes.NewReader(data)
	return nil
}

func (f *File) hash() (uint32, error) {
	if f.hashValid {
		return f.hashValue, nil
	}

	if err := f.load(); err != nil {
		return 0, err
	}

	offset, err := f.Seek(0, os.SEEK_CUR)
	if err != nil {
		return 0, err
	}

	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return 0, err
	}

	hasher := crc32.NewIEEE()
	if _, err := io.Copy(hasher, f.reader); err != nil {
		return 0, err
	}

	if _, err := f.Seek(offset, os.SEEK_SET); err != nil {
		return 0, err
	}

	f.hashValue = hasher.Sum32()
	f.hashValid = true
	return f.hashValue, nil
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
