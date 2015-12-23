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
	"path/filepath"
	"sync"
)

type goldsmith struct {
	srcDir, dstDir string
	contexts       []*context

	refs   map[string]bool
	refMtx sync.Mutex

	errors   []error
	errorMtx sync.Mutex
}

func (gs *goldsmith) queueFiles() {
	files := make(chan string)
	go scanDir(gs.srcDir, files, nil)

	ctx := newContext(gs)

	go func() {
		defer close(ctx.output)
		for path := range files {
			relPath, err := filepath.Rel(gs.srcDir, path)
			if err != nil {
				panic(err)
			}

			f := NewFileFromAsset(relPath, path)
			ctx.DispatchFile(f)
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

func (gs *goldsmith) exportFile(f *file) error {
	absPath := filepath.Join(gs.dstDir, f.path)
	if err := f.export(absPath); err != nil {
		return err
	}

	gs.referenceFile(f.path)
	return nil
}

func (gs *goldsmith) referenceFile(path string) {
	gs.refMtx.Lock()
	defer gs.refMtx.Unlock()

	path = cleanPath(path)

	for {
		gs.refs[path] = true
		if path == "." {
			break
		}

		path = filepath.Dir(path)
	}
}

func (gs *goldsmith) fault(f *file, err error) {
	gs.errorMtx.Lock()
	ferr := &Error{err: err}
	if f != nil {
		ferr.path = f.path
	}
	gs.errors = append(gs.errors, ferr)
	gs.errorMtx.Unlock()
}

//
//	Goldsmith Implementation
//

func (gs *goldsmith) Chain(p Plugin) Goldsmith {
	ctx := newContext(gs)
	go ctx.chain(p)
	return gs
}

func (gs *goldsmith) Complete() []error {
	ctx := gs.contexts[len(gs.contexts)-1]
	for f := range ctx.output {
		gs.exportFile(f)
	}

	gs.cleanupFiles()
	return gs.errors
}
