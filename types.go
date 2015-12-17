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

import "bytes"

type Goldsmith interface {
	Chain(p Plugin) Goldsmith
	Complete() ([]*File, []error)
}

func New(srcDir, dstDir string) Goldsmith {
	gs := &goldsmith{srcDir: srcDir, dstDir: dstDir}
	gs.queueFiles()
	return gs
}

type File struct {
	Path string
	Meta map[string]interface{}
	Buff bytes.Buffer
	Err  error
}

func NewFile(path string) *File {
	return &File{
		Path: cleanPath(path),
		Meta: make(map[string]interface{}),
	}
}

type Context interface {
	SrcDir() string
	DstDir() string

	AddFile(file *File)
	RefFile(path string)
}

type Plugin interface{}

type Accepter interface {
	Accept(file *File) bool
}

type Initializer interface {
	Initialize(ctx Context) error
}

type Finalizer interface {
	Finalize(ctx Context) error
}

type Processor interface {
	Process(ctx Context, file *File) bool
}
