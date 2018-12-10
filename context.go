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

func (context *Context) DispatchFile(file *File) {
	context.outputFiles <- file
}

func (context *Context) DispatchAndCacheFile(outputFile *File, inputFiles ...*File) {
	context.goldsmith.storeFile(context, outputFile, inputFiles)
	context.outputFiles <- outputFile
}

func (context *Context) RetrieveCachedFile(outputPath string, inputFiles ...*File) *File {
	return context.goldsmith.retrieveFile(context, outputPath, inputFiles)
}

func (context *Context) step() {
	defer close(context.outputFiles)

	var err error
	var filter Filter
	if initializer, ok := context.plugin.(Initializer); ok {
		filter, err = initializer.Initialize(context)
		if err != nil {
			context.goldsmith.fault(context.plugin.Name(), nil, err)
			return
		}
	}

	if context.inputFiles != nil {
		processor, _ := context.plugin.(Processor)

		var wg sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for inputFile := range context.inputFiles {
					accept := processor != nil

					var fileFilters []Filter
					fileFilters = append(fileFilters, context.fileFilters...)
					if filter != nil {
						fileFilters = append(fileFilters, filter)
					}

					for _, fileFilter := range fileFilters {
						if accept, err = fileFilter.Accept(context, inputFile); err != nil {
							context.goldsmith.fault(fileFilter.Name(), inputFile, err)
							return
						}
						if !accept {
							break
						}
					}

					if accept {
						if _, err := inputFile.Seek(0, os.SEEK_SET); err != nil {
							context.goldsmith.fault("core", inputFile, err)
						}
						if err := processor.Process(context, inputFile); err != nil {
							context.goldsmith.fault(context.plugin.Name(), inputFile, err)
						}
					} else {
						context.outputFiles <- inputFile
					}
				}
			}()
		}
		wg.Wait()
	}

	if finalizer, ok := context.plugin.(Finalizer); ok {
		if err := finalizer.Finalize(context); err != nil {
			context.goldsmith.fault(context.plugin.Name(), nil, err)
		}
	}
}
