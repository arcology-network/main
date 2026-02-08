package query

type NoopStep struct{}

func (s NoopStep) Start(ctx *QueryContext, cont Continuation) {
	cont(nil, nil)
}
