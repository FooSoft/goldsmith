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
	"io/ioutil"
	"path/filepath"

	"github.com/bmatcuk/doublestar"
)

type stage struct {
	input, output chan File
}

type goldsmith struct {
	stages []stage
	files  chan File
}

func NewGoldsmith(path string) (Goldsmith, error) {
	gs := new(goldsmith)
	if err := gs.scan(path); err != nil {
		return nil, err
	}

	return gs, nil
}

func (gs *goldsmith) scan(path string) error {
	matches, err := doublestar.Glob(filepath.Join(path, "**"))
	if err != nil {
		return err
	}

	s := stage{
		input:  nil,
		output: make(chan File, len(matches)),
	}

	for _, match := range matches {
		path, err := filepath.Rel(path, match)
		if err != nil {
			return err
		}

		s.output <- gs.NewFile(path)
	}

	gs.stages = append(gs.stages, s)
	return nil
}

func (gs *goldsmith) stage() stage {
	s := stage{
		input:  gs.stages[len(gs.stages)-1].output,
		output: make(chan File),
	}

	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) NewFile(path string) File {
	return &file{path: path}
}

func (gs *goldsmith) Apply(p Processor) Goldsmith {
	s := gs.stage()
	go p.Process(gs, s.input, s.output)
	return gs
}

func (gs *goldsmith) Complete(path string) []File {
	s := gs.stages[len(gs.stages)-1]

	var files []File
	for file := range s.output {
		data, err := file.Data()
		if err != nil {
			file.SetError(err)
			continue
		}

		absPath := filepath.Join(path, file.Path())
		if err := ioutil.WriteFile(absPath, data.Bytes(), 0644); err != nil {
			file.SetError(err)
			continue
		}

		files = append(files, file)
	}

	return files
}
