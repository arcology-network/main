package query

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"testing"
	"time"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/actor/rpc"
	"github.com/arcology-network/streamer/broker"
	"github.com/arcology-network/streamer/logger"
)

type BusinessA struct {
	dispatcher *Dispatcher
	scheduler  *MockScheduler
	observer   *MockObserver
}

func NewBusinessA() *BusinessA {
	a := &BusinessA{
		dispatcher: NewDispatcher(),
		scheduler:  &MockScheduler{},
		observer:   &MockObserver{},
	}

	// 注册 QueryPlan
	a.dispatcher.Register(
		"GetBlock",
		&GetBlockQueryPlan{},
	)

	a.dispatcher.Register(
		"latestHeight",
		&InlineQueryPlan{
			fn: func(ctx *QueryContext) (interface{}, error) {
				// return &mtypes.QueryResult{
				// 	Data: 30,
				// }, nil
				return 30, nil
			},
		},
	)

	a.dispatcher.Register(
		"GetNonce",
		&GetNonceQueryPlan{},
	)

	a.dispatcher.Register(
		"GetItems",
		&ReceiptQueryPlan{},
	)

	return a
}

func (a *BusinessA) Inputs() ([]string, bool) {
	return []string{}, false
}

func (a *BusinessA) Outputs() map[string]int {
	return map[string]int{}
}
func (a *BusinessA) RpcConfig() (string, int) {
	return "BusinessA", 20
}

// Action
func (a *BusinessA) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Query", a.Query)
	reg.Register("QueryContinuationAction", a.QueryContinuationAction)
}

func (a *BusinessA) Query(ctx *actor.ActionContext) error {
	req := ctx.RPC.Request.(*mtypes.QueryRequest)

	plan := a.dispatcher.GetQueryPlan(req.QueryType)
	if plan == nil {
		ctx.ExecCtx.SendRpcResponse(ErrPlanNotFound.Error(), nil)
		return nil
	}

	a.dispatcher.Query(
		req.QueryType,
		req,
		ctx.ExecCtx,
		a.observer,
		a.scheduler,
		func(resp interface{}, err error) {
			// ctx.ExecCtx.EndCasecade()
			fmt.Printf("***********BusinessA.Query err:%v\n", err)
			if err != nil {
				ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
				return
			}
			ctx.ExecCtx.SendRpcResponse("", resp)
		},
	)

	return nil
}

// func (a *BusinessA) QueryContinuationAction(ctx *actor.ActionContext) error {
// 	logger.Log.Debug(
// 		context.Background(), "BusinessA",
// 		"[QueryContinuationAction] called",
// 		logger.F("rpc_req", ctx.RPC.Request),
// 		logger.F("messages_len", len(ctx.Messages)),
// 	)
// 	if len(ctx.Messages) == 0 {
// 		logger.Log.Debug(context.Background(), "BusinessA", "no messages")
// 		return nil
// 	}
// 	msg := ctx.Messages[0]
// 	logger.Log.Debug(
// 		context.Background(), "BusinessA",
// 		"[QueryContinuationAction] message",
// 		logger.F("next_step", msg.NextStep),
// 		logger.F("payload", msg.Data),
// 	)
// 	logger.Log.Debug(context.Background(), "BusinessA", fmt.Sprintf("req:%v", ctx.RPC.Request))
// 	qctxAny := ctx.ExecCtx.GetLocal("query_ctx")
// 	if qctxAny == nil {
// 		logger.Log.Debug(context.Background(), "BusinessA", "query_ctx not found")
// 		return nil
// 	}
// 	qctx := qctxAny.(*QueryContext)

// 	cont, ok := qctx.TakeContinuation(msg.ContId)
// 	if !ok {
// 		logger.Log.Debug(context.Background(), "BusinessA", "continuation not found", logger.F("contId", msg.ContId))
// 		return nil
// 	}
// 	logger.Log.Debug(context.Background(), "BusinessA", "continuation hit")
// 	cont(ctx.RPC.Request, nil)
// 	return nil
// }

func (a *BusinessA) QueryContinuationAction(ctx *actor.ActionContext) error {
	if len(ctx.Messages) == 0 {
		ctx.ExecCtx.LogErr("no messages")
		return nil
	}

	msg := ctx.Messages[0]

	log.Printf("[CONT] action enter -- contId:%s", msg.ContId)

	cont, ok := TakeContinuationById(msg.ContId)
	log.Printf("[CONT] take -- contId:%s ok:%v", msg.ContId, ok)

	if !ok {
		ctx.ExecCtx.LogErr("continuation not found", logger.F("contId", msg.ContId))
		return nil
	}

	cont(ctx.RPC.Request, nil)
	log.Printf("[CONT] take -- contId:%s cont END ", msg.ContId)
	return nil
}

type InlineQueryPlan struct {
	fn func(*QueryContext) (interface{}, error)
}

func (p *InlineQueryPlan) Start(ctx *QueryContext, cont Continuation) {
	resp, err := p.fn(ctx)
	cont(resp, err)
}

type BusinessB struct {
}

func NewBusinessB() *BusinessB {
	a := &BusinessB{}
	return a
}

func (a *BusinessB) Inputs() ([]string, bool) {
	return []string{}, false
}

func (a *BusinessB) Outputs() map[string]int {
	return map[string]int{}
}
func (a *BusinessB) RpcConfig() (string, int) {
	return "BusinessB", 20
}

// Action
func (a *BusinessB) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("GetBlockHeight", a.GetBlockHeight)
	reg.Register("GetBlock", a.GetBlock)
	reg.Register("GetNonce", a.GetNonce)
}

func (a *BusinessB) GetBlockHeight(ctx *actor.ActionContext) error {
	logger.Log.Debug(context.Background(), "BusinessB", "BusinessB.GetBlockHeight", logger.F("params", ctx.RPC.Request))
	ctx.ExecCtx.SendRpcResponse("", 3)
	return nil
}

func (a *BusinessB) GetBlock(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", ctx.RPC.Request.(int)*7)
	return nil
}
func (a *BusinessB) GetNonce(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", 7)
	return nil
}

type GetBlockQueryPlan struct {
	// 如果你后面需要参数化，可以放字段
}

func (p *GetBlockQueryPlan) Start(
	ctx *QueryContext,
	cont Continuation,
) {
	root := p.buildSteps()
	StartStep(ctx, root, cont)
}
func (p *GetBlockQueryPlan) buildSteps() Step {
	// Step 1: hash -> height
	getHeight := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			req := ctx.Req.(*mtypes.QueryRequest)
			ctx.Ctx.InvokeRPC(
				"BusinessB",
				"GetBlockHeight",
				// &mtypes.GetBlockHeightReq{
				// 	Hash: req.Data,
				// },
				req.Data.(int),
				"QueryContinuationAction",
				actor.NMeta("contId", ctx.RegisterCont(cont)),
			)
		},
	}

	valHeight := &ValueStep{Inner: getHeight}

	// Step 2: height -> block
	getBlock := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			height := ctx.Get(valHeight).(int)

			ctx.Ctx.InvokeRPC(
				"BusinessB",
				"GetBlock",
				// &mtypes.GetBlockReq{
				// 	Height: height,
				// },
				height,
				"QueryContinuationAction",
				actor.NMeta("contId", ctx.RegisterCont(cont)),
			)
		},
	}

	valBlock := &ValueStep{Inner: getBlock}

	// 顺序执行
	return &SequentialStep{
		Steps: []Step{
			valHeight,
			valBlock,
		},
	}
}

type GetNonceQueryPlan struct {
	// 如果你后面需要参数化，可以放字段
}

func (p *GetNonceQueryPlan) Start(
	ctx *QueryContext,
	cont Continuation,
) {
	ctx.Vars["nonceAdd"] = 5
	root := p.buildSteps()
	StartStep(ctx, root, cont)
}
func (p *GetNonceQueryPlan) buildSteps() Step {
	getNonce := &RpcStep{
		Call: func(ctx *QueryContext, cont Continuation) {
			// req := ctx.Req.(mtypes.RequestBalance)
			ctx.Ctx.InvokeRPC(
				"BusinessB",
				"GetNonce",
				0,
				"QueryContinuationAction",
				actor.NMeta("contId", ctx.RegisterCont(cont)),
			)
		},
	}

	valNonce := &ValueStep{Inner: getNonce}

	// return &SequentialStep{
	// 	Steps: []Step{
	// 		valNonce,
	// 	},
	// }

	// processNonce := &MapStep{
	// 	From: valNonce,
	// 	Map: func(v interface{}) (interface{}, error) {
	// 		nonce := v.(int)
	// 		return nonce + 1, nil // 示例加工
	// 	},
	// }

	process := &FuncStep{
		Do: func(ctx *QueryContext, cont Continuation) {
			nonce := ctx.Get(valNonce).(int)
			nonceAdd := ctx.Vars["nonceAdd"].(int)
			cont(nonce+nonceAdd, nil)
		},
	}

	requireNonce := &RequireStep{
		From: process,
		Check: func(v interface{}) error {
			if v.(int) == 12 {
				fmt.Printf("********************check*********\n")
				return errors.New("nonce add OK")
			}
			return nil
		},
	}

	return &SequentialStep{
		Steps: []Step{
			// processNonce,
			valNonce,
			requireNonce,
		},
	}
}

type ReceiptQueryPlan struct{}

func (p *ReceiptQueryPlan) Start(
	ctx *QueryContext,
	cont Continuation,
) {
	root := p.buildSteps()
	StartStep(ctx, root, cont)
}
func (p *ReceiptQueryPlan) buildSteps() Step {
	return &SequentialStep{
		Steps: []Step{
			&FuncStep{
				Do: func(ctx *QueryContext, cont Continuation) {
					req := ctx.Req.(*mtypes.QueryRequest).Data.([]int)
					ctx.Vars["nums"] = make([]int, 0, len(req))
					cont(nil, nil)
				},
			},

			&ForEachStep{
				Items: func(ctx *QueryContext) []interface{} {
					req := ctx.Req.(*mtypes.QueryRequest).Data.([]int)
					items := make([]interface{}, len(req))
					for i, h := range req {
						items[i] = h
					}
					return items
				},
				Body: func(item any) Step {
					return &SequentialStep{
						Steps: []Step{
							&FuncStep{
								Do: func(ctx *QueryContext, cont Continuation) {
									num := item.(int)
									ctx.Vars["val"] = num + 5
									cont(nil, nil) // ✅ 必须
								},
							},
						},
					}

				},
				Collect: func(parent *QueryContext, _ any, sub *QueryContext) {
					parent.Vars["nums"] = append(
						parent.Vars["nums"].([]int),
						sub.Vars["val"].(int),
					)
				},
			},

			// 3️⃣ 最终返回
			&ResultStep{
				Build: func(ctx *QueryContext) any {
					return ctx.Vars["nums"]
				},
			},
		},
	}
}

type BusinessC struct {
	sender actor.OutboundSender
	ret1   int
	ret2   int
}

func NewBusinessC() *BusinessC {
	a := &BusinessC{
		ret1: 0,
		ret2: 0,
	}
	return a
}

func (a *BusinessC) SetSender(sender actor.OutboundSender) {
	a.sender = sender
}

func (a *BusinessC) Inputs() ([]string, bool) {
	return []string{}, false
}

func (a *BusinessC) Outputs() map[string]int {
	return map[string]int{}
}
func (a *BusinessC) RegisterActions(reg actor.ActionRegistrar) {
}
func (a *BusinessC) send(ss *broker.StatefulStreamer) {
	//sync
	ret, _ := a.sender.SendSync("BusinessA", "Query", &mtypes.QueryRequest{
		QueryType: "GetBlock",
		Data:      2,
	}, 10)
	a.ret1 = ret.(int)
	logger.Log.Debug(context.Background(), "BusinessC", "GetBlock return", logger.F("ret", ret))
	time.Sleep(1 * time.Second)

	ret, _ = a.sender.SendSync("BusinessA", "Query", &mtypes.QueryRequest{
		QueryType: "latestHeight",
		Data:      2,
	}, 10)
	a.ret2 = ret.(int)
	logger.Log.Debug(context.Background(), "BusinessC", "latestHeight return", logger.F("ret", ret))
	time.Sleep(1 * time.Second)

	ret, err := a.sender.SendSync("BusinessA", "Query", &mtypes.QueryRequest{
		QueryType: "GetNonce",
		Data:      2,
	}, 10)

	logger.Log.Debug(context.Background(), "BusinessC", "GetNonce return", logger.F("ret", ret), logger.F("err", err.Error()))
	time.Sleep(1 * time.Second)

	ret, err = a.sender.SendSync("BusinessA", "Query", &mtypes.QueryRequest{
		QueryType: "GetItems",
		Data:      []int{10, 20},
	}, 10)
	logger.Log.Debug(context.Background(), "BusinessC", "GetItems return", logger.F("ret", ret))
	time.Sleep(1 * time.Second)
}

func TestRpcQuery(t *testing.T) {
	logger.InitLog("./log.toml", "")
	rpc.InitGlobalRPCFactory()
	ss := broker.NewStatefulStreamer()
	rpc.InitGlobalRPCClient(ss, 10, 60)

	concurrency := runtime.NumCPU()

	a := NewBusinessA()
	actor.CreateActor("BusinessA", ss, []actor.Business{a}, []string{"BusinessA"}, concurrency, []string{""})

	b := NewBusinessB()
	actor.CreateActor("BusinessB", ss, []actor.Business{b}, []string{"BusinessB"}, concurrency, []string{""})

	c := NewBusinessC()
	c.SetSender(actor.NewSendAdaptor(ss, rpc.GlobalRPCClient))
	actor.CreateActor("BusinessC", ss, []actor.Business{c}, []string{"BusinessC"}, concurrency, []string{""})

	ss.Serve()

	c.send(ss)

	if c.ret1 != 21 {
		t.Errorf("rpc query test err,a.ret:%v", c.ret1)
	}

	if c.ret2 != 30 {
		t.Errorf("rpc query test err,a.ret:%v", c.ret2)
	}
}
