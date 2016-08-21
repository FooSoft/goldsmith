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
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

type Goldsmith interface {
	Chain(p Plugin, filters ...string) Goldsmith
	End(dstDir string) []error
}

func Begin(srcDir string, filters ...string) Goldsmith {
	gs := &goldsmith{srcDir: srcDir, refs: make(map[string]bool)}
	gs.Chain(new(loader), filters...)
	return gs
}

type File interface {
	Path() string
	Name() string
	Dir() string
	Ext() string
	Size() int64
	ModTime() time.Time

	Value(key string) (interface{}, bool)
	SetValue(key string, value interface{})
	CopyValues(src File)

	Read(p []byte) (int, error)
	WriteTo(w io.Writer) (int64, error)
	Seek(offset int64, whence int) (int64, error)
}

func NewFileFromData(path string, data []byte) File {
	return &file{
		path:    path,
		Meta:    make(map[string]interface{}),
		reader:  bytes.NewReader(data),
		size:    int64(len(data)),
		modTime: time.Now(),
	}
}

func NewFileFromAsset(path, asset string) (File, error) {
	info, err := os.Stat(asset)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, errors.New("assets must be files")
	}

	f := &file{
		path:    path,
		Meta:    make(map[string]interface{}),
		size:    info.Size(),
		modTime: info.ModTime(),
		asset:   asset,
	}

	return f, nil
}

type Context interface {
	DispatchFile(f File)

	SrcDir() string
	DstDir() string
}

type Error struct {
	Name string
	Path string
	Err  error
}

func (e Error) Error() string {
	var path string
	if len(e.Path) > 0 {
		path = "@" + e.Path
	}

	return fmt.Sprintf("[%s%s]: %s", e.Name, path, e.Err.Error())
}

type Initializer interface {
	Initialize(ctx Context) ([]string, error)
}

type Processor interface {
	Process(ctx Context, f File) error
}

type Finalizer interface {
	Finalize(ctx Context) error
}

type Plugin interface {
	Name() string
}
