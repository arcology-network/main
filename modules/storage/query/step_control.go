package query

// ---------- Timeout ----------
type TimeoutStep struct {
	Inner   Step
	Timeout uint64 // 逻辑时间 / tick
}

func (t *TimeoutStep) Start(ctx *QueryContext, cont Continuation) {
	done := false

	// 超时回调
	ctx.Scheduler.ScheduleAfter(t.Timeout, func() {
		if done || ctx.finished {
			return
		}
		done = true
		cont(nil, ErrTimeout)
	})

	// 正常执行 Inner
	StartStep(ctx, t.Inner, func(resp interface{}, err error) {
		if done || ctx.finished {
			return
		}
		done = true
		cont(resp, err)
	})
}

// ---------- Retry ----------

type RetryStep struct {
	Inner    Step
	MaxRetry int
}

func (r *RetryStep) Start(ctx *QueryContext, cont Continuation) {
	attempt := 0

	var run func()
	run = func() {
		StartStep(ctx, r.Inner, func(resp interface{}, err error) {
			if err == nil {
				cont(resp, nil)
				return
			}
			attempt++
			if attempt > r.MaxRetry {
				cont(nil, err)
				return
			}
			run()
		})
	}

	run()
}

// ---------- Fallback ----------

type FallbackStep struct {
	Primary Step
	Backup  Step
}

func (f *FallbackStep) Start(ctx *QueryContext, cont Continuation) {
	StartStep(ctx, f.Primary, func(resp interface{}, err error) {
		if err == nil {
			cont(resp, nil)
			return
		}
		StartStep(ctx, f.Backup, cont)
	})
}
