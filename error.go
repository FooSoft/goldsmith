package goldsmith

import "fmt"

// Error wraps the core error type to provide a plugin or filter name in
// addition to the file path that was being processed at the time.
type Error struct {
	Name string
	Path string
	Err  error
}

// Error returns a string representation of the error.
func (err Error) Error() string {
	var path string
	if len(err.Path) > 0 {
		path = "@" + err.Path
	}

	return fmt.Sprintf("[%s%s]: %s", err.Name, path, err.Err.Error())
}
