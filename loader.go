package goldsmith

import "path/filepath"

type loader struct{}

func (*loader) Name() string {
	return "loader"
}

func (*loader) Initialize(context *Context) error {
	scannedInfo := make(chan fileInfo)
	go scanDir(context.goldsmith.sourceDir, scannedInfo)

	for info := range scannedInfo {
		if info.IsDir() {
			continue
		}

		relPath, _ := filepath.Rel(context.goldsmith.sourceDir, info.path)
		file, err := context.CreateFileFromAsset(relPath, info.path)
		if err != nil {
			return err
		}

		context.DispatchFile(file)
	}

	return nil
}
