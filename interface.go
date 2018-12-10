package goldsmith

type Initializer interface {
	Initialize(context *Context) (Filter, error)
}

type Processor interface {
	Process(context *Context, file *File) error
}

type Finalizer interface {
	Finalize(context *Context) error
}

type Component interface {
	Name() string
}

type Filter interface {
	Component
	Accept(context *Context, file *File) (bool, error)
}

type Plugin interface {
	Component
}
