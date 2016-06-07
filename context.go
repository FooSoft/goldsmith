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
	"runtime"
	"sync"
)

type context struct {
	gs            *goldsmith
	plug          Plugin
	input, output chan *file
}

func (ctx *context) chain() {
	defer close(ctx.output)

	init, _ := ctx.plug.(Initializer)
	accept, _ := ctx.plug.(Accepter)
	proc, _ := ctx.plug.(Processor)
	fin, _ := ctx.plug.(Finalizer)

	if init != nil {
		if err := init.Initialize(ctx); err != nil {
			ctx.gs.fault(nil, err)
			return
		}
	}

	if ctx.input != nil {
		var wg sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for f := range ctx.input {
					if proc == nil || accept != nil && !accept.Accept(ctx, f) {
						ctx.output <- f
					} else {
						if _, err := f.Seek(0, os.SEEK_SET); err != nil {
							ctx.gs.fault(f, err)
						}
						if err := proc.Process(ctx, f); err != nil {
							ctx.gs.fault(f, err)
						}
					}
				}
			}()
		}
		wg.Wait()
	}

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

func (ctx *context) SrcDir() string {
	return ctx.gs.srcDir
}

func (ctx *context) DstDir() string {
	return ctx.gs.dstDir
}
