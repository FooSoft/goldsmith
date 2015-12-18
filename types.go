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
	"io"
	"runtime"
)

type Goldsmith interface {
	Chain(p Plugin) Goldsmith
	Complete() bool
}

func New(srcDir, dstDir string) Goldsmith {
	return NewThrottled(srcDir, dstDir, uint(runtime.NumCPU()))
}

func NewThrottled(srcDir, dstDir string, targetFileCount uint) Goldsmith {
	gs := &goldsmith{srcDir: srcDir, dstDir: dstDir}
	gs.queueFiles(targetFileCount)
	return gs
}

type File interface {
	Path() string

	Keys() []string
	Value(key string, def interface{}) interface{}
	SetValue(key string, value interface{})

	Read(p []byte) (int, error)
}

type Context interface {
	NewFile(path string, r io.Reader) File
	CopyFile(dst, src string) File
	RefFile(path string)

	SrcDir() string
	DstDir() string
}

type Plugin interface{}

type Accepter interface {
	Accept(file File) bool
}

type Initializer interface {
	Initialize(ctx Context) error
}

type Finalizer interface {
	Finalize(ctx Context, fs []File) error
}

type Processor interface {
	Process(ctx Context, f File) error
}
