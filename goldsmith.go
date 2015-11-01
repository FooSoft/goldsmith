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

	"github.com/bmatcuk/doublestar"
)

type stage struct {
	input, output chan File
}

type goldsmith struct {
	stages []stage
	files  chan File
	err    error
}

func New(src string) Goldsmith {
	gs := new(goldsmith)
	gs.scan(src)
	return gs
}

func (gs *goldsmith) scan(srcDir string) {
	matches, err := doublestar.Glob(filepath.Join(srcDir, "**"))
	if err != nil {
		gs.err = err
		return
	}

	s := stage{nil, make(chan File, len(matches))}
	for _, match := range matches {
		relPath, err := filepath.Rel(srcDir, match)
		if err != nil {
			gs.err = err
			return
		}

		s.output <- &file{relPath: relPath, srcPath: match}
	}

	close(s.output)
	gs.stages = append(gs.stages, s)
}

func (gs *goldsmith) makeStage() stage {
	s := stage{gs.stages[len(gs.stages)-1].output, make(chan File)}
	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) NewFile(relPath string) File {
	return &file{relPath: relPath}
}

func (gs *goldsmith) SetError(err error) {
	gs.err = err
}

func (gs *goldsmith) taskSingle(ts TaskerSingle) {
	s := gs.makeStage()

	var wg sync.WaitGroup
	for file := range s.input {
		wg.Add(1)
		go func(f File) {
			s.output <- ts.TaskSingle(gs, f)
			wg.Done()
		}(file)
	}

	go func() {
		wg.Wait()
		close(s.output)
	}()
}

func (gs *goldsmith) taskMultiple(tm TaskerMultiple) {
	s := gs.makeStage()
	tm.TaskMultiple(gs, s.input, s.output)
}

func (gs *goldsmith) Task(task interface{}) Goldsmith {
	if gs.err == nil {
		switch t := task.(type) {
		case TaskerSingle:
			gs.taskSingle(t)
		case TaskerMultiple:
			gs.taskMultiple(t)
		}
	}

	return gs
}

func (gs *goldsmith) Complete(dstDir string) ([]File, error) {
	s := gs.stages[len(gs.stages)-1]

	var files []File
	for file := range s.output {
		data := file.Data()
		if data == nil {
			continue
		}

		absPath := filepath.Join(dstDir, file.Path())

		if err := os.MkdirAll(path.Dir(absPath), 0755); err != nil {
			file.SetError(err)
			continue
		}

		f, err := os.Create(absPath)
		if err != nil {
			file.SetError(err)
			continue
		}
		defer f.Close()

		if _, err := f.Write(data); err != nil {
			file.SetError(err)
			continue
		}

		files = append(files, file)
	}

	return files, gs.err
}
