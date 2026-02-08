package query

type ValueStep struct {
	Inner Step
	Bind  func(ctx *QueryContext, v any)
}

func (v *ValueStep) Start(ctx *QueryContext, cont Continuation) {
	StartStep(ctx, v.Inner, func(resp interface{}, err error) {
		if err != nil {
			cont(nil, err)
			return
		}

		if v.Bind != nil {
			v.Bind(ctx, resp)
		}

		ctx.Set(v, resp)
		cont(resp, nil)
	})
}
