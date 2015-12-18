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
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type goldsmith struct {
	srcDir, dstDir string
	refs           map[string]bool
	mtx            sync.Mutex
	stages         []*stage
	active         int64
	stalled        int64
	tainted        bool
}

func (gs *goldsmith) queueFiles(target uint) {
	files := make(chan string)
	go scanDir(gs.srcDir, files, nil)

	s := newStage(gs)

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

			f := NewFileFromAsset(relPath, path)
			s.DispatchFile(f)
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

func (gs *goldsmith) exportFile(f *file) error {
	defer atomic.AddInt64(&gs.active, -1)

	absPath := filepath.Join(gs.dstDir, f.path)
	if err := f.export(absPath); err != nil {
		return err
	}

	gs.referenceFile(f.path)
	return nil
}

func (gs *goldsmith) referenceFile(path string) {
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

func (gs *goldsmith) fault(s *stage, f *file, err error) {
	log.Printf("%s\t%s\t%s", s.name, f.path, err)
	gs.tainted = true
}

//
//	Goldsmith Implementation
//

func (gs *goldsmith) Chain(p Plugin) Goldsmith {
	s := newStage(gs)
	go s.chain(p)
	return gs
}

func (gs *goldsmith) Complete() bool {
	s := gs.stages[len(gs.stages)-1]
	for f := range s.output {
		gs.exportFile(f)
	}

	gs.cleanupFiles()
	return gs.tainted
}
