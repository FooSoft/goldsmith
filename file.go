/*
 * Copyright (c) 2015 Alex Yatskov <alex@foosoft.net>
 * Author: Alex Yatskov <alex@foosoft.net>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

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
	if len(f.asset) > 0 && fileCached(f.asset, dstPath) {
		return nil
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
	value, ok := f.Meta[key]
	return value, ok
}

func (f *file) SetValue(key string, value interface{}) {
	f.Meta[key] = value
}

func (f *file) CopyValues(src File) {
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
