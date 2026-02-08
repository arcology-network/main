package query

type ResultStep struct {
	Build func(ctx *QueryContext) interface{}
}

func (s *ResultStep) Start(ctx *QueryContext, cont Continuation) {
	cont(s.Build(ctx), nil)
}

// type SubPlanReturn struct {
// 	Value any
// }

type ReturnFromSubPlanStep struct {
	Cond  func(ctx *QueryContext) bool
	Value func(ctx *QueryContext) any
	Err   error
}

func (s *ReturnFromSubPlanStep) Start(
	ctx *QueryContext,
	cont Continuation, // ⚠️ 注意：这里不再使用 cont
) {
	// Cond 不满足，也是一种“明确返回”
	if s.Cond != nil && !s.Cond(ctx) {
		// ctx.Return(nil, s.Err)
		cont(nil, nil)
		return
	}

	var v any
	if s.Value != nil {
		v = s.Value(ctx)
	}

	ctx.Return(v, s.Err)
}

type CallStep struct {
	Plan Step
	Bind func(ctx *QueryContext, v any)
}

// func (s *CallStep) Start(
// 	ctx *QueryContext,
// 	cont Continuation,
// ) {
// 	// subCtx := ctx.Fork()

// 	StartStep(ctx, s.Plan, func(v any, err error) {
// 		if err != nil {
// 			cont(nil, err)
// 			return
// 		}

//			// 子 plan 正常 or 提前 Return 的值
//			if s.Bind != nil {
//				s.Bind(ctx, v)
//			}
//			cont(nil, nil)
//		})
//	}
func (s *CallStep) Start(
	ctx *QueryContext,
	cont Continuation,
) {
	// ⭐ 必须 fork：Call 是一个子计划边界
	subCtx := ctx.Fork()

	// ⭐ 接管子计划的 Return
	subCtx.OnReturn(func(v any, err error) {
		if err != nil {
			cont(nil, err)
			return
		}

		if s.Bind != nil {
			s.Bind(ctx, v) // 注意：Bind 到父 ctx
		}

		cont(nil, nil)
	})

	// ⭐ 启动子计划
	StartStep(subCtx, s.Plan, func(v any, err error) {
		if err != nil {
			cont(nil, err)
			return
		}

		// ⚠️ 如果已经 Return 过，这里不能再处理
		if subCtx.IsReturned() {
			return
		}

		if s.Bind != nil {
			s.Bind(ctx, v)
		}

		cont(nil, nil)
	})
}
