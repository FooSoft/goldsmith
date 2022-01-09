package goldsmith

type filterEntry struct {
	filter Filter
	index  int
}

type filterStack []filterEntry

func (self *filterStack) accept(file *File) bool {
	for _, entry := range *self {
		if entry.index >= file.index && !entry.filter.Accept(file) {
			return false
		}
	}

	return true
}

func (self *filterStack) push(filter Filter, index int) {
	*self = append(*self, filterEntry{filter, index})
}

func (self *filterStack) pop() {
	count := len(*self)
	if count == 0 {
		panic("attempted to pop empty filter stack")
	}

	*self = (*self)[:count-1]
}
