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
)

type file struct {
	path string
	meta map[string]interface{}

	srcData *bytes.Reader
	srcPath string
}

func newFileFromData(path string, srcData []byte) *file {
	return &file{
		path:    path,
		meta:    make(map[string]interface{}),
		srcData: bytes.NewReader(srcData),
	}
}

func newFileFromPath(path, srcPath string) *file {
	return &file{
		path:    path,
		meta:    make(map[string]interface{}),
		srcPath: srcPath,
	}
}

func (f *file) rewind() {
	if f.srcData != nil {
		f.srcData.Seek(0, os.SEEK_SET)
	}
}

func (f *file) export(dstPath string) error {
	if err := os.MkdirAll(path.Dir(dstPath), 0755); err != nil {
		return err
	}

	if err := f.cache(); err != nil {
		return err
	}

	f.rewind()

	fh, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fh.Close()

	var buff [1024]byte
	for {
		count, err := f.Read(buff[:])
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if _, err := fh.Write(buff[:count]); err != nil {
			return err
		}
	}

	return nil
}

func (f *file) cache() error {
	if f.srcData != nil {
		return nil
	}

	data, err := ioutil.ReadFile(f.srcPath)
	if err != nil {
		return err
	}

	f.srcData = bytes.NewReader(data)
	return nil
}

func (f *file) Path() string {
	return f.path
}

func (f *file) Keys() (keys []string) {
	for key := range f.meta {
		keys = append(keys, key)
	}

	return keys
}

func (f *file) Value(key string, def interface{}) interface{} {
	if value, ok := f.meta[key]; ok {
		return value
	}

	return def
}

func (f *file) SetValue(key string, value interface{}) {
	f.meta[key] = value
}

func (f *file) Read(p []byte) (int, error) {
	if err := f.cache(); err != nil {
		return 0, err
	}

	return f.srcData.Read(p)
}
