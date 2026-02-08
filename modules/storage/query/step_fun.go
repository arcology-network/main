package query

type FuncStep struct {
	Do func(ctx *QueryContext, cont Continuation)
}

func (f *FuncStep) Start(ctx *QueryContext, cont Continuation) {
	f.Do(ctx, cont)
}
