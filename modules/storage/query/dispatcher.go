package query

import "github.com/arcology-network/streamer/actor"

type Dispatcher struct {
	plans map[string]QueryPlan
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		plans: make(map[string]QueryPlan),
	}
}

func (d *Dispatcher) Register(name string, plan QueryPlan) {
	d.plans[name] = plan
}

func (d *Dispatcher) GetQueryPlan(name string) QueryPlan {
	return d.plans[name]
}
func (d *Dispatcher) Query(
	name string,
	req interface{},
	execCtx *actor.ExecutionContext,
	obs StepObserver,
	scheduler Scheduler,
	cont Continuation,
) {
	plan := d.plans[name]
	if plan == nil {
		cont(nil, ErrPlanNotFound)
		return
	}

	// ⭐ finalCont 只在这里注入一次
	ctx := NewQueryContext(req, execCtx, obs, scheduler, func(resp interface{}, err error) {
		// 清理 runtime 状态
		// execCtx.SetLocal("query_ctx", nil)
		cont(resp, err)
	})

	// 让业务 / RPC / SubPlan 能拿到当前 QueryContext
	// execCtx.SetLocal("query_ctx", ctx)

	// ctx.Ctx.StartCasecade()

	// ⚠️ 这里只是“启动”，不是“结束”
	plan.Start(ctx, func(resp interface{}, err error) {
		// ⭐ 正常跑完，但没有提前 Return
		if ctx.finished {
			return
		}
		ctx.Return(resp, err)
	})
}
