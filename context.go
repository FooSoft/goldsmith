package goldsmith

import (
	"os"
	"runtime"
	"sync"
)

type context struct {
	gs            *goldsmith
	plug          Plugin
	filters       []Filter
	input, output chan *file
}

func (ctx *context) chain() {
	defer close(ctx.output)

	var err error
	var filters []Filter
	if initializer, ok := ctx.plug.(Initializer); ok {
		filters, err = initializer.Initialize(ctx)
		if err != nil {
			ctx.gs.fault(ctx.plug.Name(), nil, err)
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
					for _, filter := range append(ctx.filters, filters...) {
						if accept, err = filter.Accept(ctx, f); err != nil {
							ctx.gs.fault(filter.Name(), f, err)
							return
						}

						if !accept {
							break
						}
					}

					if accept {
						if _, err := f.Seek(0, os.SEEK_SET); err != nil {
							ctx.gs.fault("core", f, err)
						}
						if err := processor.Process(ctx, f); err != nil {
							ctx.gs.fault(ctx.plug.Name(), f, err)
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
			ctx.gs.fault(ctx.plug.Name(), nil, err)
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
