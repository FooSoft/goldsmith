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
)

type Goldsmith interface {
	Chain(p Plugin) Goldsmith
	End(dstDir string) []error
}

func Begin(srcDir string) Goldsmith {
	gs := &goldsmith{srcDir: srcDir, refs: make(map[string]bool)}
	gs.Chain(new(loader))
	return gs
}

type File interface {
	Path() string

	Value(key string) (interface{}, bool)
	SetValue(key string, value interface{})
	CopyValues(src File)

	Read(p []byte) (int, error)
	WriteTo(w io.Writer) (int64, error)
	Seek(offset int64, whence int) (int64, error)
}

func NewFileFromData(path string, data []byte) File {
	return &file{
		path:   path,
		Meta:   make(map[string]interface{}),
		reader: bytes.NewReader(data),
	}
}

func NewFileFromAsset(path, asset string) File {
	return &file{
		path:  path,
		Meta:  make(map[string]interface{}),
		asset: asset,
	}
}

type Context interface {
	DispatchFile(f File)

	SrcDir() string
	DstDir() string
}

type Error struct {
	Err  error
	Path string
}

func (e Error) Error() string {
	return e.Err.Error()
}

type Initializer interface {
	Initialize(ctx Context) error
}

type Accepter interface {
	Accept(ctx Context, f File) bool
}

type Processor interface {
	Process(ctx Context, f File) error
}

type Finalizer interface {
	Finalize(ctx Context) error
}

type Plugin interface{}
