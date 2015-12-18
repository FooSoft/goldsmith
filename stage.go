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
	"sync"
	"sync/atomic"
)

type stage struct {
	name          string
	gs            *goldsmith
	input, output chan *file
}

func newStage(gs *goldsmith) *stage {
	s := &stage{gs: gs, output: make(chan *file)}
	if len(gs.stages) > 0 {
		s.input = gs.stages[len(gs.stages)-1].output
	}

	gs.stages = append(gs.stages, s)
	return s
}

func (s *stage) chain(p Plugin) {
	defer close(s.output)
	s.name = p.Name()

	init, _ := p.(Initializer)
	accept, _ := p.(Accepter)
	proc, _ := p.(Processor)
	fin, _ := p.(Finalizer)

	var (
		wg    sync.WaitGroup
		mtx   sync.Mutex
		batch []File
	)

	dispatch := func(f *file) {
		if fin == nil {
			s.output <- f
		} else {
			mtx.Lock()
			batch = append(batch, f)
			mtx.Unlock()

			atomic.AddInt64(&s.gs.stalled, 1)
		}
	}

	if init != nil {
		if err := init.Initialize(s); err != nil {
			s.gs.fault(s, "Initialization", nil, err)
			return
		}
	}

	for f := range s.input {
		if accept != nil && !accept.Accept(f) {
			s.output <- f
		} else if proc == nil {
			dispatch(f)
		} else {
			wg.Add(1)
			go func(f *file) {
				defer wg.Done()
				f.rewind()
				keep, err := proc.Process(s, f)
				if err != nil {
					s.gs.fault(s, "Processing", f, err)
				} else if keep {
					dispatch(f)
				} else {
					atomic.AddInt64(&s.gs.active, -1)
				}
			}(f)
		}
	}

	wg.Wait()

	if fin != nil {
		if err := fin.Finalize(s, batch); err != nil {
			s.gs.fault(s, "Finalization", nil, err)
		}

		for _, f := range batch {
			atomic.AddInt64(&s.gs.stalled, -1)
			s.output <- f.(*file)
		}
	}
}

//
//	Context Implementation
//

func (s *stage) DispatchFile(f File) {
	atomic.AddInt64(&s.gs.active, 1)
	s.output <- f.(*file)
}

func (s *stage) ReferenceFile(path string) {
	s.gs.referenceFile(path)
}

func (s *stage) SrcDir() string {
	return s.gs.srcDir
}

func (s *stage) DstDir() string {
	return s.gs.dstDir
}
