package query

import "github.com/arcology-network/streamer/actor"

type QueryContinuation interface {
	QueryContinuationAction(ctx *actor.ActionContext) error
}

type Step interface {
	Start(ctx *QueryContext, cont Continuation)
}

func StartStep(ctx *QueryContext, step Step, cont Continuation) {
	// ⭐ ① 启动前检查
	if ctx.finished {
		return
	}

	if ctx.Observer != nil {
		ctx.Observer.OnStepStart(ctx, step)
	}

	step.Start(ctx, func(resp interface{}, err error) {
		// ⭐ ② 回调时再检查（异步场景非常重要）
		if ctx.finished {
			return
		}

		if ctx.Observer != nil {
			ctx.Observer.OnStepFinish(ctx, step, err)
		}
		cont(resp, err)
	})
}
