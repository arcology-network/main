package query

type IfStep struct {
	Cond func(ctx *QueryContext) bool
	Then Step
	Else Step // 可为 nil
}

func (s *IfStep) Start(
	ctx *QueryContext,
	cont Continuation,
) {
	if s.Cond(ctx) {
		if s.Then != nil {
			StartStep(ctx, s.Then, cont)
			return
		}
	} else {
		if s.Else != nil {
			StartStep(ctx, s.Else, cont)
			return
		}
	}
	cont(nil, nil)
}
