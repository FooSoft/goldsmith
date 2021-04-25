package goldsmith

import "path/filepath"

type loader struct{}

func (*loader) Name() string {
	return "loader"
}

func (*loader) Initialize(context *Context) error {
	infos := make(chan fileInfo)
	go scanDir(context.goldsmith.sourceDir, infos)

	for info := range infos {
		if info.IsDir() {
			continue
		}

		relPath, _ := filepath.Rel(context.goldsmith.sourceDir, info.path)

		file := &File{
			sourcePath: relPath,
			Meta:       make(map[string]interface{}),
			modTime:    info.ModTime(),
			size:       info.Size(),
			dataPath:   info.path,
		}

		context.DispatchFile(file)
	}

	return nil
}
