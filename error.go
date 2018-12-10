package goldsmith

import "fmt"

type Error struct {
	Name string
	Path string
	Err  error
}

func (err Error) Error() string {
	var path string
	if len(err.Path) > 0 {
		path = "@" + err.Path
	}

	return fmt.Sprintf("[%s%s]: %s", err.Name, path, err.Err.Error())
}
