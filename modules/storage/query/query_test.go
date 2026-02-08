package query

import (
	"fmt"

	"errors"
	"testing"
)

func TestQueryPlan_ParallelMergeWithRetry(t *testing.T) {
	scheduler := &MockScheduler{}
	observer := &MockObserver{}

	// ---------- Step A：第一次失败，第二次成功 ----------
	aAttempts := 0
	stepA := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			aAttempts++
			if aAttempts == 1 {
				cont(nil, errors.New("fail once"))
				return
			}
			cont("A_OK", nil)
		},
	}

	retryA := &RetryStep{
		Inner:    stepA,
		MaxRetry: 1,
	}

	// ---------- Step B：直接成功 ----------
	stepB := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			cont("B_OK", nil)
		},
	}

	valA := &ValueStep{Inner: retryA}
	valB := &ValueStep{Inner: stepB}

	parallel := &ParallelStep{
		Steps: []Step{valA, valB},
	}

	merge := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			fmt.Printf("merge get A=%v B=%v\n", ctx.Get(valA), ctx.Get(valB))
			a := ctx.Get(valA)
			b := ctx.Get(valB)
			cont(a.(string)+"+"+b.(string), nil)
		},
	}

	plan := &SimplePlan{
		Root: &SequentialStep{
			Steps: []Step{parallel, merge},
		},
	}

	// ---------- 执行 ----------
	var (
		result interface{}
		err    error
		called int
	)

	ctx := NewQueryContext(nil, nil, observer, scheduler,
		func(resp interface{}, err error) {
			t.Fatalf("ctx.Return should not be called in this test")
		})
	plan.Start(ctx, func(resp interface{}, e error) {
		called++
		result = resp
		err = e
	})

	// ---------- 验证 ----------
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "A_OK+B_OK" {
		t.Fatalf("unexpected result: %v", result)
	}

	if aAttempts != 2 {
		t.Fatalf("retry not triggered, attempts=%d", aAttempts)
	}

	if called != 1 {
		t.Fatalf("continuation called %d times", called)
	}
}

func TestTimeoutStep(t *testing.T) {
	scheduler := &MockScheduler{}

	step := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			// 永远不回调
		},
	}

	timeout := &TimeoutStep{
		Inner:   step,
		Timeout: 10,
	}

	var (
		err    error
		called int
	)

	ctx := NewQueryContext(nil, nil, nil, scheduler,
		func(resp interface{}, err error) {
			t.Fatalf("ctx.Return should not be called in this test")
		})
	timeout.Start(ctx, func(_ interface{}, e error) {
		called++
		err = e
	})

	// 触发定时器
	scheduler.FireAll()

	if err != ErrTimeout {
		t.Fatalf("expected timeout error, got %v", err)
	}

	if called != 1 {
		t.Fatalf("continuation called %d times", called)
	}
}

func TestDispatcher_PlanNotFound(t *testing.T) {
	d := NewDispatcher()

	var err error
	d.Query(
		"not-exist",
		nil,
		nil,
		&MockObserver{},
		&MockScheduler{},
		func(_ interface{}, e error) {
			err = e
		},
	)

	if err != ErrPlanNotFound {
		t.Fatalf("expected ErrPlanNotFound, got %v", err)
	}
}
