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
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type stage struct {
	input, output chan *File
}

type goldsmith struct {
	srcDir, dstDir string
	stages         []stage
	files          chan *File
	refs           map[string]bool
	err            error
}

func New(srcDir, dstDir string) Goldsmith {
	gs := &goldsmith{
		srcDir: srcDir,
		dstDir: dstDir,
		refs:   make(map[string]bool),
	}

	gs.scanFs()
	return gs
}

func (gs *goldsmith) scanFs() {
	fileMatches, _, err := scanDir(gs.srcDir)
	if err != nil {
		gs.err = err
		return
	}

	s := gs.makeStage()

	go func() {
		defer close(s.output)

		for _, match := range fileMatches {
			relPath, err := filepath.Rel(gs.srcDir, match)
			if err != nil {
				panic(err)
			}

			file, _ := gs.NewFile(relPath)

			var f *os.File
			if f, file.Err = os.Open(match); file.Err == nil {
				_, file.Err = file.Buff.ReadFrom(f)
				f.Close()
			}

			s.output <- file
		}
	}()
}

func (gs *goldsmith) cleanupFiles() {
	fileMatches, _, err := scanDir(gs.dstDir)
	if err != nil {
		gs.err = err
		return
	}

	for _, path := range fileMatches {
		relPath, err := filepath.Rel(gs.dstDir, path)
		if err != nil {
			gs.err = err
			return
		}

		if contained, _ := gs.refs[relPath]; !contained {
			if err := os.Remove(path); err != nil {
				gs.err = err
				return
			}
		}
	}
}

func (gs *goldsmith) exportFile(file *File) {
	defer func() {
		file.Buff = nil
	}()

	if file.Err != nil {
		return
	}

	absPath := filepath.Join(gs.dstDir, file.Path)
	if file.Err = os.MkdirAll(path.Dir(absPath), 0755); file.Err != nil {
		return
	}

	var f *os.File
	if f, file.Err = os.Create(absPath); file.Err == nil {
		defer f.Close()
		if _, file.Err = f.Write(file.Buff.Bytes()); file.Err == nil {
			gs.refs[file.Path] = true
		}
	}
}

func (gs *goldsmith) makeStage() stage {
	s := stage{output: make(chan *File)}
	if len(gs.stages) > 0 {
		s.input = gs.stages[len(gs.stages)-1].output
	}

	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) chain(s stage, c Chainer) {
	f, _ := c.(Filterer)

	allowed := make(chan *File)
	defer close(allowed)

	go c.Chain(gs, allowed, s.output)

	for file := range s.input {
		if f.Filter(file.Path) {
			s.output <- file
		} else {
			allowed <- file
		}
	}
}

func (gs *goldsmith) NewFile(path string) (*File, error) {
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("absolute paths are not supported: %s", path)
	}

	file := &File{
		Path: path,
		Meta: make(map[string]interface{}),
		Buff: new(bytes.Buffer),
	}

	return file, nil
}

func (gs *goldsmith) RefFile(path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths are not supported: %s", path)
	}

	gs.refs[path] = true
	return nil
}

func (gs *goldsmith) SrcDir() string {
	return gs.srcDir
}

func (gs *goldsmith) DstDir() string {
	return gs.dstDir
}

func (gs *goldsmith) Chain(c Chainer, err error) Goldsmith {
	if gs.err != nil {
		return gs
	}

	if gs.err = err; gs.err == nil {
		go gs.chain(gs.makeStage(), c)
	}

	return gs
}

func (gs *goldsmith) Complete() ([]*File, error) {
	s := gs.stages[len(gs.stages)-1]

	var files []*File
	for file := range s.output {
		gs.exportFile(file)
		files = append(files, file)
	}

	gs.cleanupFiles()
	return files, gs.err
}
