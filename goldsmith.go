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
	"log"
	"os"
	"path"
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

func NewGoldsmith(src string) Goldsmith {
	gs := new(goldsmith)
	gs.scan(src)
	return gs
}

func (gs *goldsmith) scan(srcDir string) error {
	matches, err := doublestar.Glob(filepath.Join(srcDir, "**"))
	if err != nil {
		return err
	}

	s := stage{
		input:  nil,
		output: make(chan File, len(matches)),
	}

	for _, match := range matches {
		relPath, err := filepath.Rel(srcDir, match)
		if err != nil {
			return err
		}

		s.output <- &file{relPath: relPath, srcPath: match}
	}

	close(s.output)
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

func (gs *goldsmith) NewFile(relPath string) File {
	return &file{relPath: relPath}
}

func (gs *goldsmith) Apply(p Processor) Goldsmith {
	s := gs.stage()
	go p.Process(gs, s.input, s.output)
	return gs
}

func (gs *goldsmith) Complete(dstDir string) []File {
	s := gs.stages[len(gs.stages)-1]

	var files []File
	for file := range s.output {
		log.Print(file)

		data := file.Data()
		if data == nil {
			continue
		}

		absPath := filepath.Join(dstDir, file.Path())

		if err := os.MkdirAll(path.Dir(absPath), 0755); err != nil {
			file.SetError(err)
			continue
		}

		if err := ioutil.WriteFile(absPath, data.Bytes(), 0644); err != nil {
			file.SetError(err)
			continue
		}

		files = append(files, file)
	}

	return files
}
