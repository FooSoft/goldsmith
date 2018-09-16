package goldsmith

import "path/filepath"

type loader struct{}

func (*loader) Name() string {
	return "loader"
}

func (*loader) Initialize(ctx Context) ([]Filter, error) {
	infos := make(chan fileInfo)
	go scanDir(ctx.SrcDir(), infos)

	for info := range infos {
		if info.IsDir() {
			continue
		}

		relPath, _ := filepath.Rel(ctx.SrcDir(), info.path)

		f := &file{
			path:    relPath,
			Meta:    make(map[string]interface{}),
			modTime: info.ModTime(),
			size:    info.Size(),
			asset:   info.path,
		}

		ctx.DispatchFile(f)
	}

	return nil, nil
}
