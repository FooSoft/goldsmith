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
	defer close(s.output)

	for _, match := range matches {
		relPath, err := filepath.Rel(srcDir, match)
		if err != nil {
			panic(err)
		}

		file := File{
			Path: relPath,
			Meta: make(map[string]interface{}),
		}

		var f *os.File
		if f, file.Err = os.Open(match); file.Err == nil {
			defer f.Close()
			_, file.Err = file.Buff.ReadFrom(f)
		}

		s.output <- file
	}

	gs.stages = append(gs.stages, s)
}

func (gs *goldsmith) makeStage() stage {
	s := stage{gs.stages[len(gs.stages)-1].output, make(chan File)}
	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) chainSingle(ts ChainerSingle) {
	s := gs.makeStage()

	var wg sync.WaitGroup
	for file := range s.input {
		wg.Add(1)
		go func(f File) {
			s.output <- ts.ChainSingle(f)
			wg.Done()
		}(file)
	}

	go func() {
		wg.Wait()
		close(s.output)
	}()
}

func (gs *goldsmith) chainMultiple(tm ChainerMultiple) {
	s := gs.makeStage()
	tm.ChainMultiple(s.input, s.output)
}

func (gs *goldsmith) Chain(chain interface{}, err error) Goldsmith {
	if gs.err != nil {
		return gs
	}

	if gs.err = err; gs.err != nil {
		switch t := chain.(type) {
		case ChainerSingle:
			gs.chainSingle(t)
		case ChainerMultiple:
			gs.chainMultiple(t)
		}
	}

	return gs
}

func (gs *goldsmith) Complete(dstDir string) ([]File, error) {
	s := gs.stages[len(gs.stages)-1]

	var files []File
	for file := range s.output {
		if file.Err == nil {
			absPath := filepath.Join(dstDir, file.Path)
			if file.Err = os.MkdirAll(path.Dir(absPath), 0755); file.Err != nil {
				continue
			}

			var f *os.File
			if f, file.Err = os.Create(absPath); f == nil {
				_, file.Err = f.Write(file.Buff.Bytes())
				f.Close()
			}
		}

		file.Buff.Reset()
		files = append(files, file)
	}

	return files, gs.err
}
