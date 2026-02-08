package query

type ParallelStep struct {
	Steps []Step
}

func (p *ParallelStep) Start(ctx *QueryContext, cont Continuation) {
	if len(p.Steps) == 0 {
		cont(nil, nil)
		return
	}

	remain := len(p.Steps)
	done := false

	for _, step := range p.Steps {
		StartStep(ctx, step, func(_ interface{}, err error) {
			if done {
				return
			}
			if err != nil {
				done = true
				cont(nil, err)
				return
			}
			remain--
			if remain == 0 {
				done = true
				cont(nil, nil)
			}
		})
	}
}
