package query

type ForEachStep struct {
	Items   func(ctx *QueryContext) []interface{}
	Body    func(item interface{}) Step
	Collect func(parent *QueryContext, item any, sub *QueryContext)
}

func (s *ForEachStep) Start(
	ctx *QueryContext,
	cont Continuation,
) {
	items := s.Items(ctx)
	if len(items) == 0 {
		cont(nil, nil)
		return
	}

	var run func(i int)
	run = func(i int) {
		if i >= len(items) {
			// 所有 item 执行完
			cont(nil, nil)
			return
		}

		item := items[i]

		// ⭐ 每个 item 使用独立子 ctx
		subCtx := ctx.Fork()

		// ⭐⭐⭐ 关键：监听 subCtx.Return
		subCtx.OnReturn(func(v any, err error) {
			if err != nil {
				cont(nil, err)
				return
			}

			if s.Collect != nil {
				s.Collect(ctx, item, subCtx)
			}

			run(i + 1)
		})

		step := s.Body(item)

		StartStep(subCtx, step, func(_ any, err error) {
			if err != nil {
				cont(nil, err)
				return
			}

			// ⚠️ 注意：如果已经 Return 过，这里应当 no-op
			if subCtx.IsReturned() {
				return
			}

			// ⭐⭐ 核心：合并结果
			if s.Collect != nil {
				s.Collect(ctx, item, subCtx)
			}

			// 继续下一个
			run(i + 1)
		})
	}
	run(0)
}
