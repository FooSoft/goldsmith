package goldsmith

type Initializer interface {
	Initialize(ctx *Context) ([]Filter, error)
}

type Processor interface {
	Process(ctx *Context, file *File) error
}

type Finalizer interface {
	Finalize(ctx *Context) error
}

type Component interface {
	Name() string
}

type Filter interface {
	Component
	Accept(ctx *Context, file *File) (bool, error)
}

type Plugin interface {
	Component
}
