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
	Chain(p Plugin, err error) Goldsmith
	Complete() ([]*File, []error)
}

type Plugin interface{}

type Initializer interface {
	Initialize(ctx Context) error
}

type Finalizer interface {
	Finalize(ctx Context) error
}

type Processor interface {
	Process(ctx Context, file *File) bool
}

type Chainer interface {
	Chain(ctx Context, input, output chan *File)
}

type FileType int

const (
	FileNormal FileType = iota
	FileStatic
	FileReference
)

type File struct {
	Path string
	Meta map[string]interface{}
	Buff bytes.Buffer
	Err  error
	Type FileType
}

type Context interface {
	SrcDir() string
	DstDir() string

	NewFile(path string) *File
	NewFileStatic(path string) *File
	NewFileRef(path string) *File
}
