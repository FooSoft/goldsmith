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

	var filters []string
	if initializer, ok := ctx.plug.(Initializer); ok {
		var err error
		filters, err = initializer.Initialize(ctx)
		if err != nil {
			ctx.gs.fault(nil, err)
			return
		}
	}

	if ctx.input != nil {
		processor, _ := ctx.plug.(Processor)

		var wg sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for f := range ctx.input {
					accept := processor != nil
					matcher := func(patterns []string) {
						if accept && len(patterns) > 0 {
							accept = false
							for _, pattern := range patterns {
								if match, _ := doublestar.PathMatch(pattern, f.Path()); match {
									accept = true
									break
								}
							}
						}
					}

					matcher(filters)
					matcher(ctx.filters)

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

	if finalizer, ok := ctx.plug.(Finalizer); ok {
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
