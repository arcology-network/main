package query

type MapStep struct {
	From Step
	Map  func(v interface{}) (interface{}, error)
}

func (s *MapStep) Start(ctx *QueryContext, cont Continuation) {
	StartStep(ctx, s.From, func(v interface{}, err error) {
		if err != nil {
			cont(nil, err)
			return
		}
		out, err := s.Map(v)
		cont(out, err)
	})
}
