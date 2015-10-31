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
	"os"
)

type file struct {
	relPath, srcPath string
	meta             map[string]interface{}
	buff             *bytes.Buffer
	err              error
}

func (f *file) Path() string {
	return f.relPath
}

func (f *file) SetPath(path string) {
	f.relPath = path
}

func (f *file) Property(key, def string) interface{} {
	value, ok := f.meta[key]
	if ok {
		return value
	}

	return def
}

func (f *file) SetProperty(key string, value interface{}) {
	if f.meta == nil {
		f.meta = make(map[string]interface{})
	}

	f.meta[key] = value
}

func (f *file) Error() error {
	return f.err
}

func (f *file) SetError(err error) {
	f.err = err
}

func (f *file) Bytes() []byte {
	if f.buff != nil {
		return f.buff.Bytes()
	}

	var buff bytes.Buffer
	if len(f.srcPath) > 0 {
		file, err := os.Open(f.srcPath)
		if err != nil {
			f.SetError(err)
			return nil
		}
		defer file.Close()

		if _, err := buff.ReadFrom(file); err != nil {
			f.SetError(err)
			return nil
		}
	}

	f.buff = &buff
	return f.buff.Bytes()
}

func (f *file) SetBytes(data []byte) {
	f.buff = bytes.NewBuffer(data)
}
