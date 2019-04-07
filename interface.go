package goldsmith

// Plugin contains the minimum set of methods required on plugins. Plugins can
// also optionally implement Initializer, Processor, and Finalizer interfaces.
type Plugin interface {
	Name() string
}

// Initializer is used to optionally initialize a plugin and to specify a
// filter to be used for determining which files will be processed.
type Initializer interface {
	Initialize(context *Context) (Filter, error)
}

// Processor allows for optional processing of files passing through a plugin.
type Processor interface {
	Process(context *Context, file *File) error
}

// Finalizer allows for optional finalization of a plugin after all files
// queued in the chain have passed through it.
type Finalizer interface {
	Finalize(context *Context) error
}

// Filter is used to determine which files should continue in the chain.
type Filter interface {
	Name() string
	Accept(file *File) (bool, error)
}
