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
	"runtime"
	"sync"
)

type context struct {
	gs            *goldsmith
	input, output chan *file
}

func newContext(gs *goldsmith) *context {
	ctx := &context{gs: gs, output: make(chan *file)}
	if len(gs.contexts) > 0 {
		ctx.input = gs.contexts[len(gs.contexts)-1].output
	}

	gs.contexts = append(gs.contexts, ctx)
	return ctx
}

func (ctx *context) chain(p Plugin) {
	defer close(ctx.output)

	init, _ := p.(Initializer)
	accept, _ := p.(Accepter)
	proc, _ := p.(Processor)
	fin, _ := p.(Finalizer)

	if init != nil {
		if err := init.Initialize(ctx); err != nil {
			ctx.gs.fault(nil, err)
			return
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range ctx.input {
				if proc == nil || accept != nil && !accept.Accept(ctx, f) {
					ctx.output <- f
				} else {
					f.rewind()
					if err := proc.Process(ctx, f); err != nil {
						ctx.gs.fault(f, err)
					}
				}
			}
		}()
	}
	wg.Wait()

	if fin != nil {
		if err := fin.Finalize(ctx); err != nil {
			ctx.gs.fault(nil, err)
		}
	}
}

//
//	Context Implementation
//

func (ctx *context) DispatchFile(f File) {
	ctx.output <- f.(*file)
}

func (ctx *context) ReferenceFile(path string) {
	ctx.gs.referenceFile(path)
}

func (ctx *context) SrcDir() string {
	return ctx.gs.srcDir
}

func (ctx *context) DstDir() string {
	return ctx.gs.dstDir
}
