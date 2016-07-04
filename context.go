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

	"github.com/bmatcuk/doublestar"
)

type context struct {
	gs            *goldsmith
	plug          Plugin
	filters       []string
	input, output chan *file
}

func (ctx *context) chain() {
	defer close(ctx.output)

	initializer, _ := ctx.plug.(Initializer)
	accepter, _ := ctx.plug.(Accepter)
	processor, _ := ctx.plug.(Processor)
	finalizer, _ := ctx.plug.(Finalizer)

	if initializer != nil {
		if err := initializer.Initialize(ctx); err != nil {
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
					accept := processor != nil && (accepter == nil || accepter.Accept(ctx, f))
					if accept && len(ctx.filters) > 0 {
						accept = false
						for _, filter := range ctx.filters {
							if match, _ := doublestar.PathMatch(filter, f.Path()); match {
								accept = true
								break
							}
						}
					}

					if accept {
						if _, err := f.Seek(0, os.SEEK_SET); err != nil {
							ctx.gs.fault(f, err)
						}
						if err := processor.Process(ctx, f); err != nil {
							ctx.gs.fault(f, err)
						}
					} else {
						ctx.output <- f
					}
				}
			}()
		}
		wg.Wait()
	}

	if finalizer != nil {
		if err := finalizer.Finalize(ctx); err != nil {
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
