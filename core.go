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

	"github.com/bmatcuk/doublestar"
)

type goldsmith struct {
	Context

	files []File
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

	for _, match := range matches {
		path, err := filepath.Rel(gs.srcPath, match)
		if err != nil {
			return err
		}

		file := File{path, make(map[string]interface{})}
		gs.files = append(gs.files, file)
	}

	return nil
}

func (gs *goldsmith) ApplyAll(p Processor) Applier {
	return gs.Apply(p, "*")
}

func (gs *goldsmith) Apply(p Processor, pattern string) Applier {
	inputFiles := make(chan File)
	outputFiles := make(chan File)

	for _, file := range gs.files {
		if matched, _ := doublestar.Match(pattern, file.Path); matched {
			inputFiles <- file
		}
	}

	p.ProcessFiles(inputFiles, outputFiles)
	return gs
}
