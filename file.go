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

type file struct {
	path string
	Meta map[string]interface{}

	reader  *bytes.Reader
	size    int64
	modTime time.Time

	asset string
}

func (f *file) export(dstDir string) error {
	dstPath := filepath.Join(dstDir, f.path)
	if len(f.asset) > 0 {
		dstInfo, err := os.Stat(dstPath)
		if err == nil && dstInfo.ModTime().Unix() >= f.ModTime().Unix() {
			return nil
		}
	}

	if err := os.MkdirAll(path.Dir(dstPath), 0755); err != nil {
		return err
	}

	fw, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fw.Close()

	if f.reader == nil {
		fr, err := os.Open(f.asset)
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

func (f *file) cache() error {
	if f.reader != nil {
		return nil
	}

	data, err := ioutil.ReadFile(f.asset)
	if err != nil {
		return err
	}

	f.reader = bytes.NewReader(data)
	return nil
}

//
//	File Implementation
//

func (f *file) Path() string {
	return f.path
}

func (f *file) Name() string {
	return path.Base(f.path)
}

func (f *file) Dir() string {
	return path.Dir(f.path)
}

func (f *file) Ext() string {
	return path.Ext(f.path)
}

func (f *file) Size() int64 {
	return f.size
}

func (f *file) ModTime() time.Time {
	return f.modTime
}

func (f *file) Value(key string) (interface{}, bool) {
	return getDelimValue(f.Meta, key)
}

func (f *file) SetValue(key string, value interface{}) bool {
	return setDelimValue(f.Meta, key, value)
}

func (f *file) InheritValues(src File) {
	rf := src.(*file)
	for name, value := range rf.Meta {
		f.SetValue(name, value)
	}
}

func (f *file) Read(p []byte) (int, error) {
	if err := f.cache(); err != nil {
		return 0, err
	}

	return f.reader.Read(p)
}

func (f *file) WriteTo(w io.Writer) (int64, error) {
	if err := f.cache(); err != nil {
		return 0, err
	}

	return f.reader.WriteTo(w)
}

func (f *file) Seek(offset int64, whence int) (int64, error) {
	if f.reader == nil && offset == 0 && (whence == os.SEEK_SET || whence == os.SEEK_CUR) {
		return 0, nil
	}

	if err := f.cache(); err != nil {
		return 0, err
	}

	return f.reader.Seek(offset, whence)
}
