package goldsmith

import (
	"bytes"
	"errors"
	"os"
	"runtime"
	"sync"
	"time"
)

type Context struct {
	goldsmith *Goldsmith

	plugin Plugin
	hash   uint32

	fileFilters []Filter
	inputFiles  chan *File
	outputFiles chan *File
}

func (*Context) CreateFileFromData(sourcePath string, data []byte) *File {
	return &File{
		sourcePath: sourcePath,
		Meta:       make(map[string]interface{}),
		reader:     bytes.NewReader(data),
		size:       int64(len(data)),
		modTime:    time.Now(),
	}
}

func (*Context) CreateFileFromAsset(sourcePath, dataPath string) (*File, error) {
	info, err := os.Stat(dataPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("assets must be files")
	}

	file := &File{
		sourcePath: sourcePath,
		dataPath:   dataPath,
		Meta:       make(map[string]interface{}),
		size:       info.Size(),
		modTime:    info.ModTime(),
	}

	return file, nil
}

func (ctx *Context) DispatchFile(file *File) {
	ctx.outputFiles <- file
}

func (ctx *Context) DispatchAndCacheFile(outputFile *File, inputFiles ...*File) {
	ctx.goldsmith.storeFile(ctx, outputFile, inputFiles)
	ctx.outputFiles <- outputFile
}

func (ctx *Context) RetrieveCachedFile(outputPath string, inputFiles ...*File) *File {
	return ctx.goldsmith.retrieveFile(ctx, outputPath, inputFiles)
}

func (ctx *Context) step() {
	defer close(ctx.outputFiles)

	var err error
	var filters []Filter
	if initializer, ok := ctx.plugin.(Initializer); ok {
		filters, err = initializer.Initialize(ctx)
		if err != nil {
			ctx.goldsmith.fault(ctx.plugin.Name(), nil, err)
			return
		}
	}

	if ctx.inputFiles != nil {
		processor, _ := ctx.plugin.(Processor)

		var wg sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for inputFile := range ctx.inputFiles {
					accept := processor != nil
					for _, filter := range append(ctx.fileFilters, filters...) {
						if accept, err = filter.Accept(ctx, inputFile); err != nil {
							ctx.goldsmith.fault(filter.Name(), inputFile, err)
							return
						}
						if !accept {
							break
						}
					}

					if accept {
						if _, err := inputFile.Seek(0, os.SEEK_SET); err != nil {
							ctx.goldsmith.fault("core", inputFile, err)
						}
						if err := processor.Process(ctx, inputFile); err != nil {
							ctx.goldsmith.fault(ctx.plugin.Name(), inputFile, err)
						}
					} else {
						ctx.outputFiles <- inputFile
					}
				}
			}()
		}
		wg.Wait()
	}

	if finalizer, ok := ctx.plugin.(Finalizer); ok {
		if err := finalizer.Finalize(ctx); err != nil {
			ctx.goldsmith.fault(ctx.plugin.Name(), nil, err)
		}
	}
}
