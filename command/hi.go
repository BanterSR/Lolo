package command

type hi struct {
	base
}

func (h *hi) Options() *Options {
	return &Options{}
}

func (h *hi) Handle(ctx *Context) {
	ctx.Response().String("hi")
}
