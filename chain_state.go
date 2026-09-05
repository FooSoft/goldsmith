package goldsmith

import (
	"fmt"
	"sync"
)

type chainState struct {
	contexts []*Context

	cache   *cache
	filters filterStack
	index   int

	errors []error
	mutex  sync.Mutex
}

func (self *chainState) fault(name string, file *File, err error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	var faultError error
	if file == nil {
		faultError = fmt.Errorf("[%s]: %w", name, err)
	} else {
		faultError = fmt.Errorf("[%s@%s]: %w", name, file.Path(), err)
	}

	self.errors = append(self.errors, faultError)
}
