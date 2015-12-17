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
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type stage struct {
	gs            *goldsmith
	input, output chan *File
	err           error
}

type goldsmith struct {
	srcDir, dstDir string
	stages         []*stage
	refs           map[string]bool
	mtx            sync.Mutex
	active         int64
	stalled        int64
}

func (gs *goldsmith) queueFiles(target uint) {
	files := make(chan string)
	go scanDir(gs.srcDir, files, nil)

	s := gs.newStage()

	go func() {
		defer close(s.output)

		for path := range files {
			for {
				if gs.active-gs.stalled >= int64(target) {
					time.Sleep(time.Millisecond)
				} else {
					break
				}
			}

			relPath, err := filepath.Rel(gs.srcDir, path)
			if err != nil {
				panic(err)
			}

			file := NewFile(relPath)

			var f *os.File
			if f, file.Err = os.Open(path); file.Err == nil {
				_, file.Err = file.Buff.ReadFrom(f)
				f.Close()
			}

			s.AddFile(file)
		}
	}()
}

func (gs *goldsmith) cleanupFiles() {
	var (
		files = make(chan string)
		dirs  = make(chan string)
	)

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
	defer func() {
		file.Buff = *bytes.NewBuffer(nil)
		atomic.AddInt64(&gs.active, -1)
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
			gs.refFile(file.Path)
		}
	}
}

func (gs *goldsmith) refFile(path string) {
	gs.mtx.Lock()
	defer gs.mtx.Unlock()

	if gs.refs == nil {
		gs.refs = make(map[string]bool)
	}

	path = cleanPath(path)

	for {
		gs.refs[path] = true
		if path == "." {
			break
		}

		path = filepath.Dir(path)
	}
}

func (gs *goldsmith) newStage() *stage {
	s := &stage{gs: gs, output: make(chan *File)}
	if len(gs.stages) > 0 {
		s.input = gs.stages[len(gs.stages)-1].output
	}

	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) chain(s *stage, p Plugin) {
	defer close(s.output)

	init, _ := p.(Initializer)
	accept, _ := p.(Accepter)
	proc, _ := p.(Processor)
	fin, _ := p.(Finalizer)

	var (
		wg    sync.WaitGroup
		mtx   sync.Mutex
		batch []*File
	)

	dispatch := func(f *File) {
		if fin == nil {
			s.output <- f
		} else {
			mtx.Lock()
			batch = append(batch, f)
			atomic.AddInt64(&s.gs.stalled, 1)
			mtx.Unlock()
		}
	}

	if init != nil {
		if s.err = init.Initialize(s); s.err != nil {
			return
		}
	}

	for file := range s.input {
		if file.Err != nil || accept != nil && !accept.Accept(file) {
			s.output <- file
		} else if proc == nil {
			dispatch(file)
		} else {
			wg.Add(1)
			go func(f *File) {
				defer wg.Done()
				if proc.Process(s, f) {
					dispatch(f)
				} else {
					f.Buff = *bytes.NewBuffer(nil)
					atomic.AddInt64(&gs.active, -1)
				}
			}(file)
		}
	}

	wg.Wait()

	if fin != nil {
		if s.err = fin.Finalize(s, batch); s.err == nil {
			for _, file := range batch {
				atomic.AddInt64(&s.gs.stalled, -1)
				s.output <- file
			}
		}
	}
}

func (gs *goldsmith) Chain(p Plugin) Goldsmith {
	go gs.chain(gs.newStage(), p)
	return gs
}

func (gs *goldsmith) Complete() ([]*File, []error) {
	s := gs.stages[len(gs.stages)-1]

	var files []*File
	for file := range s.output {
		gs.exportFile(file)
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
