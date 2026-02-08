package query

type SequentialStep struct {
	Steps []Step
}

func (s *SequentialStep) Start(ctx *QueryContext, cont Continuation) {
	if ctx.IsReturned() {
		return
	}

	if len(s.Steps) == 0 {
		cont(nil, nil)
		return
	}

	var run func(i int)
	run = func(i int) {
		if ctx.IsReturned() {
			return
		}

		if i >= len(s.Steps) {
			// 正常跑完所有 step（没有 Return）
			cont(nil, nil)
			return
		}

		StartStep(ctx, s.Steps[i], func(resp any, err error) {
			// ⭐ 统一在回调中检查 Return
			if ctx.IsReturned() {
				return
			}

			if err != nil {
				cont(nil, err)
				return
			}

			// ⭐ 最后一个 step，交出结果
			if i == len(s.Steps)-1 {
				cont(resp, nil)
				return
			}

			run(i + 1)
		})
	}

	run(0)
}

// func (s *SequentialStep) Start(ctx *QueryContext, cont Continuation) {
// 	if ctx.finished {
// 		return
// 	}

// 	if len(s.Steps) == 0 {
// 		cont(nil, nil)
// 		return
// 	}

// 	var run func(i int)
// 	run = func(i int) {
// 		if i >= len(s.Steps) {
// 			// ⚠️ 不应该走到这里
// 			cont(nil, nil)
// 			return
// 		}

// 		StartStep(ctx, s.Steps[i], func(resp interface{}, err error) {
// 			// ⭐ 异步回调时再检查
// 			if ctx.finished {
// 				return
// 			}

// 			if err != nil {
// 				cont(nil, err)
// 				return
// 			}

// 			// ⭐ 如果是最后一个 Step，把它的结果返回
// 			if i == len(s.Steps)-1 {
// 				cont(resp, nil)
// 				return
// 			}

// 			run(i + 1)
// 		})
// 	}

// 	run(0)
// }
