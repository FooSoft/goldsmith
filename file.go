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
	path string
	meta map[string]interface{}
	buff *bytes.Buffer
	err  error
}

func (f *file) Path() string {
	return f.path
}

func (f *file) SetPath(path string) {
	f.path = path
}

func (f *file) Property(key, def string) interface{} {
	value, ok := f.meta[key]
	if ok {
		return value
	}

	return def
}

func (f *file) SetProperty(key string, value interface{}) {
	f.meta[key] = value
}

func (f *file) Error() error {
	return f.err
}

func (f *file) SetError(err error) {
	f.err = err
}

func (f *file) Data() (*bytes.Buffer, error) {
	if f.buff != nil {
		return f.buff, nil
	}

	file, err := os.Open(f.path)
	if err != nil {
		f.SetError(err)
		return nil, err
	}
	defer file.Close()

	var buff bytes.Buffer
	if _, err := buff.ReadFrom(file); err != nil {
		f.SetError(err)
		return nil, err
	}

	f.buff = &buff
	return f.buff, nil
}
