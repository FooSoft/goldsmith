package goldsmith

import "fmt"

type Error struct {
	Name string
	Path string
	Err  error
}

func (e Error) Error() string {
	var path string
	if len(e.Path) > 0 {
		path = "@" + e.Path
	}

	return fmt.Sprintf("[%s%s]: %s", e.Name, path, e.Err.Error())
}
