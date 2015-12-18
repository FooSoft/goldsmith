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

	reader *bytes.Reader
	asset  string
}

func (f *file) rewind() {
	if f.reader != nil {
		f.reader.Seek(0, os.SEEK_SET)
	}
}

func (f *file) export(dstPath string) error {
	if err := os.MkdirAll(path.Dir(dstPath), 0755); err != nil {
		return err
	}

	if err := f.cache(); err != nil {
		return err
	}

	fh, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fh.Close()

	f.rewind()
	if _, err := f.WriteTo(fh); err != nil {
		return err
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

	f.Rewrite(data)
	return nil
}

//
//	File Implementation
//

func (f *file) Path() string {
	return f.path
}

func (f *file) Rename(path string) {
	f.path = path
}

func (f *file) Meta() map[string]interface{} {
	return f.meta
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

func (f *file) Rewrite(data []byte) {
	f.reader = bytes.NewReader(data)
}
