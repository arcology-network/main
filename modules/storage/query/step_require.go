package query

type RequireStep struct {
	From  Step
	Check func(v interface{}) error
}

func (r *RequireStep) Start(ctx *QueryContext, cont Continuation) {
	StartStep(ctx, r.From, func(v interface{}, err error) {
		if err != nil {
			cont(nil, err)
			return
		}
		if e := r.Check(v); e != nil {
			cont(nil, e)
			return
		}
		cont(v, nil)
	})
}
