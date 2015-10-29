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
	"path/filepath"
	"sync"

	"github.com/bmatcuk/doublestar"
)

type stage struct {
	input  chan File
	output chan File
}

type goldsmith struct {
	Context

	stages []stage
	files  chan File
	wg     sync.WaitGroup
}

func NewGoldsmith(srcPath, dstPath string) (Applier, error) {
	gs := &goldsmith{Context: Context{srcPath, dstPath}}
	if err := gs.scan(); err != nil {
		return nil, err
	}

	return gs, nil
}

func (gs *goldsmith) scan() error {
	matches, err := doublestar.Glob(filepath.Join(gs.srcPath, "**"))
	if err != nil {
		return err
	}

	gs.files = make(chan File, len(matches))

	for _, match := range matches {
		path, err := filepath.Rel(gs.srcPath, match)
		if err != nil {
			return err
		}

		gs.files <- File{path, make(map[string]interface{})}
	}

	return nil
}

func (gs *goldsmith) stage() stage {
	s := stage{output: make(chan File)}
	if len(gs.stages) == 0 {
		s.input = gs.files
	} else {
		s.input = gs.stages[len(gs.stages)-1].output
	}

	gs.stages = append(gs.stages, s)
	return s
}

func (gs *goldsmith) Apply(p Processor) Applier {
	return gs.ApplyTo(p, "*")
}

func (gs *goldsmith) ApplyTo(p Processor, pattern string) Applier {
	s := gs.stage()

	gs.wg.Add(1)
	go func() {
		p.ProcessFiles(s.input, s.output)
		gs.wg.Done()
	}()

	return gs
}

func (gs *goldsmith) Wait() Applier {
	gs.wg.Wait()
	return gs
}
