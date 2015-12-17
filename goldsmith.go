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
	"os"
	"path"
	"path/filepath"
	"sync"
)

type stage struct {
	input, output chan *File
	err           error
}

type goldsmith struct {
	srcDir, dstDir string
	stages         []*stage
	refs           map[string]bool
}

func New(srcDir, dstDir string) Goldsmith {
	gs := &goldsmith{srcDir: srcDir, dstDir: dstDir}
	gs.queueFiles()
	return gs
}

func (gs *goldsmith) NewFile(path string) *File {
	return &File{
		Path: cleanPath(path),
		Meta: make(map[string]interface{}),
	}
}

func (gs *goldsmith) NewFileStatic(path string) *File {
	file := gs.NewFile(path)
	file.Type = FileStatic
	return file
}

func (gs *goldsmith) NewFileRef(path string) *File {
	file := gs.NewFile(path)
	file.Type = FileReference
	return file
}

func (gs *goldsmith) queueFiles() {
	files := make(chan string)
	go scanDir(gs.srcDir, files, nil)

	s := gs.makeStage()

	go func() {
		defer close(s.output)

		for path := range files {
			relPath, err := filepath.Rel(gs.srcDir, path)
			if err != nil {
				panic(err)
			}

			file := gs.NewFile(relPath)

			var f *os.File
			if f, file.Err = os.Open(path); file.Err == nil {
				_, file.Err = file.Buff.ReadFrom(f)
				f.Close()
			}

			s.output <- file
		}
	}()
}

func (gs *goldsmith) cleanupFiles() {
	files := make(chan string)
	dirs := make(chan string)
	go scanDir(gs.dstDir, files, dirs)

	for files != nil || dirs != nil {
		var (
			path string
			ok   bool
		)

		select {
		case path, ok = <-files:
			if !ok {
				files = nil
				continue
			}
		case path, ok = <-dirs:
			if !ok {
				dirs = nil
				continue
			}
		default:
			continue
		}

		relPath, err := filepath.Rel(gs.dstDir, path)
		if err != nil {
			panic(err)
		}

		if contained, _ := gs.refs[relPath]; contained {
			continue
		}

		os.RemoveAll(path)
	}
}

func (gs *goldsmith) exportFile(file *File) {
	if file.Err != nil {
		return
	}

	if file.Type == FileReference {
		gs.refFile(file.Path)
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
			gs.refFile(file.Path)
		}
	}
}

func (gs *goldsmith) makeStage() *stage {
	s := &stage{output: make(chan *File)}
	if len(gs.stages) > 0 {
		s.input = gs.stages[len(gs.stages)-1].output
	}

	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) chain(s *stage, p Plugin) {
	defer close(s.output)

	if init, ok := p.(Initializer); ok {
		if s.err = init.Initialize(gs); s.err != nil {
			return
		}
	}

	if proc, ok := p.(Processor); ok {
		var wg sync.WaitGroup
		for file := range s.input {
			go func(f *File) {
				defer wg.Done()
				if proc.Process(gs, f) {
					s.output <- f
				}
			}(file)
		}

		wg.Wait()
	} else {
		for file := range s.input {
			s.output <- file
		}
	}

	if fin, ok := p.(Finalizer); ok {
		s.err = fin.Finalize(gs)
	}
}

func (gs *goldsmith) refFile(path string) {
	path = cleanPath(path)

	if gs.refs == nil {
		gs.refs = make(map[string]bool)
	}

	for {
		gs.refs[path] = true
		if path == "." {
			break
		}

		path = filepath.Dir(path)
	}
}

func (gs *goldsmith) SrcDir() string {
	return gs.srcDir
}

func (gs *goldsmith) DstDir() string {
	return gs.dstDir
}

func (gs *goldsmith) Chain(p Plugin) Goldsmith {
	go gs.chain(gs.makeStage(), p)
	return gs
}

func (gs *goldsmith) Complete() ([]*File, []error) {
	s := gs.stages[len(gs.stages)-1]

	var files []*File
	for file := range s.output {
		gs.exportFile(file)
		file.Buff.Reset()
		files = append(files, file)
	}

	gs.cleanupFiles()

	var errs []error
	for _, s := range gs.stages {
		if s.err != nil {
			errs = append(errs, s.err)
		}
	}

	gs.stages = nil
	gs.refs = nil

	return files, errs
}

func cleanPath(path string) string {
	if filepath.IsAbs(path) {
		var err error
		if path, err = filepath.Rel("/", path); err != nil {
			panic(err)
		}
	}

	return filepath.Clean(path)
}

func scanDir(root string, files, dirs chan string) {
	defer func() {
		if files != nil {
			close(files)
		}
		if dirs != nil {
			close(dirs)
		}
	}()

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if dirs != nil {
				dirs <- path
			}
		} else {
			if files != nil {
				files <- path
			}
		}

		return nil
	})
}
