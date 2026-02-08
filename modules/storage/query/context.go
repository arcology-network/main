package query

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/arcology-network/streamer/actor"
)

var contRoots sync.Map
var globalContID atomic.Uint64

type Continuation func(resp interface{}, err error)

type QueryContext struct {
	// ---------- 输入 ----------
	Req interface{}

	// ---------- 执行控制 ----------
	mu       sync.Mutex
	finished bool // 是否已经 Return
	failed   bool // 是否已经 Fail

	// Plan 级最终 continuation（return 用）
	finalCont Continuation

	// ---------- Step 结果 ----------
	Values map[Step]interface{}

	// ---------- 逻辑变量 ----------
	Vars map[string]interface{}

	// ---------- 执行环境 ----------
	Observer  StepObserver
	Scheduler Scheduler
	Ctx       *actor.ExecutionContext

	// ---------- RPC Continuations ----------
	continuations map[string]Continuation
	contID        uint64

	onReturn   Continuation
	returnOnce sync.Once
	Root       *QueryContext
}

func NewQueryContext(
	req interface{},
	execCtx *actor.ExecutionContext,
	obs StepObserver,
	scheduler Scheduler,
	finalCont Continuation,
) *QueryContext {
	ctx := &QueryContext{
		Req:           req,
		Values:        make(map[Step]interface{}),
		Observer:      obs,
		Scheduler:     scheduler,
		Ctx:           execCtx,
		continuations: make(map[string]Continuation),
		Vars:          make(map[string]interface{}),
		finalCont:     finalCont,
	}
	ctx.Root = ctx
	return ctx
}

func (ctx *QueryContext) OnReturn(cont Continuation) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	// 只能注册一次（子计划只应该有一个 Return 接收者）
	ctx.onReturn = cont
}

func (ctx *QueryContext) IsReturned() bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.finished
}

func (ctx *QueryContext) Return(v any, err error) {
	if ctx.onReturn == nil && ctx.finalCont == nil {
		panic("Return called but no continuation registered")
	}
	var cont Continuation
	ctx.returnOnce.Do(func() {
		ctx.mu.Lock()
		ctx.finished = true

		if ctx.onReturn != nil {
			cont = ctx.onReturn
		} else {
			cont = ctx.finalCont
		}
		ctx.mu.Unlock()
	})

	if cont != nil {
		cont(v, err)
	}
}

func (ctx *QueryContext) Fail(err error) {
	ctx.mu.Lock()
	if ctx.failed || ctx.finished {
		ctx.mu.Unlock()
		return
	}
	ctx.failed = true
	final := ctx.finalCont
	ctx.mu.Unlock()

	final(nil, err)
}

func (ctx *QueryContext) RegisterCont(cont Continuation) string {
	root := ctx.Root
	root.mu.Lock()
	defer root.mu.Unlock()

	// root.contID++
	root.contID = globalContID.Add(1)
	name := fmt.Sprintf("_query_cont_%d", root.contID)
	root.continuations[name] = cont

	log.Printf("[CONT] register -- contId:%s root:%s", name, fmt.Sprintf("%p", ctx.Root))

	contRoots.Store(name, root)

	return name
}

func TakeContinuationById(id string) (Continuation, bool) {
	v, ok := contRoots.Load(id)
	if !ok {
		return nil, false
	}
	root := v.(*QueryContext)

	cont, ok := root.TakeContinuation(id)
	if ok {
		contRoots.Delete(id)
	}
	return cont, ok
}

func (ctx *QueryContext) TakeContinuation(id string) (Continuation, bool) {
	root := ctx.Root
	root.mu.Lock()
	defer root.mu.Unlock()

	cont, ok := root.continuations[id]
	if !ok {
		return nil, false
	}
	delete(root.continuations, id)
	return cont, true
}

func (ctx *QueryContext) Fork() *QueryContext {
	n := &QueryContext{
		Req:       ctx.Req,
		Ctx:       ctx.Ctx,
		Observer:  ctx.Observer,
		Scheduler: ctx.Scheduler,
		finalCont: ctx.finalCont,

		Vars:          make(map[string]interface{}),
		Values:        make(map[Step]interface{}),
		continuations: make(map[string]Continuation),

		Root: ctx.Root,
	}

	if n.Root == nil {
		n.Root = ctx
	}

	for k, v := range ctx.Vars {
		n.Vars[k] = v
	}

	return n
}

func (ctx *QueryContext) Set(step Step, val interface{}) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.Values[step] = val
}

func (ctx *QueryContext) Get(step Step) interface{} {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.Values[step]
}
